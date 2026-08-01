package zim

import (
	"fmt"
	"io"

	"github.com/cookiengineer/gozim/compress/xz"
	"github.com/cookiengineer/gozim/compress/zstd"
)

// Decompressor defines the interface for decompressing ZIM cluster data.
type Decompressor interface {
	// Decompress reads all data from reader r and returns the decompressed bytes.
	Decompress(r io.Reader) ([]byte, error)
}

// Compressor defines the interface for compressing data for ZIM cluster storage.
type Compressor interface {
	// Compress compresses the given byte slice.
	Compress(data []byte) ([]byte, error)
}

// CompressorFactory creates compressors and decompressors based on compression type.
type CompressorFactory struct{}

// NewCompressorFactory creates a new CompressorFactory.
func NewCompressorFactory() *CompressorFactory {
	return &CompressorFactory{}
}

// DecompressorForType returns a Decompressor for the given compression type.
func (cf *CompressorFactory) DecompressorForType(compressionType Compression) (Decompressor, error) {
	switch compressionType {
	case CompressionNone:
		return &NoneDecompressor{}, nil
	case CompressionZstd:
		return &ZstdDecompressor{}, nil
	case CompressionLzma:
		return &XZDecompressor{}, nil
	default:
		return nil, fmt.Errorf("%w: compression type %d", ErrUnsupportedCompression, compressionType)
	}
}

// CompressorForType returns a Compressor for the given compression type.
func (cf *CompressorFactory) CompressorForType(compressionType Compression) (Compressor, error) {
	switch compressionType {
	case CompressionNone:
		return &NoneCompressor{}, nil
	case CompressionZstd:
		return &ZstdCompressor{}, nil
	default:
		return nil, fmt.Errorf("%w: compression type %d", ErrUnsupportedCompression, compressionType)
	}
}

// NoneDecompressor passes data through without modification.
type NoneDecompressor struct{}

func (d *NoneDecompressor) Decompress(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// NoneCompressor passes data through without modification.
type NoneCompressor struct{}

func (c *NoneCompressor) Compress(data []byte) ([]byte, error) {
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// ZstdDecompressor decompresses zstd-compressed data using the vendored klauspost/compress library.
type ZstdDecompressor struct{}

func (d *ZstdDecompressor) Decompress(r io.Reader) ([]byte, error) {
	decoder, err := zstd.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("zstd decompressor init: %w", err)
	}
	defer decoder.Close()

	return io.ReadAll(decoder)
}

// ZstdCompressor compresses data using zstd at the default compression level.
type ZstdCompressor struct{}

func (c *ZstdCompressor) Compress(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
	)
	if err != nil {
		return nil, fmt.Errorf("zstd compressor init: %w", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(data, make([]byte, 0, len(data)/2))
	return compressed, nil
}

// XZDecompressor decompresses XZ/LZMA-compressed data using the vendored ulikunitz/xz library.
// This is used for reading legacy ZIM files that use LZMA compression.
type XZDecompressor struct{}

func (d *XZDecompressor) Decompress(r io.Reader) ([]byte, error) {
	reader, err := xz.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("xz decompressor init: %w", err)
	}

	return io.ReadAll(reader)
}
