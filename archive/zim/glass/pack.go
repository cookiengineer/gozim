package glass

import (
	"encoding/binary"
	"fmt"
)

// packUint encodes a uint64 as a variable-length integer using 7 bits per byte.
// The MSB of each byte is a continuation bit: set = more bytes follow.
func packUint(value uint64) []byte {
	if value == 0 {
		return []byte{0}
	}

	var buf [10]byte
	i := 0

	for value >= 128 {
		buf[i] = byte(value&0x7F) | 0x80
		i++
		value >>= 7
	}
	buf[i] = byte(value)
	i++

	return buf[:i]
}

// unpackUint decodes a variable-length integer from buf.
// Returns the value, number of bytes consumed, and any error.
func unpackUint(buf []byte) (uint64, int, error) {
	var value uint64
	var shift uint

	for i, b := range buf {
		value |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return value, i + 1, nil
		}
		shift += 7
		if shift >= 64 {
			return 0, 0, fmt.Errorf("glass: varint too long")
		}
	}

	return 0, 0, fmt.Errorf("glass: truncated varint")
}

// packUintPreservingSort encodes a uint64 such that lexicographic byte
// comparison equals numeric comparison. Uses leading 1-bits to indicate length.
func packUintPreservingSort(value uint64) []byte {
	if value < 0x8000 {
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(value))
		return buf
	}

	if value < 0x400000 {
		buf := make([]byte, 3)
		buf[0] = 0x80 | byte(value>>16)
		buf[1] = byte(value >> 8)
		buf[2] = byte(value)
		return buf
	}

	if value < 0x20000000 {
		buf := make([]byte, 4)
		buf[0] = 0xC0 | byte(value>>24)
		buf[1] = byte(value >> 16)
		buf[2] = byte(value >> 8)
		buf[3] = byte(value)
		return buf
	}

	if value < 0x1000000000 {
		buf := make([]byte, 5)
		buf[0] = 0xE0 | byte(value>>32)
		buf[1] = byte(value >> 24)
		buf[2] = byte(value >> 16)
		buf[3] = byte(value >> 8)
		buf[4] = byte(value)
		return buf
	}

	if value < 0x80000000000 {
		buf := make([]byte, 6)
		buf[0] = 0xF0 | byte(value>>40)
		buf[1] = byte(value >> 32)
		buf[2] = byte(value >> 24)
		buf[3] = byte(value >> 16)
		buf[4] = byte(value >> 8)
		buf[5] = byte(value)
		return buf
	}

	if value < 0x4000000000000 {
		buf := make([]byte, 7)
		buf[0] = 0xF8 | byte(value>>48)
		buf[1] = byte(value >> 40)
		buf[2] = byte(value >> 32)
		buf[3] = byte(value >> 24)
		buf[4] = byte(value >> 16)
		buf[5] = byte(value >> 8)
		buf[6] = byte(value)
		return buf
	}

	if value < 0x200000000000000 {
		buf := make([]byte, 8)
		buf[0] = 0xFC | byte(value>>56)
		buf[1] = byte(value >> 48)
		buf[2] = byte(value >> 40)
		buf[3] = byte(value >> 32)
		buf[4] = byte(value >> 24)
		buf[5] = byte(value >> 16)
		buf[6] = byte(value >> 8)
		buf[7] = byte(value)
		return buf
	}

	// 9 bytes.
	buf := make([]byte, 9)
	buf[0] = 0xFE
	binary.BigEndian.PutUint64(buf[1:], value)
	return buf
}

// unpackUintPreservingSort decodes a sort-preserving encoded uint64.
// Returns the value, number of bytes consumed, and any error.
func unpackUintPreservingSort(buf []byte) (uint64, int, error) {
	if len(buf) < 2 {
		return 0, 0, fmt.Errorf("glass: truncated sort-preserving uint")
	}

	first := buf[0]

	if first&0x80 == 0 {
		if len(buf) < 2 {
			return 0, 0, fmt.Errorf("glass: truncated 2-byte sort-preserving uint")
		}
		value := uint64(first)<<8 | uint64(buf[1])
		return value, 2, nil
	}

	leadingOnes := 0
	temp := first
	for temp&0x80 != 0 {
		leadingOnes++
		temp <<= 1
	}

	if leadingOnes == 8 {
		// 9-byte format: 0xFE followed by 8-byte big-endian.
		if len(buf) < 9 {
			return 0, 0, fmt.Errorf("glass: truncated 9-byte sort-preserving uint")
		}
		value := binary.BigEndian.Uint64(buf[1:9])
		return value, 9, nil
	}

	nBytes := leadingOnes + 1
	if len(buf) < nBytes {
		return 0, 0, fmt.Errorf("glass: truncated %d-byte sort-preserving uint", nBytes)
	}

	mask := byte(0xFF) >> uint(leadingOnes+1)
	value := uint64(first & mask)

	for i := 1; i < nBytes; i++ {
		value = (value << 8) | uint64(buf[i])
	}

	return value, nBytes, nil
}

// packStringPreservingSort encodes a string for sort-preserving key use.
// Null bytes (\x00) are escaped as \x00\xff. The string is terminated by
// a bare \x00 byte. The last string in a sequence may omit the terminator.
func packStringPreservingSort(s string) []byte {
	var result []byte

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == 0 {
			result = append(result, 0x00, 0xFF)
		} else {
			result = append(result, ch)
		}
	}

	return result
}

// packStringPreservingSortTerminated encodes a string with a null terminator.
func packStringPreservingSortTerminated(s string) []byte {
	result := packStringPreservingSort(s)
	result = append(result, 0x00)
	return result
}

// unpackStringPreservingSort decodes a sort-preserving encoded string.
// Returns the string, number of bytes consumed, and any error.
func unpackStringPreservingSort(buf []byte) (string, int, error) {
	var result []byte
	i := 0

	for i < len(buf) {
		if buf[i] == 0 {
			if i+1 < len(buf) && buf[i+1] == 0xFF {
				result = append(result, 0)
				i += 2
				continue
			}
			// Bare null: end of string (don't include the null).
			i++
			return string(result), i, nil
		}
		result = append(result, buf[i])
		i++
	}

	// End of buffer: unterminated string (last in sequence).
	return string(result), i, nil
}

// packBool encodes a boolean as a single byte.
func packBool(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// unpackBool decodes a boolean from a byte.
func unpackBool(b byte) bool {
	return b != 0
}
