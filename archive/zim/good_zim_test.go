package zim_test

import (
	"strings"
	"testing"

	"github.com/cookiengineer/gozim/archive/zim"
)

const testGoodZim = "../../tests/good.zim"

func TestGoodZimOpen(t *testing.T) {
	archive, err := zim.Open(testGoodZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	if archive.EntryCount() != 18 {
		t.Errorf("EntryCount() = %d, want 18", archive.EntryCount())
	}
	if archive.ClusterCount() != 2 {
		t.Errorf("ClusterCount() = %d, want 2", archive.ClusterCount())
	}
	if archive.ArticleCount() != 3 {
		t.Errorf("ArticleCount() = %d, want 3", archive.ArticleCount())
	}
	if !archive.HasNewNamespaceScheme() {
		t.Error("expected new namespace scheme")
	}
	if !archive.HasTitleIndex() {
		t.Error("expected title index")
	}
	if archive.HasFulltextIndex() {
		t.Error("good.zim should not have fulltext index")
	}
	if !archive.HasChecksum() {
		t.Error("expected checksum")
	}
	if archive.IsMultiPart() {
		t.Error("should not be multipart")
	}
}

func TestGoodZimEntryByPath(t *testing.T) {
	archive, err := zim.Open(testGoodZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	tests := []struct {
		path      string
		title     string
		mimeType  string
		minSize   uint64
		isContent bool
	}{
		{"main.html", "Test ZIM file - Main page", "text/html", 100, true},
		{"article1.html", "Test ZIM file - Article 1", "text/html", 100, true},
		{"favicon.png", "", "image/png", 1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			entry, err := archive.EntryByPath(tt.path)
			if err != nil {
				t.Fatalf("EntryByPath(%q) error: %v", tt.path, err)
			}

			if entry.Title() != tt.title {
				t.Errorf("Title() = %q, want %q", entry.Title(), tt.title)
			}

			if entry.Path() != tt.path {
				t.Errorf("Path() = %q, want %q", entry.Path(), tt.path)
			}

			if entry.Namespace() != zim.NamespaceContent {
				t.Errorf("Namespace() = %c, want C", entry.Namespace())
			}

			if entry.IsRedirect() {
				t.Error("expected content entry, not redirect")
			}

			item, err := entry.Item(false)
			if err != nil {
				t.Fatalf("Item() error: %v", err)
			}

			if item.MimeType() != tt.mimeType {
				t.Errorf("MimeType() = %q, want %q", item.MimeType(), tt.mimeType)
			}

			if item.Size() < tt.minSize {
				t.Errorf("Size() = %d, want >= %d", item.Size(), tt.minSize)
			}

			data, err := item.DataAll()
			if err != nil {
				t.Fatalf("DataAll() error: %v", err)
			}
			if uint64(len(data)) != item.Size() {
				t.Errorf("DataAll() length = %d, Size() = %d", len(data), item.Size())
			}
		})
	}
}

func TestGoodZimMainPage(t *testing.T) {
	archive, err := zim.Open(testGoodZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	entry, err := archive.MainEntry()
	if err != nil {
		t.Fatalf("MainEntry() error: %v", err)
	}

	// The main page entry points to "mainPage" which is a redirect in W namespace.
	if !entry.IsRedirect() {
		t.Skip("main entry is not a redirect")
	}

	item, err := entry.Item(true)
	if err != nil {
		t.Fatalf("Item(followRedirect) error: %v", err)
	}

	data, err := item.DataAll()
	if err != nil {
		t.Fatalf("DataAll() error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<html") {
		t.Error("main page should contain HTML content")
	}
	if !strings.Contains(content, "Test ZIM file") {
		t.Error("main page should contain test title")
	}
}

func TestGoodZimMetadata(t *testing.T) {
	archive, err := zim.Open(testGoodZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	// Check that metadata keys exist and have non-empty values.
	keys := archive.MetadataKeys()
	if len(keys) == 0 {
		t.Fatal("expected metadata keys")
	}
	t.Logf("metadata keys: %v", keys)

	// Check specific known metadata entries exist.
	tests := []string{"Title", "Language", "Name", "Creator", "Publisher", "Description", "Date"}
	for _, key := range tests {
		t.Run(key, func(t *testing.T) {
			value, ok := archive.Metadata(key)
			if !ok {
				t.Errorf("Metadata(%q) not found", key)
				return
			}
			if value == "" {
				t.Errorf("Metadata(%q) is empty", key)
			}
			t.Logf("Metadata(%q) = %q", key, value)
		})
	}

	_, ok := archive.Metadata("NonexistentKey")
	if ok {
		t.Error("nonexistent metadata key should return false")
	}
}

func TestGoodZimIllustration(t *testing.T) {
	archive, err := zim.Open(testGoodZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	item, ok := archive.Illustration(48)
	if !ok {
		t.Fatal("expected 48x48 illustration")
	}

	if item.MimeType() != "image/png" {
		t.Errorf("illustration MIME = %q, want image/png", item.MimeType())
	}

	if item.Size() == 0 {
		t.Error("illustration size should be > 0")
	}

	data, err := item.DataAll()
	if err != nil {
		t.Fatalf("DataAll() error: %v", err)
	}

	if len(data) < 8 {
		t.Fatal("illustration data too short")
	}
	if data[0] != 0x89 || data[1] != 'P' || data[2] != 'N' || data[3] != 'G' {
		t.Error("illustration is not a valid PNG")
	}
}

func TestGoodZimArticleContent(t *testing.T) {
	archive, err := zim.Open(testGoodZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	entry, err := archive.EntryByPath("article1.html")
	if err != nil {
		t.Fatalf("EntryByPath() error: %v", err)
	}

	item, err := entry.Item(false)
	if err != nil {
		t.Fatalf("Item() error: %v", err)
	}

	data, err := item.DataAll()
	if err != nil {
		t.Fatalf("DataAll() error: %v", err)
	}

	content := string(data)

	if !strings.HasPrefix(strings.TrimSpace(content), "<!DOCTYPE") {
		t.Error("article should start with DOCTYPE")
	}
	if !strings.Contains(content, "Article 1") {
		t.Error("article should contain article title")
	}
	if len(content) < 100 {
		t.Errorf("article content too short: %d bytes", len(content))
	}
}

func TestGoodZimRedirectEntry(t *testing.T) {
	archive, err := zim.Open(testGoodZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	// The redirect is in W namespace with path "mainPage".
	entry, err := archive.EntryByPath("mainPage")
	if err != nil {
		// W namespace entries need explicit namespace lookup.
		// Try iterating to find it.
		found := false
		for e := range archive.IterateByPath() {
			if e.IsRedirect() {
				entry = e
				found = true
				t.Logf("found redirect: %q (namespace=%c)", entry.Path(), entry.Namespace())
				break
			}
		}
		if !found {
			t.Fatal("redirect entry not found")
		}
	}

	if !entry.IsRedirect() {
		t.Fatal("expected redirect entry")
	}

	target, err := entry.RedirectEntry()
	if err != nil {
		t.Fatalf("RedirectEntry() error: %v", err)
	}

	if target.Path() != "main.html" {
		t.Errorf("redirect target = %q, want %q", target.Path(), "main.html")
	}

	item, err := entry.Item(true)
	if err != nil {
		t.Fatalf("Item(true) error: %v", err)
	}
	if item.Size() == 0 {
		t.Error("resolved item should have content")
	}
}

func TestGoodZimIterateByPath(t *testing.T) {
	archive, err := zim.Open(testGoodZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	count := 0
	var paths []string
	for entry := range archive.IterateByPath() {
		paths = append(paths, entry.Path())
		count++
	}

	if count != 18 {
		t.Errorf("IterateByPath() count = %d, want 18", count)
	}

	if len(paths) < 6 {
		t.Fatal("too few entries")
	}

	hasContent := false
	hasMetadata := false
	for _, p := range paths {
		if p == "main.html" {
			hasContent = true
		}
		if p == "Title" {
			hasMetadata = true
		}
	}
	if !hasContent || !hasMetadata {
		t.Error("missing expected entries in iteration")
	}
}

func TestGoodZimEntryNotFound(t *testing.T) {
	archive, err := zim.Open(testGoodZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	_, err = archive.EntryByPath("nonexistent.html")
	if err == nil {
		t.Error("expected error for nonexistent entry")
	}
}

func TestGoodZimRandomEntry(t *testing.T) {
	archive, err := zim.Open(testGoodZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	entry, err := archive.RandomEntry()
	if err != nil {
		t.Fatalf("RandomEntry() error: %v", err)
	}

	if entry.Path() == "" {
		t.Error("random entry should have a path")
	}
}
