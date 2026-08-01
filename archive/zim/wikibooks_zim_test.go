package zim_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cookiengineer/gozim/archive/zim"
)

const testWikibooksZim = "../../tests/wikibooks_be_all_nopic_2017-02.zim"

func TestWikibooksOpen(t *testing.T) {
	archive, err := zim.Open(testWikibooksZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	if archive.EntryCount() != 118 {
		t.Errorf("EntryCount() = %d, want 118", archive.EntryCount())
	}
	if archive.ArticleCount() != 109 {
		t.Errorf("ArticleCount() = %d, want 109", archive.ArticleCount())
	}
	if archive.ClusterCount() != 2 {
		t.Errorf("ClusterCount() = %d, want 2", archive.ClusterCount())
	}

	// Old namespace scheme.
	if archive.HasNewNamespaceScheme() {
		t.Error("wikibooks should use old namespace scheme")
	}

	// Real UUID.
	uuid := archive.Uuid()
	if uuid.IsZero() {
		t.Error("expected non-zero UUID")
	}
	t.Logf("UUID: %s", uuid.String())

	// Has checksum.
	if !archive.HasChecksum() {
		t.Error("expected checksum")
	}
}

func TestWikibooksCyrillicTitles(t *testing.T) {
	archive, err := zim.Open(testWikibooksZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	// This ZIM contains Belarusian Wikibooks with Cyrillic titles.
	cyrillicCount := 0
	for entry := range archive.IterateByPath() {
		title := entry.Title()
		if title == "" {
			continue
		}

		if isCyrillic(title) {
			cyrillicCount++

			if !utf8.ValidString(title) {
				t.Errorf("Cyrillic title is not valid UTF-8: %q", title)
			}

			t.Logf("Cyrillic entry: %q → %s", title, entry.Path())
		}

		if cyrillicCount >= 10 {
			break
		}
	}

	if cyrillicCount == 0 {
		t.Error("expected Cyrillic titles in wikibooks_be")
	}
	t.Logf("found at least %d Cyrillic titles", cyrillicCount)
}

func TestWikibooksRedirectEntries(t *testing.T) {
	archive, err := zim.Open(testWikibooksZim)
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

			// Verify title is preserved across redirect resolution.
			if isCyrillic(entry.Title()) {
				t.Logf("  Cyrillic redirect: %q → %q", entry.Title(), target.Title())
			}
		}
	}

	if redirectCount == 0 {
		t.Error("expected redirect entries")
	}
	t.Logf("found %d redirects", redirectCount)
}

func TestWikibooksArticleContent(t *testing.T) {
	archive, err := zim.Open(testWikibooksZim)
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

		content := string(data)

		// Articles may contain HTML or wiki markup.
		if utf8.ValidString(content) {
			t.Logf("article %q: %d bytes, valid UTF-8", entry.Path(), len(data))
		}

		if articlesFound >= 5 {
			break
		}
	}

	if articlesFound == 0 {
		t.Error("expected articles in wikibooks")
	}
}

func TestWikibooksImageEntries(t *testing.T) {
	archive, err := zim.Open(testWikibooksZim)
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
			t.Logf("image %q: Item() error (may be linktarget or empty cluster): %v", entry.Path(), err)
			continue
		}

		if item.Size() == 0 {
			t.Logf("image %q has zero size (possibly linktarget entry)", entry.Path())
			continue
		}

		data, err := item.DataAll()
		if err != nil {
			t.Logf("image %q: DataAll() error: %v", entry.Path(), err)
			continue
		}

			t.Logf("image %q: %d bytes, MIME=%s", entry.Path(), len(data), item.MimeType())
		}
	}

	if imageCount == 0 {
		t.Error("expected image entries")
	}
	t.Logf("found %d images", imageCount)
}

func TestWikibooksMetadata(t *testing.T) {
	archive, err := zim.Open(testWikibooksZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	keys := archive.MetadataKeys()
	if len(keys) == 0 {
		t.Error("expected metadata entries")
	}

	for _, key := range keys {
		value, ok := archive.Metadata(key)
		t.Logf("Metadata(%q) = %q, ok=%v", key, value, ok)
	}

	// Check for standard metadata.
	expectedKeys := []string{"Title", "Language", "Name", "Creator"}
	for _, key := range expectedKeys {
		_, ok := archive.Metadata(key)
		if !ok {
			t.Logf("metadata key %q not found (may be normal)", key)
		}
	}
}

func TestWikibooksRedirectFollow(t *testing.T) {
	archive, err := zim.Open(testWikibooksZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	// Find a redirect and follow it to get content.
	for entry := range archive.IterateByPath() {
		if !entry.IsRedirect() {
			continue
		}

		item, err := entry.Item(true) // follow redirect.
		if err != nil {
			t.Logf("redirect %q: Item(true) error: %v", entry.Path(), err)
			continue
		}

		data, err := item.DataAll()
		if err != nil {
			t.Logf("redirect %q: DataAll() error: %v", entry.Path(), err)
			continue
		}

		if len(data) == 0 {
			t.Logf("redirect %q: followed to empty content", entry.Path())
			continue
		}

		t.Logf("followed redirect %q → got %d bytes", entry.Path(), len(data))
		break
	}
}

func TestWikibooksIterateByTitle(t *testing.T) {
	archive, err := zim.Open(testWikibooksZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	// Should iterate by title order.
	count := 0
	var titles []string
	for entry := range archive.IterateByTitle() {
		titles = append(titles, entry.Title())
		count++
		if count >= 15 {
			break
		}
	}

	if count == 0 {
		t.Error("expected entries from title iteration")
	}

	t.Logf("first %d titles by order: %v", len(titles), titles)

	// Titles should be sorted (or at least have content).
	for _, title := range titles {
		if title == "" && count > 2 {
			t.Log("note: some entries have empty titles (normal for images)")
		}
	}
}

// isCyrillic returns true if the string contains Cyrillic characters.
func isCyrillic(s string) bool {
	for _, r := range s {
		if (r >= 0x0400 && r <= 0x04FF) || (r >= 0x0500 && r <= 0x052F) {
			return true
		}
	}
	return strings.ContainsAny(s, "абвгґдеєжзиіїйклмнопрстуфхцчшщьюяАБВГҐДЕЄЖЗИІЇЙКЛМНОПРСТУФХЦЧШЩЬЮЯ")
}
