package zim

import (
	"crypto/md5"
	"fmt"
	"io"
)

// ComputeChecksum calculates the MD5 checksum of the file content
// up to (but not including) the checksum position.
func ComputeChecksum(reader Reader, checksumPos uint64) ([]byte, error) {
	if checksumPos == 0 {
		return nil, nil
	}

	h := md5.New()

	// Read in chunks to avoid loading the entire file.
	chunkSize := int64(64 * 1024) // 64KB chunks.
	var offset int64

	for offset < int64(checksumPos) {
		remaining := int64(checksumPos) - offset
		if remaining > chunkSize {
			remaining = chunkSize
		}

		buf := make([]byte, remaining)
		n, err := reader.ReadAt(buf, offset)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("reading for checksum at offset %d: %w", offset, err)
		}

		if n > 0 {
			h.Write(buf[:n])
			offset += int64(n)
		} else {
			break
		}
	}

	return h.Sum(nil), nil
}

// VerifyChecksum checks that the stored checksum matches the computed one.
func VerifyChecksum(reader Reader, checksumPos uint64) error {
	computed, err := ComputeChecksum(reader, checksumPos)
	if err != nil {
		return err
	}

	stored := make([]byte, ChecksumSize)
	if _, err := reader.ReadAt(stored, int64(checksumPos)); err != nil {
		return fmt.Errorf("reading stored checksum: %w", err)
	}

	if !bytesEq(computed, stored) {
		return fmt.Errorf("%w: computed %x, stored %x", ErrChecksum, computed, stored)
	}

	return nil
}

func bytesEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
