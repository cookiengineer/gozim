// Package xz implements reading and writing of XZ compressed data.
//
// XZ is a container format that wraps LZMA2-compressed data with
// integrity checks and index metadata. It provides better compression
// than gzip and is commonly used for software distribution.
//
// # Reading
//
// Use NewReader to decompress an XZ stream:
//
//	r, err := xz.NewReader(input)
//	data, _ := io.ReadAll(r)
//
// # Writing
//
// Use NewWriter to compress data into an XZ stream:
//
//	w, err := xz.NewWriter(output)
//	w.Write(data)
//	w.Close()
//
// # Configuration
//
// Both ReaderConfig and WriterConfig provide Verify methods and
// alternative constructors. The WriterConfig supports selecting the
// checksum type:
//
//	cfg := xz.WriterConfig{
//	    CheckSum: xz.SHA256,
//	    DictCap:  8 * 1024 * 1024,
//	}
//	w, _ := cfg.NewWriter(output)
//
// Available checksum types (CheckSum field):
//
//   - None (0x0): no integrity check
//   - CRC32 (0x1): 32-bit CRC (IEEE polynomial)
//   - CRC64 (0x4): 64-bit CRC (ECMA polynomial, default)
//   - SHA256 (0xA): SHA-256 hash
//
// # Format
//
// An XZ stream consists of:
//   - Stream Header (12 bytes): magic, flags, CRC32
//   - Block Headers and compressed data
//   - Index: list of block records
//   - Stream Footer (12 bytes): CRC32, index size, flags, magic
//
// Multiple streams may be concatenated. The reader transparently
// handles stream boundaries. Set SingleStream in ReaderConfig to
// disable this behavior.
//
// ValidHeader checks whether data begins with a valid XZ header:
//
//	if xz.ValidHeader(data[:12]) { ... }
package xz
