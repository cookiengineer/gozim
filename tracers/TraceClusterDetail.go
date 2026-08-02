package tracers

import (
	"encoding/binary"
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
	"github.com/cookiengineer/gozim/utils"
)

func TraceClusterDetail(index int, offset uint64, data []byte) {

	fmt.Println()
	fmt.Printf("  Cluster %d at offset 0x%X, %d raw bytes\n", index, offset, len(data))
	fmt.Println()

	if len(data) < 1 {
		fmt.Println("  (empty)")
		fmt.Println()
		return
	}

	info := zim.ParseClusterInfo(data[0])
	fmt.Printf("  info byte:         0x%02X\n", data[0])
	fmt.Printf("  compression:        %d (%s)\n", info.Compression, describeCompression(info.Compression))
	fmt.Printf("  extended offsets:   %v\n", info.IsExtended)

	offsetSize := 4
	if info.IsExtended {
		offsetSize = 8
	}

	if info.Compression == zim.CompressionNone {
		if len(data) < 1+offsetSize {
			fmt.Println("  (too short for offset table)")
			fmt.Println()
			return
		}

		var firstOff uint64
		if info.IsExtended {
			firstOff = binary.LittleEndian.Uint64(data[1:])
		} else {
			firstOff = uint64(binary.LittleEndian.Uint32(data[1:]))
		}

		numOffsets := int(firstOff) / offsetSize
		if numOffsets < 2 {
			fmt.Printf("  offset table:      invalid (first offset=%d, numOffsets=%d)\n\n", firstOff, numOffsets)
			return
		}

		fmt.Printf("  offset table:       %d entries (%d bytes each)\n", numOffsets, offsetSize)
		maxOffsets := 8
		if numOffsets > maxOffsets {
			fmt.Printf("  (showing first %d of %d offsets)\n", maxOffsets, numOffsets)
		}
		for j := 0; j < numOffsets && j < maxOffsets; j++ {
			var offVal uint64
			pos := 1 + j*offsetSize
			if info.IsExtended {
				offVal = binary.LittleEndian.Uint64(data[pos:])
			} else {
				offVal = uint64(binary.LittleEndian.Uint32(data[pos:]))
			}
			blobRelative := offVal - firstOff
			fmt.Printf("    offset[%d] = %d (blob-relative: %d)\n", j, offVal, blobRelative)
		}

		numBlobs := numOffsets - 1
		fmt.Printf("  blobs:              %d\n", numBlobs)
		maxBlobs := 5
		if numBlobs > maxBlobs {
			fmt.Printf("  (showing first %d of %d blobs)\n", maxBlobs, numBlobs)
		}
		for j := 0; j < numBlobs && j < maxBlobs; j++ {
			var start, end uint64
			pos := 1 + j*offsetSize
			if info.IsExtended {
				start = binary.LittleEndian.Uint64(data[pos:]) - firstOff
				end = binary.LittleEndian.Uint64(data[pos+offsetSize:]) - firstOff
			} else {
				start = uint64(binary.LittleEndian.Uint32(data[pos:])) - firstOff
				end = uint64(binary.LittleEndian.Uint32(data[pos+offsetSize:])) - firstOff
			}
			size := end - start
			blobStart := 1 + int(firstOff) + int(start)
			blobPreview := data[blobStart:min(blobStart+16, len(data))]
			contentType := utils.DetectContentType(blobPreview)
			fmt.Printf("    blob[%d]: %d bytes (offset %d-%d)   %s\n", j, size, start, end, contentType)
		}

		rawBlobStart := 1 + int(firstOff)
		rawPreview := data[rawBlobStart:min(rawBlobStart+32, len(data))]
		fmt.Printf("  raw blob data at:  0x%X (+%d)\n", offset+uint64(rawBlobStart), rawBlobStart)
		fmt.Printf("  first 32 bytes:    %s\n", renderHex(rawPreview))
	} else {
		compStart := 1
		compSize := len(data) - compStart
		fmt.Printf("  compressed data:    %d bytes at offset +%d\n", compSize, compStart)
		fmt.Printf("  compressed head:    %s\n", renderHex(truncateBytes(data[compStart:], 32)))
		fmt.Printf("  (compressed — decompress the full cluster\n")
		fmt.Printf("   with ParseCluster for blob-by-blob inspection)\n")
	}
	fmt.Println()

}

