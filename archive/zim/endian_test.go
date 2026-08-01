package zim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestReadUint16(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint16
	}{
		{"zero", []byte{0x00, 0x00}, 0},
		{"one", []byte{0x01, 0x00}, 1},
		{"max", []byte{0xFF, 0xFF}, 65535},
		{"typical", []byte{0x34, 0x12}, 0x1234},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := readUint16(tt.input)
			if result != tt.expected {
				t.Errorf("readUint16(%x) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReadUint32(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint32
	}{
		{"zero", []byte{0x00, 0x00, 0x00, 0x00}, 0},
		{"one", []byte{0x01, 0x00, 0x00, 0x00}, 1},
		{"magic", []byte{0x5A, 0x49, 0x4D, 0x04}, 0x044d495a}, // ZIM magic
		{"max", []byte{0xFF, 0xFF, 0xFF, 0xFF}, 4294967295},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := readUint32(tt.input)
			if result != tt.expected {
				t.Errorf("readUint32(%x) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReadUint64(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint64
	}{
		{"zero", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 0},
		{"one", []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 1},
		{"large", []byte{0x00, 0xE4, 0x0B, 0x54, 0x02, 0x00, 0x00, 0x00}, 10000000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := readUint64(tt.input)
			if result != tt.expected {
				t.Errorf("readUint64(%x) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPutUint16(t *testing.T) {
	buf := make([]byte, 2)
	putUint16(buf, 0, 0x1234)
	if !bytes.Equal(buf, []byte{0x34, 0x12}) {
		t.Errorf("putUint16 = %x, want %x", buf, []byte{0x34, 0x12})
	}
}

func TestPutUint32(t *testing.T) {
	buf := make([]byte, 4)
	putUint32(buf, 0, Magic)
	expected := make([]byte, 4)
	binary.LittleEndian.PutUint32(expected, Magic)
	if !bytes.Equal(buf, expected) {
		t.Errorf("putUint32 = %x, want %x", buf, expected)
	}
}

func TestPutUint64(t *testing.T) {
	buf := make([]byte, 8)
	putUint64(buf, 0, 0xDEADBEEFCAFEBABE)
	expected := make([]byte, 8)
	binary.LittleEndian.PutUint64(expected, 0xDEADBEEFCAFEBABE)
	if !bytes.Equal(buf, expected) {
		t.Errorf("putUint64 = %x, want %x", buf, expected)
	}
}

func TestEndianRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		value  uint32
	}{
		{"zero", 0},
		{"magic", Magic},
		{"max", 4294967295},
		{"entry_count", 12345},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, 4)
			putUint32(buf, 0, tt.value)
			result := readUint32(buf)
			if result != tt.value {
				t.Errorf("round trip failed: put %d, read %d", tt.value, result)
			}
		})
	}
}

