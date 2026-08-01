// Package lzma implements reading and writing of LZMA and LZMA2 compressed data.
//
// LZMA (Lempel-Ziv-Markov chain algorithm) provides high-ratio lossless
// compression using a sliding window, a binary tree match finder, and
// range encoding as the entropy coder. LZMA2 extends LZMA with support
// for chunking, flushing, and dictionary size encoding in one byte.
//
// This package supports both the legacy LZMA1 single-stream format and
// the LZMA2 chunked format used in the XZ container.
//
// # Reading
//
// Use NewReader for LZMA1 streams and NewReader2 for LZMA2 streams:
//
//	r, err := lzma.NewReader(input)
//	data, _ := io.ReadAll(r)
//
//	r2, err := lzma.NewReader2(input)
//	data, _ := io.ReadAll(r2)
//
// # Writing
//
// Use NewWriter for LZMA1 and NewWriter2 for LZMA2:
//
//	w, err := lzma.NewWriter(output)
//	w.Write(data)
//	w.Close()
//
//	w2, err := lzma.NewWriter2(output)
//	w2.Write(data)
//	w2.Close()
//
// # Configuration
//
// Both readers and writers support configuration via their respective
// Config types that provide Verify methods and alternative constructors:
//
//	cfg := lzma.Writer2Config{DictCap: 1 << 20, Properties: &lzma.Properties{LC: 3, LP: 0, PB: 2}}
//	w, _ := cfg.NewWriter2(output)
//
// # Properties
//
// LZMA compression is controlled by three parameters encoded in a single byte:
//
//   - LC (literal context bits): [0, 8], default 3
//   - LP (literal position bits): [0, 4], default 0
//   - PB (position bits): [0, 4], default 2
//
// The encoded byte is computed as (PB*5 + LP)*9 + LC. Use
// PropertiesForCode and Properties.Code to convert between the byte
// and the Properties struct.
//
// # Match Algorithms
//
// The encoder supports two match-finding algorithms:
//
//   - AlgorithmHashTable: faster, uses chained hashing (default)
//   - AlgorithmBinaryTree: slower, better compression ratio
//
// # Dictionary Size
//
// The dictionary size determines the sliding window and affects both
// memory usage and compression ratio. For LZMA2, dictionary sizes are
// encoded via EncodeDictCap/DecodeDictCap using a compact logarithmic
// scheme.
package lzma
