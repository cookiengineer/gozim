package zim

import (
	"encoding/binary"
	"testing"
)

// =============================================================================
// Spec: Magic number (0x044d495a)
// =============================================================================

func TestSpec_Magic(t *testing.T) {
	// Spec §Header: magicNumber = 72173914 (0x044d495a)
	if Magic != 0x044d495a {
		t.Errorf("Magic = 0x%08x, spec says 0x044d495a", Magic)
	}
	// Verify little-endian byte representation is "ZIM\x04"
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, Magic)
	if string(buf) != "ZIM\x04" {
		t.Errorf("Magic byte representation = %x, want 'ZIM\\x04'", buf)
	}
}

// =============================================================================
// Spec: Header size (80 bytes)
// =============================================================================

func TestSpec_HeaderSize(t *testing.T) {
	// Spec: Header is exactly 80 bytes
	if HeaderSize != 80 {
		t.Errorf("HeaderSize = %d, spec says 80", HeaderSize)
	}
}

// =============================================================================
// Spec: Compression codes
// =============================================================================

func TestSpec_CompressionCodes(t *testing.T) {
	// Spec §Clusters: 0=obsolete none, 1=none, 4=LZMA2, 5=zstd
	if CompressionNone != 1 {
		t.Errorf("CompressionNone = %d, spec says 1", CompressionNone)
	}
	if CompressionLzma != 4 {
		t.Errorf("CompressionLzma = %d, spec says 4", CompressionLzma)
	}
	if CompressionZstd != 5 {
		t.Errorf("CompressionZstd = %d, spec says 5", CompressionZstd)
	}
}

// =============================================================================
// Spec: Sentinel MIME type indices
// =============================================================================

func TestSpec_MimeSentinels(t *testing.T) {
	// Spec §Redirect entry: mimeType = 0xFFFF
	if mimeRedirect != 0xFFFF {
		t.Errorf("mimeRedirect = 0x%04x, spec says 0xFFFF", mimeRedirect)
	}
	// Spec §Linktarget/deleted: 0xFFFE, 0xFFFD
	if mimeLinkTarget != 0xFFFE {
		t.Errorf("mimeLinkTarget = 0x%04x, spec says 0xFFFE", mimeLinkTarget)
	}
	if mimeDeleted != 0xFFFD {
		t.Errorf("mimeDeleted = 0x%04x, spec says 0xFFFD", mimeDeleted)
	}
}

// =============================================================================
// Spec: Major/minor version handling
// =============================================================================

func TestSpec_Versions(t *testing.T) {
	if MajorVersion != 6 {
		t.Errorf("MajorVersion = %d, spec says 6", MajorVersion)
	}
	if MinorVersion != 3 {
		t.Errorf("MinorVersion = %d, spec says 3", MinorVersion)
	}
	if OldMajorVersion != 5 {
		t.Errorf("OldMajorVersion = %d, spec says 5", OldMajorVersion)
	}
}

// =============================================================================
// Spec: Sentinels (0xFFFFFFFF, 0xFFFFFFFFFFFFFFFF)
// =============================================================================

func TestSpec_Sentinels(t *testing.T) {
	h := NewHeader()

	if h.MainPage != 0xFFFFFFFF {
		t.Errorf("MainPage default = 0x%08x, spec says 0xFFFFFFFF", h.MainPage)
	}
	if h.LayoutPage != 0xFFFFFFFF {
		t.Errorf("LayoutPage default = 0x%08x, spec says 0xFFFFFFFF", h.LayoutPage)
	}
	if h.TitleIdxPos != 0xFFFFFFFFFFFFFFFF {
		t.Errorf("TitleIdxPos default = 0x%016x, spec says 0xFFFFFFFFFFFFFFFF", h.TitleIdxPos)
	}
}

// =============================================================================
// Spec: MIME list directly follows header (offset 80)
// =============================================================================

func TestSpec_MimeListPos(t *testing.T) {
	h := NewHeader()
	if h.MimeListPos != HeaderSize {
		t.Errorf("MimeListPos default = %d, spec says %d (header size)", h.MimeListPos, HeaderSize)
	}
}

// =============================================================================
// Spec: Checksum size (16 bytes = MD5)
// =============================================================================

func TestSpec_ChecksumSize(t *testing.T) {
	if ChecksumSize != 16 {
		t.Errorf("ChecksumSize = %d, spec says 16 (MD5)", ChecksumSize)
	}
}

// =============================================================================
// Spec: New namespace scheme detection
// =============================================================================

func TestSpec_NewNamespace(t *testing.T) {
	tests := []struct {
		major, minor uint16
		want         bool
	}{
		{5, 0, false}, // Old format before major/minor split
		{5, 1, false}, // v5 regardless of minor
		{6, 0, false}, // v6.0 still uses old namespaces
		{6, 1, true},  // v6.1 introduces new namespaces
		{6, 2, true},  // v6.2 continues new namespaces
		{6, 3, true},  // v6.3 continues new namespaces
	}

	for _, tt := range tests {
		h := &Header{MajorVersion: tt.major, MinorVersion: tt.minor}
		if got := h.HasNewNamespaceScheme(); got != tt.want {
			t.Errorf("v%d.%d HasNewNamespaceScheme() = %v, want %v", tt.major, tt.minor, got, tt.want)
		}
	}
}

// =============================================================================
// Spec: HasChecksum detection logic
// =============================================================================

func TestSpec_HasChecksum(t *testing.T) {
	// Spec: checksum is present when MIME list pos >= 80 AND checksum pos > 0
	tests := []struct {
		name        string
		mimeListPos uint64
		checksumPos uint64
		want        bool
	}{
		{"both_ok", HeaderSize, 100, true},
		{"mime_before_header", 20, 100, false},
		{"checksum_zero", HeaderSize, 0, false},
		{"both_bad", 20, 0, false},
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

// =============================================================================
// Spec: HasTitleIndex detection
// =============================================================================

func TestSpec_HasTitleIndex(t *testing.T) {
	// Spec: title index exists when TitleIdxPos != 0xFFFFFFFFFFFFFFFF
	h := &Header{TitleIdxPos: 0xFFFFFFFFFFFFFFFF}
	if h.HasTitleIndex() {
		t.Error("sentinel should indicate no title index")
	}
	h.TitleIdxPos = 0xFFFFFFFFFFFFFFFE
	if !h.HasTitleIndex() {
		t.Error("non-sentinel should indicate title index")
	}
}
