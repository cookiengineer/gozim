package zim_test

import (
	"testing"

	"github.com/cookiengineer/gozim/archive/zim"
)

const testBadChecksumZim = "../../tests/bad_checksum.zim"

func TestBadChecksumZimOpen(t *testing.T) {
	archive, err := zim.Open(testBadChecksumZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	if !archive.HasChecksum() {
		t.Error("expected checksum to exist")
	}

	checksum := archive.Checksum()
	if checksum == "" {
		t.Error("checksum string should not be empty")
	}

	// The checksum in this file is all zeros — it should differ from the computed one.
	expectedBad := "00000000000000000000000000000000"
	if checksum != expectedBad {
		t.Errorf("Checksum() = %q, want all zeros %q", checksum, expectedBad)
	}

	// Integrity check should catch the mismatch.
	report, err := zim.CheckFile(testBadChecksumZim, zim.CheckChecksum)
	if err != nil {
		t.Fatalf("CheckFile() error: %v", err)
	}

	if report.ChecksumOK {
		t.Error("checksum validation should fail for bad_checksum.zim")
	}

	// Should have errors about the mismatch.
	hasMismatch := false
	for _, errMsg := range report.Errors {
		if len(errMsg) > 0 {
			hasMismatch = true
			break
		}
	}
	if !hasMismatch {
		t.Error("expected checksum mismatch errors")
	}
}

func TestBadChecksumZimContentIntact(t *testing.T) {
	// Despite checksum corruption, content should still be readable.
	archive, err := zim.Open(testBadChecksumZim)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer archive.Close()

	entry, err := archive.EntryByPath("main.html")
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
		t.Error("content should still be readable despite checksum corruption")
	}
}

func TestBadChecksumZimEntriesMatchGood(t *testing.T) {
	bad, err := zim.Open(testBadChecksumZim)
	if err != nil {
		t.Fatalf("Open(bad) error: %v", err)
	}
	defer bad.Close()

	good, err := zim.Open(testGoodZim)
	if err != nil {
		t.Fatalf("Open(good) error: %v", err)
	}
	defer good.Close()

	// Same entry count.
	if bad.EntryCount() != good.EntryCount() {
		t.Errorf("bad EntryCount = %d, good EntryCount = %d", bad.EntryCount(), good.EntryCount())
	}

	// Same cluster count.
	if bad.ClusterCount() != good.ClusterCount() {
		t.Errorf("bad ClusterCount = %d, good ClusterCount = %d", bad.ClusterCount(), good.ClusterCount())
	}

	// Same article count.
	if bad.ArticleCount() != good.ArticleCount() {
		t.Errorf("bad ArticleCount = %d, good ArticleCount = %d", bad.ArticleCount(), good.ArticleCount())
	}
}
