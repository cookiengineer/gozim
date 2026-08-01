package lzma_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/cookiengineer/gozim/compress/lzma"
)

func TestLZMADecode(t *testing.T) {
	data := []byte("Hello, LZMA compressed world! This is a test of the lzma library.")

	var compressed bytes.Buffer
	writer, err := lzma.NewWriter(&compressed)
	if err != nil {
		t.Fatalf("NewWriter error: %v", err)
	}
	_, err = writer.Write(data)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	writer.Close()

	t.Logf("compressed %d → %d bytes (%.1f%%)", len(data), compressed.Len(), 100*float64(compressed.Len())/float64(len(data)))

	reader, err := lzma.NewReader(&compressed)
	if err != nil {
		t.Fatalf("NewReader error: %v", err)
	}

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("round-trip failed: got %d bytes, want %d bytes", len(decompressed), len(data))
	}
}

func TestLZMADecodeLarge(t *testing.T) {
	data := bytes.Repeat([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ"), 500)

	var compressed bytes.Buffer
	writer, _ := lzma.NewWriter(&compressed)
	writer.Write(data)
	writer.Close()

	reader, _ := lzma.NewReader(&compressed)
	decompressed, _ := io.ReadAll(reader)

	if !bytes.Equal(decompressed, data) {
		t.Errorf("large data round-trip failed")
	}
}

func TestLZMADecodeEmpty(t *testing.T) {
	var compressed bytes.Buffer
	writer, _ := lzma.NewWriter(&compressed)
	writer.Close()

	reader, err := lzma.NewReader(&compressed)
	if err != nil {
		t.Fatal(err)
	}
	decompressed, _ := io.ReadAll(reader)
	if len(decompressed) != 0 {
		t.Errorf("expected 0 bytes, got %d", len(decompressed))
	}
}

func TestLZMAWriterFormats(t *testing.T) {
	data := []byte("test with different lzma formats")

	tests := []struct {
		name       string
		writerFn   func(io.Writer) (io.WriteCloser, error)
		readerFn   func(io.Reader) (io.Reader, error)
	}{
		{
			"lzma1",
			func(w io.Writer) (io.WriteCloser, error) { return lzma.NewWriter(w) },
			func(r io.Reader) (io.Reader, error) { return lzma.NewReader(r) },
		},
		{
			"lzma2",
			func(w io.Writer) (io.WriteCloser, error) { return lzma.NewWriter2(w) },
			func(r io.Reader) (io.Reader, error) { return lzma.NewReader2(r) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var compressed bytes.Buffer
			wc, err := tt.writerFn(&compressed)
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			wc.Write(data)
			wc.Close()

			reader, err := tt.readerFn(&compressed)
			if err != nil {
				t.Fatalf("reader error: %v", err)
			}
			decompressed, _ := io.ReadAll(reader)

			if !bytes.Equal(decompressed, data) {
				t.Errorf("%s round-trip failed", tt.name)
			}
			t.Logf("%s: %d → %d bytes (%.1f%%)", tt.name, len(data), compressed.Len(), 100*float64(compressed.Len())/float64(len(data)))
		})
	}
}

func TestLZMAReaderInvalid(t *testing.T) {
	_, err := lzma.NewReader(bytes.NewReader([]byte("invalid lzma data")))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}
