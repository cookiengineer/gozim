// Package hash provides rolling hash functions used by the LZMA
// compressor to maintain a sliding window of byte sequences.
//
// Two rolling hash implementations are provided:
//
//   - CyclicPoly: a cyclic polynomial hash using precomputed random
//     coefficients for each byte value.
//   - RabinKarp: the Rabin-Karp rolling hash with a configurable
//     multiplicative constant.
//
// Both implement the Roller interface, which computes the rolling hash
// of a fixed-size window one byte at a time.
//
// The Hashes function computes all rolling hash values over a byte
// slice using any Roller:
//
//	r := hash.NewCyclicPoly(4)
//	hashes := hash.Hashes(r, data)
package hash
