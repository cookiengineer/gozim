// Package huff0 provides fast Huffman encoding and decoding for
// the Zstandard literals section.
//
// Huffman coding assigns variable-length bit sequences to symbols
// based on their frequency. This package implements the Huffman
// variant used in the Zstandard format, with 11-bit maximum code
// length and FSE-compressed Huffman table headers.
//
// # Compression
//
// Compress data with a new Huffman table each time:
//
//	var s huff0.Scratch
//	compressed, _, err := huff0.Compress1X(data, &s)
//	compressed, _, err := huff0.Compress4X(data, &s)
//
// Compress4X splits the input into four independent streams for
// better compression on larger inputs.
//
// # Decompression
//
// Decompress by first reading the Huffman table, then the data:
//
//	s, remain, err := huff0.ReadTable(compressed, nil)
//	dec := s.Decoder()
//	data, err := dec.Decompress1X(nil, remain)
//	data, err := dec.Decompress4X(nil, remain)
//
// # Table reuse
//
// For repeated compression of similar data, build a table once and
// reuse it across compressions:
//
//	s.BuildCTable(&histogram)
//	s.Reuse = huff0.ReusePolicyMust
//	compressed, _, _ := huff0.Compress1X(data, &s)
//
// Use AppendTable to serialize a pre-built table; use ReadTable to
// deserialize it on the decoder side.
//
// # Limits
//
// BlockSizeMax (262143 bytes, or (1<<18)-1) is the maximum input size
// for a single block. ErrTooBig, ErrIncompressible, and ErrUseRLE are
// returned when the input exceeds limits or cannot be efficiently
// compressed.
package huff0
