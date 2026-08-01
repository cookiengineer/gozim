package zim

import (
	"testing"
)

func TestParseMimeList(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ml, err := ParseMimeList(nil)
		if err != nil {
			t.Fatal(err)
		}
		if ml.Len() != 0 {
			t.Errorf("Len() = %d, want 0", ml.Len())
		}
	})

	t.Run("single entry", func(t *testing.T) {
		ml, err := ParseMimeList([]byte("text/html\x00"))
		if err != nil {
			t.Fatal(err)
		}
		if ml.Len() != 1 {
			t.Errorf("Len() = %d, want 1", ml.Len())
		}
		if got := ml.MimeType(0); got != "text/html" {
			t.Errorf("MimeType(0) = %q, want %q", got, "text/html")
		}
	})

	t.Run("multiple entries", func(t *testing.T) {
		data := []byte("text/html\x00image/png\x00application/javascript\x00text/css\x00")
		ml, err := ParseMimeList(data)
		if err != nil {
			t.Fatal(err)
		}
		if ml.Len() != 4 {
			t.Errorf("Len() = %d, want 4", ml.Len())
		}
		if got := ml.MimeType(1); got != "image/png" {
			t.Errorf("MimeType(1) = %q, want %q", got, "image/png")
		}
	})

	t.Run("special indices", func(t *testing.T) {
		ml, err := ParseMimeList([]byte("text/plain\x00"))
		if err != nil {
			t.Fatal(err)
		}
		if got := ml.MimeType(0xFFFF); got != "redirect" {
			t.Errorf("MimeType(0xFFFF) = %q, want \"redirect\"", got)
		}
		if got := ml.MimeType(0xFFFE); got != "linktarget" {
			t.Errorf("MimeType(0xFFFE) = %q, want \"linktarget\"", got)
		}
		if got := ml.MimeType(0xFFFD); got != "deleted" {
			t.Errorf("MimeType(0xFFFD) = %q, want \"deleted\"", got)
		}
	})

	t.Run("out of range", func(t *testing.T) {
		ml, err := ParseMimeList([]byte("text/plain\x00"))
		if err != nil {
			t.Fatal(err)
		}
		if got := ml.MimeType(10); got != "" {
			t.Errorf("MimeType(10) = %q, want empty", got)
		}
	})
}

func TestMimeListIndex(t *testing.T) {
	ml, _ := ParseMimeList([]byte("text/html\x00image/png\x00"))

	idx, ok := ml.Index("text/html")
	if !ok {
		t.Errorf("Index(\"text/html\") not found")
	}
	if idx != 0 {
		t.Errorf("Index(\"text/html\") = %d, want 0", idx)
	}

	idx, ok = ml.Index("image/png")
	if !ok {
		t.Errorf("Index(\"image/png\") not found")
	}
	if idx != 1 {
		t.Errorf("Index(\"image/png\") = %d, want 1", idx)
	}

	_, ok = ml.Index("not/found")
	if ok {
		t.Errorf("Index(\"not/found\") should not be found")
	}
}

func TestMimeListAdd(t *testing.T) {
	ml := &MimeList{}

	idx := ml.Add("text/html")
	if idx != 0 {
		t.Errorf("Add(\"text/html\") = %d, want 0", idx)
	}

	idx = ml.Add("image/png")
	if idx != 1 {
		t.Errorf("Add(\"image/png\") = %d, want 1", idx)
	}

	// Adding same should return existing index.
	idx = ml.Add("text/html")
	if idx != 0 {
		t.Errorf("Add(\"text/html\") again = %d, want 0", idx)
	}
}

func TestMimeListEncodeDecode(t *testing.T) {
	original := []string{"text/html", "image/png", "text/css"}
	ml := &MimeList{}
	for _, m := range original {
		ml.Add(m)
	}

	encoded := ml.Encode()
	decoded, err := ParseMimeList(encoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Len() != len(original) {
		t.Errorf("Len() = %d, want %d", decoded.Len(), len(original))
	}

	for i, expected := range original {
		if got := decoded.MimeType(uint16(i)); got != expected {
			t.Errorf("MimeType(%d) = %q, want %q", i, got, expected)
		}
	}
}

func TestMimeListValidate(t *testing.T) {
	ml, _ := ParseMimeList([]byte("text/html\x00image/png\x00"))

	tests := []struct {
		index uint16
		valid bool
	}{
		{0, true},
		{1, true},
		{2, false},
		{0xFFFF, true},
		{0xFFFE, true},
		{0xFFFD, true},
	}

	for _, tt := range tests {
		err := ml.Validate(tt.index)
		if tt.valid && err != nil {
			t.Errorf("Validate(%d) unexpected error: %v", tt.index, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("Validate(%d) expected error, got nil", tt.index)
		}
	}
}
