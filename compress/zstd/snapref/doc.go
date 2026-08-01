// Package snapref is a stub implementation of Snappy decompression for
// reading snappy-encoded blocks within Zstandard frames.
//
// Snappy is a fast compression algorithm developed by Google. Some
// Zstandard implementations store blocks in snappy format instead of
// the standard compressed block format.
//
// This package returns errors for all operations because snappy
// decompression is not supported in this pure Go port. Use a
// separate snappy library or convert the data to standard Zstandard
// format if snappy support is needed.
//
// ErrCorrupt is returned when the snappy input appears corrupt. All
// Decode and DecodedLen calls return an error indicating that snappy
// is not supported.
package snapref
