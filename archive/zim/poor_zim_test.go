package zim_test

import (
	"strings"
	"testing"

	"github.com/cookiengineer/gozim/archive/zim"
)

const testPoorZim = "../../tests/poor.zim"

func TestPoorZimOpen(t *testing.T) {
	archive, err := zim.Open(testPoorZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	if archive.EntryCount() != 25 {
		t.Errorf("EntryCount() = %d, want 25", archive.EntryCount())
	}
	if archive.ArticleCount() != 14 {
		t.Errorf("ArticleCount() = %d, want 14", archive.ArticleCount())
	}
	if !archive.HasNewNamespaceScheme() {
		t.Error("expected new namespace scheme")
	}
}

func TestPoorZimRedirectLoopDetection(t *testing.T) {
	archive, err := zim.Open(testPoorZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	entry, err := archive.EntryByPath("redirect_loop.html")
	if err != nil {
		t.Fatalf("EntryByPath() error: %v", err)
	}

	if !entry.IsRedirect() {
		t.Fatal("expected redirect entry")
	}

	report, err := zim.CheckFile(testPoorZim, zim.CheckRedirectLoops)
	if err != nil {
		t.Fatalf("CheckFile() error: %v", err)
	}

	if len(report.RedirectLoops) == 0 {
		t.Error("expected redirect loops to be detected")
	}

	t.Logf("found %d redirect loops", len(report.RedirectLoops))
}

func TestPoorZimBrokenLinks(t *testing.T) {
	archive, err := zim.Open(testPoorZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	// Verify the dangling link entry exists.
	_, err = archive.EntryByPath("dangling_link.html")
	if err != nil {
		t.Fatalf("dangling_link.html should exist: %v", err)
	}

	// Verify the empty entry exists.
	emptyEntry, err := archive.EntryByPath("empty.html")
	if err != nil {
		t.Fatalf("empty.html should exist: %v", err)
	}

	item, err := emptyEntry.Item(false)
	if err != nil {
		t.Fatalf("Item() error: %v", err)
	}

	if item.Size() != 0 {
		t.Errorf("empty.html size = %d, want 0", item.Size())
	}

	data, err := item.DataAll()
	if err != nil {
		t.Fatalf("DataAll() error: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("empty.html data length = %d, want 0", len(data))
	}
}

func TestPoorZimIntegrityCheck(t *testing.T) {
	report, err := zim.CheckFile(testPoorZim, zim.CheckAll)
	if err != nil {
		t.Fatalf("CheckFile() error: %v", err)
	}

	if len(report.Errors) == 0 {
		t.Error("expected integrity errors in poor.zim")
	}

	for _, errMsg := range report.Errors {
		t.Logf("integrity error: %s", errMsg)
	}

	hasBrokenLinks := len(report.BrokenLinks) > 0
	hasRedirectLoops := len(report.RedirectLoops) > 0

	if !hasBrokenLinks && !hasRedirectLoops {
		t.Error("expected either broken links or redirect loops")
	}

	if hasBrokenLinks {
		t.Logf("found %d broken links", len(report.BrokenLinks))
		for _, link := range report.BrokenLinks {
			t.Logf("  %s → %s (%s)", link.SourceEntry, link.TargetLink, link.Reason)
		}
	}
}

func TestPoorZimExternalLinks(t *testing.T) {
	archive, err := zim.Open(testPoorZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	entries := []string{
		"external_image_http.html",
		"external_image_https.html",
		"external_image_protocol_relative.html",
	}
	for _, path := range entries {
		entry, err := archive.EntryByPath(path)
		if err != nil {
			t.Errorf("EntryByPath(%q) error: %v", path, err)
			continue
		}

		item, err := entry.Item(false)
		if err != nil {
			t.Errorf("Item(%q) error: %v", path, err)
			continue
		}

		data, err := item.DataAll()
		if err != nil {
			t.Errorf("DataAll(%q) error: %v", path, err)
			continue
		}

		content := string(data)
		if !strings.Contains(content, "img") {
			t.Errorf("%s should contain an img tag", path)
		}
	}
}

func TestPoorZimOutOfBoundsLink(t *testing.T) {
	archive, err := zim.Open(testPoorZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	entry, err := archive.EntryByPath("outofbounds_link.html")
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
	if !strings.Contains(content, "../") {
		t.Error("outofbounds_link.html should contain parent directory references")
	}

	t.Logf("outofbounds_link content length: %d", len(content))
}

func TestPoorZimRedundant(t *testing.T) {
	archive, err := zim.Open(testPoorZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	_, err = archive.EntryByPath("redundant_article.html")
	if err != nil {
		t.Fatalf("redundant_article.html should exist: %v", err)
	}

	// There is only one "redundant_article.html" — the other redundant entry
	// needs to be found via iteration. Verify content is non-empty.
	entry, err := archive.EntryByPath("redundant_article.html")
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

	if len(data) == 0 {
		t.Error("redundant article should have content")
	}
	t.Logf("redundant article has %d bytes", len(data))
}
