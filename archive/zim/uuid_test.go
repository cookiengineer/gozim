package zim

import (
	"testing"
)

func TestParseUuid(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Uuid
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			want:    Uuid{},
			wantErr: false,
		},
		{
			name:    "standard format with hyphens",
			input:   "12345678-9abc-def0-1234-56789abcdef0",
			want:    Uuid{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			wantErr: false,
		},
		{
			name:    "without hyphens",
			input:   "123456789abcdef0123456789abcdef0",
			want:    Uuid{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			wantErr: false,
		},
		{
			name:    "all zeros",
			input:   "00000000-0000-0000-0000-000000000000",
			want:    Uuid{},
			wantErr: false,
		},
		{
			name:    "invalid hex character",
			input:   "00000000-0000-0000-0000-00000000000g",
			wantErr: true,
		},
		{
			name:    "too short",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUuid(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUuid() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseUuid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUuidString(t *testing.T) {
	uuid := Uuid{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	expected := "12345678-9abc-def0-1234-56789abcdef0"
	if uuid.String() != expected {
		t.Errorf("String() = %q, want %q", uuid.String(), expected)
	}
}

func TestUuidIsZero(t *testing.T) {
	tests := []struct {
		name string
		uuid Uuid
		want bool
	}{
		{"all zeros", Uuid{}, true},
		{"non-zero first byte", Uuid{0x01}, false},
		{"non-zero last byte", Uuid{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.uuid.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadWriteUuid(t *testing.T) {
	original := Uuid{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x10, 0x32, 0x54, 0x76, 0x98, 0xba, 0xdc, 0xfe}
	buf := make([]byte, 16)

	putUuid(buf, 0, original)
	got := readUuid(buf, 0)

	if got != original {
		t.Errorf("round trip failed: %v != %v", got, original)
	}
}
