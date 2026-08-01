// Package fse implements Finite State Entropy encoding and decoding.
//
// FSE is an entropy codec based on Asymmetric Numeral Systems (ANS),
// used in the Zstandard format for encoding literal lengths, match
// lengths, and offset codes. It provides near-optimal compression
// ratios with table-driven decoding.
//
// FSE tables define a state machine with 2^AccuracyLog states. The
// decoder reads bits from a backward bitstream to transition between
// states and emit symbols.
//
// The Compress function builds an FSE table from symbol frequencies
// and encodes a symbol sequence. The Decompress function decodes a
// symbol sequence from an FSE table and bitstream.
//
// The Scratch type holds reusable buffers for compression and
// decompression to reduce allocations.
package fse
