// Package zstd provides Zstandard (zstd) compression and decompression.
//
// Zstandard is a fast lossless compression algorithm targeting real-time
// compression scenarios at zlib-level and better compression ratios.
// It is defined in RFC 8878.
//
// # Compression
//
// Use NewWriter to create an encoder and write compressed data to an
// io.Writer. Choose a compression level with WithEncoderLevel:
//
//	w, err := zstd.NewWriter(output, zstd.WithEncoderLevel(zstd.SpeedDefault))
//	w.Write(data)
//	w.Close()
//
// For in-memory compression without an io.Writer, use EncodeAll or
// the EncodeTo convenience function:
//
//	compressed := zstd.EncodeTo(nil, data)
//
// Available compression levels:
//
//   - SpeedFastest: fastest, lower compression
//   - SpeedDefault: good balance (default)
//   - SpeedBetterCompression: higher compression
//   - SpeedBestCompression: maximum compression
//
// # Decompression
//
// Use NewReader to create a decoder:
//
//	r, err := zstd.NewReader(input)
//	data, _ := io.ReadAll(r)
//
// For in-memory decompression, use DecodeAll or DecodeTo:
//
//	data, err := zstd.DecodeTo(nil, compressed)
//
// # Advanced options
//
// The encoder and decoder support many options for tuning performance
// and memory usage:
//
//	enc, _ := zstd.NewWriter(output,
//	    zstd.WithEncoderLevel(zstd.SpeedBetterCompression),
//	    zstd.WithWindowSize(1<<20),
//	    zstd.WithEncoderCRC(true),
//	    zstd.WithSingleSegment(true),
//	)
//
//	dec, _ := zstd.NewReader(input,
//	    zstd.WithDecoderMaxWindow(1<<20),
//	    zstd.WithDecoderConcurrency(4),
//	)
//
// # Dictionary support
//
// Pre-trained dictionaries can improve compression of small data:
//
//	enc, _ := zstd.NewWriter(output, zstd.WithEncoderDict(dictData))
//	dec, _ := zstd.NewReader(input, zstd.WithDecoderDicts(dictData))
//
// # Frame and Header
//
// The Header type allows inspection of zstd frame headers without
// decompressing:
//
//	var hdr zstd.Header
//	remain, _ := hdr.DecodeAndStrip(frameData)
//
// # Skip frames
//
// Zstandard skip frames (magic 0x184D2A50) are transparently skipped
// by the decoder, allowing interleaved metadata.
//
// # Snappy compatibility
//
// The SnappyConverter type can convert zstd frames with snappy-encoded
// blocks to standard zstd frames.
//
// # Zip compatibility
//
// ZipCompressor and ZipDecompressor provide WriteCloser/ReadCloser
// compatible with the zstd-in-zip specification.
package zstd
