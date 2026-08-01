package zim

import (
	"encoding/hex"
	"errors"
	"fmt"
)

// Uuid is a 16-byte universally unique identifier as stored in ZIM file headers.
type Uuid [16]byte

// ParseUuid parses a UUID from its hexadecimal string representation.
// The expected format is 32 hex digits, optionally separated by hyphens
// in the standard 8-4-4-4-12 pattern.
func ParseUuid(s string) (Uuid, error) {
	if s == "" {
		return Uuid{}, nil
	}

	// Strip hyphens for lenient parsing.
	stripped := make([]byte, 0, 32)
	for i := 0; i < len(s); i++ {
		if s[i] != '-' {
			stripped = append(stripped, s[i])
		}
	}

	if len(stripped) != 32 {
		return Uuid{}, errors.New("zim: invalid UUID: expected 32 hex digits")
	}

	var uuid Uuid
	_, err := hex.Decode(uuid[:], stripped)
	if err != nil {
		return Uuid{}, fmt.Errorf("zim: invalid UUID hex: %w", err)
	}

	return uuid, nil
}

// String returns the UUID in standard 8-4-4-4-12 hexadecimal format.
func (u Uuid) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

// IsZero returns true if all 16 bytes of the UUID are zero.
func (u Uuid) IsZero() bool {
	for _, b := range u {
		if b != 0 {
			return false
		}
	}
	return true
}

// readUuid reads a 16-byte UUID from buf starting at the given offset.
func readUuid(buf []byte, offset int) Uuid {
	var uuid Uuid
	copy(uuid[:], buf[offset:offset+16])
	return uuid
}

// putUuid writes a 16-byte UUID to buf starting at the given offset.
func putUuid(buf []byte, offset int, uuid Uuid) {
	copy(buf[offset:offset+16], uuid[:])
}
