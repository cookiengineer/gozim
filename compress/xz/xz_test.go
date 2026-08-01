package xz_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/cookiengineer/gozim/compress/xz"
)

func TestXZDecompress(t *testing.T) {
	// Test decompressing a simple xz stream.
	// We'll use xz.NewWriter to compress, then NewReader to decompress.
	data := []byte("Hello, XZ compressed world!\nThis is a test of the xz decompression library.")

	var compressed bytes.Buffer
	writer, err := xz.NewWriter(&compressed)
	if err != nil {
		t.Fatalf("NewWriter error: %v", err)
	}
	_, err = writer.Write(data)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	writer.Close()

	t.Logf("compressed %d → %d bytes (%.1f%%)", len(data), compressed.Len(), 100*float64(compressed.Len())/float64(len(data)))

	reader, err := xz.NewReader(&compressed)
	if err != nil {
		t.Fatalf("NewReader error: %v", err)
	}

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("round-trip failed: got %d bytes, want %d bytes", len(decompressed), len(data))
		t.Errorf("got:  %q", truncate(decompressed, 100))
		t.Errorf("want: %q", truncate(data, 100))
	}
}

func TestXZDecompressEmpty(t *testing.T) {
	var compressed bytes.Buffer
	writer, _ := xz.NewWriter(&compressed)
	writer.Close()

	reader, err := xz.NewReader(&compressed)
	if err != nil {
		t.Fatal(err)
	}
	decompressed, _ := io.ReadAll(reader)
	if len(decompressed) != 0 {
		t.Errorf("expected 0 bytes, got %d", len(decompressed))
	}
}

func TestXZDecompressLarge(t *testing.T) {
	data := bytes.Repeat([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ"), 2000)

	var compressed bytes.Buffer
	writer, _ := xz.NewWriter(&compressed)
	writer.Write(data)
	writer.Close()

	t.Logf("compressed %d → %d bytes (%.1f%%)", len(data), compressed.Len(), 100*float64(compressed.Len())/float64(len(data)))

	reader, _ := xz.NewReader(&compressed)
	decompressed, _ := io.ReadAll(reader)

	if !bytes.Equal(decompressed, data) {
		t.Errorf("large data round-trip failed")
	}
}

func TestXZDecompressMultiBlock(t *testing.T) {
	data := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. "), 500)

	var compressed bytes.Buffer
	writer, _ := xz.NewWriter(&compressed)
	writer.Write(data)
	writer.Close()

	reader, _ := xz.NewReader(&compressed)

	var result bytes.Buffer
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
	}

	if !bytes.Equal(result.Bytes(), data) {
		t.Errorf("multi-block read failed")
	}
}

func TestXZDecompressZIMPattern(t *testing.T) {
	// Patterns commonly found in ZIM files (HTML content).
	htmlContent := []byte(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Article Title</title>
<link rel="stylesheet" href="style.css">
</head>
<body>
<h1>Article Title</h1>
<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</p>
<p>Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.</p>
</body>
</html>`)

	var compressed bytes.Buffer
	writer, _ := xz.NewWriter(&compressed)
	writer.Write(htmlContent)
	writer.Close()

	reader, _ := xz.NewReader(&compressed)
	decompressed, _ := io.ReadAll(reader)

	if !bytes.Equal(decompressed, htmlContent) {
		t.Errorf("ZIM HTML pattern round-trip failed")
	}
}

func TestXZWriterConfig(t *testing.T) {
	data := []byte("test with custom config")

	var compressed bytes.Buffer
	writer, err := xz.NewWriter(&compressed)
	if err != nil {
		t.Fatalf("NewWriter error: %v", err)
	}
	writer.Write(data)
	writer.Close()

	reader, _ := xz.NewReader(&compressed)
	decompressed, _ := io.ReadAll(reader)

	if !bytes.Equal(decompressed, data) {
		t.Errorf("config round-trip failed")
	}
}

func TestXZReaderInvalid(t *testing.T) {
	_, err := xz.NewReader(bytes.NewReader([]byte("not a valid xz stream")))
	if err == nil {
		t.Error("expected error for invalid xz data")
	}
}

func truncate(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return append(b[:n], []byte("...")...)
}
