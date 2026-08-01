// Package le provides functions to load 32-bit and 64-bit little-endian
// integers from byte slices at arbitrary (possibly unaligned) offsets.
//
// These functions are used internally by the huff0 bit reader to avoid
// bounds checks on every access. They are not intended for general use.
//
//   - Load32(b []byte, i int32) uint32
//   - Load64(b []byte, i int32) uint64
package le
