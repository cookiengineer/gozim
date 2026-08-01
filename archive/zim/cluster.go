package zim

import (
	"bytes"
	"fmt"
)

const (
	clusterOffsetSize32 = 4
	clusterOffsetSize64 = 8
)

// ClusterInfo is the first byte of a cluster, encoding compression type and flags.
type ClusterInfo struct {
	Compression Compression
	IsExtended  bool
}

// ParseClusterInfo decodes the cluster info byte.
// Low nibble (bits 0-3): compression type.
// Bit 4 (0x10): extended flag — if set, blob offsets are 8 bytes instead of 4.
func ParseClusterInfo(infoByte byte) ClusterInfo {
	return ClusterInfo{
		Compression: Compression(infoByte & 0x0F),
		IsExtended:  (infoByte & 0x10) != 0,
	}
}

// EncodeClusterInfo encodes compression type and extended flag into a cluster info byte.
func EncodeClusterInfo(info ClusterInfo) byte {
	result := byte(info.Compression) & 0x0F
	if info.IsExtended {
		result |= 0x10
	}
	return result
}

// Cluster represents a decompressed and parsed ZIM cluster.
// It holds the decompressed data and the blob boundaries within it.
type Cluster struct {
	compression Compression
	blobOffsets []uint64
	blobData    []byte
}

// ParseCluster parses a cluster from raw (possibly compressed) bytes.
func ParseCluster(clusterBytes []byte, factory *CompressorFactory) (*Cluster, error) {
	if len(clusterBytes) < 1 {
		return nil, fmt.Errorf("%w: cluster too short for info byte", ErrFormat)
	}

	info := ParseClusterInfo(clusterBytes[0])
	offsetSize := clusterOffsetSize32
	if info.IsExtended {
		offsetSize = clusterOffsetSize64
	}

	switch info.Compression {
	case CompressionNone:
		return parseUncompressedCluster(clusterBytes, info, offsetSize)
	case CompressionZstd, CompressionLzma:
		return parseCompressedCluster(clusterBytes, info, offsetSize, factory)
	default:
		return nil, fmt.Errorf("%w: compression type %d", ErrUnsupportedCompression, info.Compression)
	}
}

// parseUncompressedCluster parses a cluster where the offset table is stored
// directly in the raw bytes, followed by the uncompressed blob data.
// The first offset equals the size of the offset table (without the info byte).
func parseUncompressedCluster(clusterBytes []byte, info ClusterInfo, offsetSize int) (*Cluster, error) {
	if len(clusterBytes) < 1+offsetSize {
		return nil, fmt.Errorf("%w: cluster too short for offset table", ErrFormat)
	}

	var firstOffset uint64
	if info.IsExtended {
		firstOffset = readUint64(clusterBytes[1:])
	} else {
		firstOffset = uint64(readUint32(clusterBytes[1:]))
	}

	numOffsets := int(firstOffset) / offsetSize
	if numOffsets < 2 {
		return nil, fmt.Errorf("%w: cluster requires at least 2 offset entries (first offset=%d, offsetSize=%d)", ErrFormat, firstOffset, offsetSize)
	}

	blobOffsets := make([]uint64, numOffsets)
	for i := 0; i < numOffsets; i++ {
		pos := 1 + i*offsetSize
		if info.IsExtended {
			blobOffsets[i] = readUint64(clusterBytes[pos:]) - firstOffset
		} else {
			blobOffsets[i] = uint64(readUint32(clusterBytes[pos:])) - firstOffset
		}
	}

	if 1+int(firstOffset) > len(clusterBytes) {
		return nil, fmt.Errorf("%w: offset table exceeds cluster size", ErrFormat)
	}

	blobData := clusterBytes[1+firstOffset:]

	return &Cluster{
		compression: info.Compression,
		blobOffsets: blobOffsets,
		blobData:    blobData,
	}, nil
}

// parseCompressedCluster parses a cluster where the entire content (offset table
// and blob data) is compressed as a single stream.
func parseCompressedCluster(clusterBytes []byte, info ClusterInfo, offsetSize int, factory *CompressorFactory) (*Cluster, error) {
	decompressor, err := factory.DecompressorForType(info.Compression)
	if err != nil {
		return nil, err
	}

	decompressedData, err := decompressor.Decompress(bytes.NewReader(clusterBytes[1:]))
	if err != nil {
		return nil, fmt.Errorf("decompressing cluster: %w", err)
	}

	if len(decompressedData) < offsetSize {
		return nil, fmt.Errorf("%w: decompressed cluster too short for offset table", ErrFormat)
	}

	var firstOffset uint64
	if info.IsExtended {
		firstOffset = readUint64(decompressedData[0:])
	} else {
		firstOffset = uint64(readUint32(decompressedData[0:]))
	}

	numOffsets := int(firstOffset) / offsetSize
	if numOffsets < 2 {
		return nil, fmt.Errorf("%w: decompressed cluster requires at least 2 offset entries (got %d offsets, first=%d)", ErrFormat, numOffsets, firstOffset)
	}

	blobOffsets := make([]uint64, numOffsets)
	for i := 0; i < numOffsets; i++ {
		pos := i * offsetSize
		if info.IsExtended {
			blobOffsets[i] = readUint64(decompressedData[pos:]) - firstOffset
		} else {
			blobOffsets[i] = uint64(readUint32(decompressedData[pos:])) - firstOffset
		}
	}

	if int(firstOffset) > len(decompressedData) {
		return nil, fmt.Errorf("%w: decompressed offset table size %d exceeds decompressed data size %d", ErrFormat, firstOffset, len(decompressedData))
	}

	blobData := decompressedData[firstOffset:]

	return &Cluster{
		compression: info.Compression,
		blobOffsets: blobOffsets,
		blobData:    blobData,
	}, nil
}

// BlobCount returns the number of blobs in this cluster.
func (c *Cluster) BlobCount() int {
	return len(c.blobOffsets) - 1
}

// Blob returns the decompressed data for the blob at the given index.
func (c *Cluster) Blob(index uint32) ([]byte, error) {
	if int(index) >= c.BlobCount() {
		return nil, fmt.Errorf("%w: blob index %d out of range (0-%d)", ErrFormat, index, c.BlobCount()-1)
	}

	start := c.blobOffsets[index]
	end := c.blobOffsets[index+1]

	if start > end {
		return nil, fmt.Errorf("%w: invalid blob offsets %d-%d", ErrFormat, start, end)
	}

	if uint64(len(c.blobData)) < end {
		return nil, fmt.Errorf("%w: blob data truncated (need %d, have %d)", ErrFormat, end, len(c.blobData))
	}

	result := make([]byte, end-start)
	copy(result, c.blobData[start:end])

	return result, nil
}

// BlobRange returns a portion of the decompressed blob data.
// This is more efficient than Blob() for partial reads.
func (c *Cluster) BlobRange(index uint32, offset, size uint64) ([]byte, error) {
	if int(index) >= c.BlobCount() {
		return nil, fmt.Errorf("%w: blob index %d out of range", ErrFormat, index)
	}

	blobStart := c.blobOffsets[index]
	blobEnd := c.blobOffsets[index+1]
	blobSize := blobEnd - blobStart

	if offset > blobSize {
		return nil, fmt.Errorf("%w: offset %d exceeds blob size %d", ErrFormat, offset, blobSize)
	}

	if offset+size > blobSize {
		size = blobSize - offset
	}

	result := make([]byte, size)
	copy(result, c.blobData[blobStart+offset:blobStart+offset+size])

	return result, nil
}

// Compression returns the compression type used for this cluster.
func (c *Cluster) Compression() Compression {
	return c.compression
}

// EncodeCluster serializes a set of blobs into the ZIM cluster binary format.
//
// For uncompressed clusters:
//
//	[info byte] [offset table: N+1 entries] [blob data concatenated]
//
// For compressed clusters:
//
//	[info byte] [compressed stream containing: offset table + blob data]
func EncodeCluster(compression Compression, blobs [][]byte, compressor Compressor) ([]byte, error) {
	isExtended := calculateTotalSize(blobs) > 0xFFFFFFFF
	offsetSize := clusterOffsetSize32
	if isExtended {
		offsetSize = clusterOffsetSize64
	}

	numBlobs := len(blobs)

	// Build offset table and concatenate blob data.
	var rawData bytes.Buffer
	blobOffsets := make([]uint64, numBlobs+1)

	// First offset = size of the offset table itself (without info byte for compressed).
	// For uncompressed, the offsets include the info byte.
	firstOffset := uint64((numBlobs + 1) * offsetSize)
	blobOffsets[0] = firstOffset

	for i, blob := range blobs {
		blobOffsets[i+1] = blobOffsets[i] + uint64(len(blob))
		rawData.Write(blob)
	}

	// Write the offset table followed by blob data into a staging buffer.
	var stagingBuf bytes.Buffer
	for _, off := range blobOffsets {
		offsetBytes := make([]byte, offsetSize)
		if isExtended {
			putUint64(offsetBytes, 0, off)
		} else {
			putUint32(offsetBytes, 0, uint32(off))
		}
		stagingBuf.Write(offsetBytes)
	}
	stagingBuf.Write(rawData.Bytes())

	info := ClusterInfo{
		Compression: compression,
		IsExtended:  isExtended,
	}

	var buf bytes.Buffer
	buf.WriteByte(EncodeClusterInfo(info))

	if compression == CompressionNone {
		// For uncompressed: the first offset is (N+1)*offsetSize (no info byte included).
		uncompOffsets := make([]uint64, numBlobs+1)
		uncompOffsets[0] = uint64((numBlobs + 1) * offsetSize)
		for i, blob := range blobs {
			uncompOffsets[i+1] = uncompOffsets[i] + uint64(len(blob))
		}

		for _, off := range uncompOffsets {
			offsetBytes := make([]byte, offsetSize)
			if isExtended {
				putUint64(offsetBytes, 0, off)
			} else {
				putUint32(offsetBytes, 0, uint32(off))
			}
			buf.Write(offsetBytes)
		}
		buf.Write(rawData.Bytes())
	} else {
		// For compressed: compress offset table + blob data together.
		compressed, err := compressor.Compress(stagingBuf.Bytes())
		if err != nil {
			return nil, fmt.Errorf("compressing cluster: %w", err)
		}
		buf.Write(compressed)
	}

	return buf.Bytes(), nil
}

// calculateTotalSize returns the total uncompressed size of all blobs.
func calculateTotalSize(blobs [][]byte) uint64 {
	var total uint64
	for _, blob := range blobs {
		total += uint64(len(blob))
	}
	return total
}


