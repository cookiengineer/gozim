package zim

import (
	"testing"
)

func TestParseDirentContent(t *testing.T) {
	// Build a content dirent manually.
	dirent := Dirent{
		MimeTypeIndex: 0,
		Namespace:     NamespaceContent,
		EntryType:     EntryTypeContent,
		ClusterNumber:  3,
		BlobNumber:     7,
		Path:           "index.html",
		Title:          "Home Page",
	}

	data := EncodeDirent(&dirent)

	parsed, bytesRead, err := ParseDirent(data)
	if err != nil {
		t.Fatalf("ParseDirent() error: %v", err)
	}

	if parsed.MimeTypeIndex != dirent.MimeTypeIndex {
		t.Errorf("MimeTypeIndex = %d, want %d", parsed.MimeTypeIndex, dirent.MimeTypeIndex)
	}
	if parsed.Namespace != dirent.Namespace {
		t.Errorf("Namespace = %c, want %c", parsed.Namespace, dirent.Namespace)
	}
	if parsed.EntryType != dirent.EntryType {
		t.Errorf("EntryType = %d, want %d", parsed.EntryType, dirent.EntryType)
	}
	if parsed.ClusterNumber != dirent.ClusterNumber {
		t.Errorf("ClusterNumber = %d, want %d", parsed.ClusterNumber, dirent.ClusterNumber)
	}
	if parsed.BlobNumber != dirent.BlobNumber {
		t.Errorf("BlobNumber = %d, want %d", parsed.BlobNumber, dirent.BlobNumber)
	}
	if parsed.Path != dirent.Path {
		t.Errorf("Path = %q, want %q", parsed.Path, dirent.Path)
	}
	if parsed.Title != dirent.Title {
		t.Errorf("Title = %q, want %q", parsed.Title, dirent.Title)
	}

	if len(data) != bytesRead {
		t.Errorf("bytesRead = %d, want %d", bytesRead, len(data))
	}
}

func TestParseDirentRedirect(t *testing.T) {
	dirent := Dirent{
		MimeTypeIndex: mimeRedirect,
		Namespace:     NamespaceContent,
		EntryType:     EntryTypeRedirect,
		RedirectIndex: 42,
		Path:          "old_page",
		Title:         "Old Page",
	}

	data := EncodeDirent(&dirent)
	parsed, _, err := ParseDirent(data)
	if err != nil {
		t.Fatalf("ParseDirent() error: %v", err)
	}

	if parsed.EntryType != EntryTypeRedirect {
		t.Errorf("EntryType = %d, want EntryTypeRedirect", parsed.EntryType)
	}
	if parsed.RedirectIndex != 42 {
		t.Errorf("RedirectIndex = %d, want 42", parsed.RedirectIndex)
	}
	if !parsed.IsRedirect() {
		t.Error("IsRedirect() should be true")
	}
}

func TestParseDirentLinkTarget(t *testing.T) {
	dirent := Dirent{
		MimeTypeIndex: mimeLinkTarget,
		Namespace:     NamespaceContent,
		EntryType:     EntryTypeLinkTarget,
		ClusterNumber:  1,
		BlobNumber:    0xFFFFFFFF,
		Path:           "target_page",
		Title:          "Target",
	}

	data := EncodeDirent(&dirent)
	parsed, _, err := ParseDirent(data)
	if err != nil {
		t.Fatalf("ParseDirent() error: %v", err)
	}

	if !parsed.IsLinkTarget() {
		t.Error("IsLinkTarget() should be true")
	}
}

func TestParseDirentDeleted(t *testing.T) {
	dirent := Dirent{
		MimeTypeIndex: mimeDeleted,
		Namespace:     NamespaceContent,
		EntryType:     EntryTypeDeleted,
		Path:          "deleted_page",
		Title:         "Deleted",
	}

	data := EncodeDirent(&dirent)
	parsed, _, err := ParseDirent(data)
	if err != nil {
		t.Fatalf("ParseDirent() error: %v", err)
	}

	if !parsed.IsDeleted() {
		t.Error("IsDeleted() should be true")
	}
}

func TestParseDirentWithExtra(t *testing.T) {
	dirent := Dirent{
		MimeTypeIndex: 0,
		Namespace:     NamespaceContent,
		EntryType:     EntryTypeContent,
		ClusterNumber:  1,
		BlobNumber:     2,
		Path:           "test.html",
		Title:          "Test",
		Extra:          []byte{0x01, 0x02, 0x03},
	}

	data := EncodeDirent(&dirent)
	parsed, _, err := ParseDirent(data)
	if err != nil {
		t.Fatalf("ParseDirent() error: %v", err)
	}

	if len(parsed.Extra) != 3 {
		t.Errorf("Extra length = %d, want 3", len(parsed.Extra))
	}
	for i, b := range parsed.Extra {
		if b != byte(i+1) {
			t.Errorf("Extra[%d] = %d, want %d", i, b, i+1)
		}
	}
}

func TestParseDirentShort(t *testing.T) {
	_, _, err := ParseDirent(nil)
	if err == nil {
		t.Error("expected error for nil data")
	}

	_, _, err = ParseDirent(make([]byte, 4))
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestDirentEncodeDecodeRoundTrip(t *testing.T) {
	original := &Dirent{
		MimeTypeIndex: 3,
		Namespace:     NamespaceArticle,
		EntryType:     EntryTypeContent,
		Version:       1,
		ClusterNumber:  99,
		BlobNumber:     12,
		Path:           "A/Home.html",
		Title:          "Home Page Title",
		Extra:          []byte{0xde, 0xad, 0xbe, 0xef},
	}

	encoded := EncodeDirent(original)
	decoded, bytesRead, err := ParseDirent(encoded)
	if err != nil {
		t.Fatalf("ParseDirent() error: %v", err)
	}

	if bytesRead != len(encoded) {
		t.Errorf("bytesRead = %d, want %d", bytesRead, len(encoded))
	}

	// Re-encode and compare.
	reencoded := EncodeDirent(decoded)
	if len(reencoded) != len(encoded) {
		t.Errorf("re-encoded length %d != original length %d", len(reencoded), len(encoded))
	}
}

func TestParseDirentOldNamespace(t *testing.T) {
	data := []byte{
		0x00, 0x00, // mime index
		0x00,                   // extra len
		byte(NamespaceArticle), // namespace
		0x00, 0x00, 0x00, 0x00, // version
		0x05, 0x00, 0x00, 0x00, // cluster number = 5
		0x02, 0x00, 0x00, 0x00, // blob number = 2
		'A', '/', 't', 'e', 's', 't', '.', 'h', 't', 'm', 'l', 0x00, // path
		'T', 'e', 's', 't', ' ', 'P', 'a', 'g', 'e', 0x00, // title
	}

	dirent, _, err := ParseDirent(data)
	if err != nil {
		t.Fatalf("ParseDirent() error: %v", err)
	}

	if dirent.Namespace != NamespaceArticle {
		t.Errorf("Namespace = %c, want A", dirent.Namespace)
	}
	if dirent.Path != "A/test.html" {
		t.Errorf("Path = %q, want \"A/test.html\"", dirent.Path)
	}
	if dirent.Title != "Test Page" {
		t.Errorf("Title = %q, want \"Test Page\"", dirent.Title)
	}
	if dirent.ClusterNumber != 5 {
		t.Errorf("ClusterNumber = %d, want 5", dirent.ClusterNumber)
	}
	if dirent.BlobNumber != 2 {
		t.Errorf("BlobNumber = %d, want 2", dirent.BlobNumber)
	}
}
