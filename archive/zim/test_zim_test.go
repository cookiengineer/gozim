package zim_test

import (
	"strings"
	"testing"

	"github.com/cookiengineer/gozim/archive/zim"
)

const testTestZim = "../../tests/test.zim"

func TestTestZimOpen(t *testing.T) {
	archive, err := zim.Open(testTestZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	if archive.EntryCount() != 231 {
		t.Errorf("EntryCount() = %d, want 231", archive.EntryCount())
	}
	if archive.ArticleCount() != 224 {
		t.Errorf("ArticleCount() = %d, want 224", archive.ArticleCount())
	}
	if archive.MediaCount() != 7 {
		t.Errorf("MediaCount() = %d, want 7", archive.MediaCount())
	}
	if archive.ClusterCount() != 42 {
		t.Errorf("ClusterCount() = %d, want 42", archive.ClusterCount())
	}

	if archive.HasNewNamespaceScheme() {
		t.Error("test.zim should use old namespace scheme")
	}

	if !archive.HasChecksum() {
		t.Error("expected checksum")
	}

	uuid := archive.Uuid()
	if uuid.IsZero() {
		t.Error("expected non-zero UUID")
	}
	t.Logf("UUID: %s", uuid.String())
}

func TestTestZimOldNamespaceEntries(t *testing.T) {
	archive, err := zim.Open(testTestZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	// In old namespace scheme, paths include namespace prefix.
	// Find an article entry.
	var articleEntry *zim.Entry
	for entry := range archive.IterateByPath() {
		if entry.Namespace() == zim.NamespaceArticle && !entry.IsRedirect() {
			articleEntry = entry
			break
		}
	}

	if articleEntry == nil {
		t.Fatal("no article entry found")
	}

	if articleEntry.Namespace() != zim.NamespaceArticle {
		t.Errorf("expected namespace A, got %c", articleEntry.Namespace())
	}

	if !strings.Contains(articleEntry.Path(), ".") {
		t.Errorf("article path should contain extension: %q", articleEntry.Path())
	}

	t.Logf("found article: %q (namespace=%c)", articleEntry.Path(), articleEntry.Namespace())
}

func TestTestZimImageEntries(t *testing.T) {
	archive, err := zim.Open(testTestZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	imageCount := 0
	for entry := range archive.IterateByPath() {
		if entry.Namespace() == zim.NamespaceImage {
			imageCount++

			item, err := entry.Item(false)
			if err != nil {
				t.Errorf("Item(%q) error: %v", entry.Path(), err)
				continue
			}

			if item.Size() == 0 {
				t.Errorf("image %q has zero size", entry.Path())
				continue
			}

			data, err := item.DataAll()
			if err != nil {
				t.Errorf("DataAll(%q) error: %v", entry.Path(), err)
				continue
			}

			if len(data) < 4 {
				t.Errorf("image %q too short: %d bytes", entry.Path(), len(data))
				continue
			}

			mimeType := item.MimeType()
			if strings.HasPrefix(mimeType, "image/png") {
				if data[0] != 0x89 || data[1] != 'P' || data[2] != 'N' || data[3] != 'G' {
					t.Logf("image %q: not standard PNG magic: %x", entry.Path(), data[:4])
				}
			}
		}
	}

	if imageCount == 0 {
		t.Error("expected image entries in test.zim")
	}
	t.Logf("found %d image entries", imageCount)
}

func TestTestZimArticleContent(t *testing.T) {
	archive, err := zim.Open(testTestZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	articlesFound := 0
	for entry := range archive.IterateByPath() {
		if entry.Namespace() != zim.NamespaceArticle {
			continue
		}
		if entry.IsRedirect() {
			continue
		}

		articlesFound++

		item, err := entry.Item(false)
		if err != nil {
			t.Errorf("Item(%q) error: %v", entry.Path(), err)
			continue
		}

		data, err := item.DataAll()
		if err != nil {
			t.Errorf("DataAll(%q) error: %v", entry.Path(), err)
			continue
		}

		if len(data) == 0 {
			t.Errorf("article %q has empty content", entry.Path())
		}

		if len(data) < 20 {
			t.Errorf("article %q content too short: %d bytes", entry.Path(), len(data))
		}

		if articlesFound >= 5 {
			break
		}
	}

	if articlesFound == 0 {
		t.Error("expected articles in test.zim")
	}
	t.Logf("checked %d articles", articlesFound)
}

func TestTestZimMetadata(t *testing.T) {
	archive, err := zim.Open(testTestZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	keys := archive.MetadataKeys()
	if len(keys) == 0 {
		t.Error("expected metadata entries")
	}

	t.Logf("metadata keys: %v", keys)

	for _, key := range keys {
		value, ok := archive.Metadata(key)
		t.Logf("Metadata(%q) = %q, ok=%v", key, value, ok)
	}
}

func TestTestZimIterateByTitle(t *testing.T) {
	archive, err := zim.Open(testTestZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	count := 0
	var titles []string
	for entry := range archive.IterateByTitle() {
		titles = append(titles, entry.Title())
		count++
		if count >= 10 {
			break
		}
	}

	if count == 0 {
		t.Error("expected entries from title iteration")
	}

	t.Logf("first %d titles: %v", len(titles), titles)
}

func TestTestZimRedirectEntry(t *testing.T) {
	archive, err := zim.Open(testTestZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	redirectCount := 0
	for entry := range archive.IterateByPath() {
		if entry.IsRedirect() {
			redirectCount++
			target, err := entry.RedirectEntry()
			if err != nil {
				t.Errorf("RedirectEntry(%q) error: %v", entry.Path(), err)
				continue
			}
			t.Logf("redirect: %q → %q", entry.Path(), target.Path())
		}
	}

	if redirectCount == 0 {
		t.Error("expected redirect entries in test.zim")
	}
	t.Logf("found %d redirects", redirectCount)
}
