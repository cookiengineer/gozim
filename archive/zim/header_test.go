package zim

import (
	"encoding/binary"
	"testing"
)

func makeValidHeaderBytes() []byte {
	buf := make([]byte, HeaderSize)

	binary.LittleEndian.PutUint32(buf[0:4], Magic)
	binary.LittleEndian.PutUint16(buf[4:6], MajorVersion)
	binary.LittleEndian.PutUint16(buf[6:8], MinorVersion)
	// UUID at offset 8 (leave all zeros)
	binary.LittleEndian.PutUint32(buf[24:28], 100)
	binary.LittleEndian.PutUint32(buf[28:32], 5)
	binary.LittleEndian.PutUint64(buf[32:40], 1024)
	binary.LittleEndian.PutUint64(buf[40:48], 0xFFFFFFFFFFFFFFFF) // no title index
	binary.LittleEndian.PutUint64(buf[48:56], 2048)
	binary.LittleEndian.PutUint64(buf[56:64], HeaderSize) // mimeListPos at 80
	binary.LittleEndian.PutUint32(buf[64:68], 0xFFFFFFFF) // no main page
	binary.LittleEndian.PutUint32(buf[68:72], 0xFFFFFFFF) // no layout page
	binary.LittleEndian.PutUint64(buf[72:80], 5000)       // checksumPos

	return buf
}

func TestParseHeaderValid(t *testing.T) {
	data := makeValidHeaderBytes()
	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("ParseHeader() unexpected error: %v", err)
	}

	if header.MagicNumber != Magic {
		t.Errorf("MagicNumber = 0x%08x, want 0x%08x", header.MagicNumber, Magic)
	}
	if header.MajorVersion != MajorVersion {
		t.Errorf("MajorVersion = %d, want %d", header.MajorVersion, MajorVersion)
	}
	if header.MinorVersion != MinorVersion {
		t.Errorf("MinorVersion = %d, want %d", header.MinorVersion, MinorVersion)
	}
	if header.EntryCount != 100 {
		t.Errorf("EntryCount = %d, want 100", header.EntryCount)
	}
	if header.ClusterCount != 5 {
		t.Errorf("ClusterCount = %d, want 5", header.ClusterCount)
	}
}

func TestParseHeaderInvalidMagic(t *testing.T) {
	data := makeValidHeaderBytes()
	binary.LittleEndian.PutUint32(data[0:4], 0xDEADBEEF)

	_, err := ParseHeader(data)
	if err == nil {
		t.Fatal("expected error for invalid magic, got nil")
	}
	if !isErrType(err, ErrFormat) {
		t.Errorf("expected ErrFormat, got %v", err)
	}
}

func TestParseHeaderUnsupportedVersion(t *testing.T) {
	data := makeValidHeaderBytes()
	binary.LittleEndian.PutUint16(data[4:6], 3) // major version 3

	_, err := ParseHeader(data)
	if err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
	if !isErrType(err, ErrUnsupportedVersion) {
		t.Errorf("expected ErrUnsupportedVersion, got %v", err)
	}
}

func TestParseHeaderTooShort(t *testing.T) {
	data := make([]byte, 10)
	_, err := ParseHeader(data)
	if err == nil {
		t.Fatal("expected error for short data, got nil")
	}
}

func TestHeaderHasChecksum(t *testing.T) {
	tests := []struct {
		name       string
		mimeListPos uint64
		checksumPos uint64
		want       bool
	}{
		{"has checksum", HeaderSize, 5000, true},
		{"no mime offset", 0, 5000, false},
		{"no checksum pos", HeaderSize, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Header{MimeListPos: tt.mimeListPos, ChecksumPos: tt.checksumPos}
			if got := h.HasChecksum(); got != tt.want {
				t.Errorf("HasChecksum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderHasNewNamespaceScheme(t *testing.T) {
	tests := []struct {
		major uint16
		minor uint16
		want  bool
	}{
		{MajorVersion, 0, false},
		{MajorVersion, 1, true},
		{MajorVersion, 3, true},
		{OldMajorVersion, 0, false},
		{OldMajorVersion, 1, false},
	}

	for _, tt := range tests {
		h := &Header{MajorVersion: tt.major, MinorVersion: tt.minor}
		if got := h.HasNewNamespaceScheme(); got != tt.want {
			t.Errorf("HasNewNamespaceScheme(v%d.%d) = %v, want %v", tt.major, tt.minor, got, tt.want)
		}
	}
}

func TestHeaderHasTitleIndex(t *testing.T) {
	h := &Header{TitleIdxPos: 0xFFFFFFFFFFFFFFFF}
	if h.HasTitleIndex() {
		t.Error("expected no title index for sentinel value")
	}

	h.TitleIdxPos = 1000
	if !h.HasTitleIndex() {
		t.Error("expected title index for valid position")
	}
}

func TestHeaderHasMainPage(t *testing.T) {
	h := &Header{MainPage: 0xFFFFFFFF}
	if h.HasMainPage() {
		t.Error("expected no main page for sentinel value")
	}

	h.MainPage = 5
	if !h.HasMainPage() {
		t.Error("expected main page for valid index")
	}
}

func TestHeaderValidate(t *testing.T) {
	fileSize := int64(10000)

	t.Run("valid header", func(t *testing.T) {
		data := makeValidHeaderBytes()
		header, _ := ParseHeader(data)
		if err := header.Validate(fileSize); err != nil {
			t.Errorf("validate failed for valid header: %v", err)
		}
	})

	t.Run("zero entry count", func(t *testing.T) {
		data := makeValidHeaderBytes()
		binary.LittleEndian.PutUint32(data[24:28], 0)
		header, _ := ParseHeader(data)
		if err := header.Validate(fileSize); err == nil {
			t.Error("expected error for zero entry count")
		}
	})

	t.Run("zero cluster count", func(t *testing.T) {
		data := makeValidHeaderBytes()
		binary.LittleEndian.PutUint32(data[28:32], 0)
		header, _ := ParseHeader(data)
		if err := header.Validate(fileSize); err == nil {
			t.Error("expected error for zero cluster count")
		}
	})

	t.Run("URL ptr out of bounds", func(t *testing.T) {
		data := makeValidHeaderBytes()
		binary.LittleEndian.PutUint64(data[32:40], uint64(fileSize+1))
		header, _ := ParseHeader(data)
		if err := header.Validate(fileSize); err == nil {
			t.Error("expected error for out-of-bounds URL ptr pos")
		}
	})

	t.Run("main page out of bounds", func(t *testing.T) {
		data := makeValidHeaderBytes()
		binary.LittleEndian.PutUint32(data[64:68], 200)
		header, _ := ParseHeader(data)
		if err := header.Validate(fileSize); err == nil {
			t.Error("expected error for main page index >= entry count")
		}
	})
}

func TestHeaderEncodeDecodeRoundTrip(t *testing.T) {
	original := NewHeader()
	original.Uuid = Uuid{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x10, 0x32, 0x54, 0x76, 0x98, 0xba, 0xdc, 0xfe}
	original.EntryCount = 42
	original.ClusterCount = 7
	original.UrlPtrPos = 2048
	original.TitleIdxPos = 5000
	original.ClusterPtrPos = 10000
	original.MimeListPos = 80
	original.MainPage = 0
	original.ChecksumPos = 20000

	encoded := original.EncodeHeader()
	decoded, err := ParseHeader(encoded)
	if err != nil {
		t.Fatalf("ParseHeader() after encode failed: %v", err)
	}

	if decoded.MagicNumber != original.MagicNumber {
		t.Errorf("MagicNumber mismatch: %x != %x", decoded.MagicNumber, original.MagicNumber)
	}
	if decoded.MajorVersion != original.MajorVersion {
		t.Errorf("MajorVersion mismatch: %d != %d", decoded.MajorVersion, original.MajorVersion)
	}
	if decoded.EntryCount != original.EntryCount {
		t.Errorf("EntryCount mismatch: %d != %d", decoded.EntryCount, original.EntryCount)
	}
	if decoded.ClusterCount != original.ClusterCount {
		t.Errorf("ClusterCount mismatch: %d != %d", decoded.ClusterCount, original.ClusterCount)
	}
	if decoded.Uuid != original.Uuid {
		t.Errorf("Uuid mismatch: %v != %v", decoded.Uuid, original.Uuid)
	}
	if decoded.MainPage != original.MainPage {
		t.Errorf("MainPage mismatch: %d != %d", decoded.MainPage, original.MainPage)
	}
	if decoded.ChecksumPos != original.ChecksumPos {
		t.Errorf("ChecksumPos mismatch: %d != %d", decoded.ChecksumPos, original.ChecksumPos)
	}
}

func TestNewHeaderDefaults(t *testing.T) {
	h := NewHeader()

	if h.MagicNumber != Magic {
		t.Errorf("MagicNumber = 0x%08x, want 0x%08x", h.MagicNumber, Magic)
	}
	if h.MajorVersion != MajorVersion {
		t.Errorf("MajorVersion = %d, want %d", h.MajorVersion, MajorVersion)
	}
	if h.MinorVersion != MinorVersion {
		t.Errorf("MinorVersion = %d, want %d", h.MinorVersion, MinorVersion)
	}
	if h.HasTitleIndex() {
		t.Error("new header should not have title index")
	}
	if h.HasMainPage() {
		t.Error("new header should not have main page")
	}
	if !h.HasNewNamespaceScheme() {
		t.Error("new header should use new namespace scheme")
	}
	if h.MimeListPos != HeaderSize {
		t.Errorf("MimeListPos = %d, want %d (header size)", h.MimeListPos, HeaderSize)
	}
}

// isErrType checks if err wraps the target sentinel error.
func isErrType(err, target error) bool {
	if err == nil {
		return false
	}
	return unwrapTo(err, target)
}

func unwrapTo(err, target error) bool {
	if err == target {
		return true
	}

	type wrapper interface {
		Unwrap() error
	}

	for {
		w, ok := err.(wrapper)
		if !ok {
			return false
		}
		err = w.Unwrap()
		if err == target {
			return true
		}
	}
}
