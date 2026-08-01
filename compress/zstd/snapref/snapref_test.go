package snapref_test

import (
	"strings"
	"testing"

	"github.com/cookiengineer/gozim/compress/zstd/snapref"
)

func TestDecodedLen(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("hello")},
		{"medium", []byte(strings.Repeat("x", 1000))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := snapref.DecodedLen(tt.data)
			if err == nil {
				t.Error("expected error, got nil")
			}
			if n != 0 {
				t.Errorf("expected 0, got %d", n)
			}
			if err.Error() != "snappy: not supported in pure Go zstd port" {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name string
		src  []byte
		dst  []byte
	}{
		{"empty", []byte{}, nil},
		{"nil_dst", []byte{1, 2, 3}, nil},
		{"with_dst", []byte{1, 2, 3}, make([]byte, 10)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := snapref.Decode(tt.dst, tt.src)
			if err == nil {
				t.Error("expected error, got nil")
			}
			if result != nil {
				t.Errorf("expected nil result, got %v", result)
			}
			if err.Error() != "snappy: snappy-encoded zstd blocks are not supported in this pure Go port" {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestErrCorrupt(t *testing.T) {
	if snapref.ErrCorrupt == nil {
		t.Error("ErrCorrupt should not be nil")
	}
	if snapref.ErrCorrupt.Error() != "snappy: corrupt input" {
		t.Errorf("unexpected error message: %v", snapref.ErrCorrupt)
	}
}

func TestDecodedLen_ConsistentError(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	for i := 0; i < 10; i++ {
		n, err := snapref.DecodedLen(data)
		if err == nil || n != 0 {
			t.Fatalf("iteration %d: expected consistent error", i)
		}
	}
}

func TestDecode_ConsistentError(t *testing.T) {
	src := []byte{0x01, 0x02, 0x03}
	dst := make([]byte, 100)
	for i := 0; i < 10; i++ {
		result, err := snapref.Decode(dst, src)
		if err == nil || result != nil {
			t.Fatalf("iteration %d: expected consistent error", i)
		}
	}
}
