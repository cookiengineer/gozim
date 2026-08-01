package zstd_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/cookiengineer/gozim/compress/zstd"
)

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("hello world")},
		{"medium", bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 10)},
		{"compressible", bytes.Repeat([]byte("aaaaaaaaaaaaaaaaaaaaaaa"), 100)},
		{"html", []byte("<!DOCTYPE html><html><head><title>Test</title></head><body><p>Hello World</p></body></html>")},
		{"json", []byte(`{"key":"value","nested":{"array":[1,2,3,4,5]}}`)},
		{"random_1k", randomBytes(1024)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed := compressData(t, tt.data)
			decompressed := decompressData(t, compressed)

			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("round-trip failed: got %d bytes, want %d bytes", len(decompressed), len(tt.data))
				if len(tt.data) < 200 {
					t.Errorf("  want: %q", tt.data)
					t.Errorf("  got:  %q", decompressed)
				}
			}

			if len(tt.data) > 100 {
				t.Logf("compression ratio: %.1f%% (%d → %d bytes)", 100*float64(len(compressed))/float64(len(tt.data)), len(tt.data), len(compressed))
			}
		})
	}
}

func TestCompressDecompressStream(t *testing.T) {
	for _, size := range []int{0, 1, 10, 100, 1000, 10000} {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			data := randomBytes(size)

			var compressedBuf bytes.Buffer
			encoder, err := zstd.NewWriter(&compressedBuf)
			if err != nil {
				t.Fatalf("NewWriter error: %v", err)
			}

			_, err = encoder.Write(data)
			if err != nil {
				t.Fatalf("Write error: %v", err)
			}

			err = encoder.Close()
			if err != nil {
				t.Fatalf("Close error: %v", err)
			}

			decoder, err := zstd.NewReader(&compressedBuf)
			if err != nil {
				t.Fatalf("NewReader error: %v", err)
			}
			defer decoder.Close()

			decompressed, err := io.ReadAll(decoder)
			if err != nil {
				t.Fatalf("ReadAll error: %v", err)
			}

			if !bytes.Equal(decompressed, data) {
				t.Errorf("stream round-trip failed: got %d bytes, want %d bytes", len(decompressed), len(data))
			}
		})
	}
}

func TestMultiBlockCompress(t *testing.T) {
	// Test data large enough to require multiple blocks.
	data := bytes.Repeat([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ"), 5000)

	var buf bytes.Buffer
	encoder, err := zstd.NewWriter(&buf,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
	)
	if err != nil {
		t.Fatalf("NewWriter error: %v", err)
	}

	_, err = encoder.Write(data)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	encoder.Close()

	decoder, err := zstd.NewReader(&buf)
	if err != nil {
		t.Fatalf("NewReader error: %v", err)
	}
	defer decoder.Close()

	decompressed, err := io.ReadAll(decoder)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("multi-block round-trip failed: got %d bytes, want %d bytes", len(decompressed), len(data))
	}
}

func TestCompressLevels(t *testing.T) {
	data := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 100)

	for _, level := range []zstd.EncoderLevel{
		zstd.SpeedFastest,
		zstd.SpeedDefault,
		zstd.SpeedBetterCompression,
	} {
		t.Run(level.String(), func(t *testing.T) {
			var buf bytes.Buffer
			encoder, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(level))
			if err != nil {
				t.Fatalf("NewWriter error: %v", err)
			}
			encoder.Write(data)
			encoder.Close()

			decoder, _ := zstd.NewReader(&buf)
			decompressed, _ := io.ReadAll(decoder)
			decoder.Close()

			if !bytes.Equal(decompressed, data) {
				t.Errorf("level %s round-trip failed", level)
			}
			t.Logf("level %s: %d → %d bytes (%.1f%%)", level, len(data), buf.Len(), 100*float64(buf.Len())/float64(len(data)))
		})
	}
}

func TestDecodeAll(t *testing.T) {
	data := []byte("quick test data for EncodeAll/DecodeAll")
	encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	compressed := encoder.EncodeAll(data, nil)
	encoder.Close()

	decoder, _ := zstd.NewReader(nil)
	decompressed, err := decoder.DecodeAll(compressed, nil)
	decoder.Close()
	if err != nil {
		t.Fatalf("DecodeAll error: %v", err)
	}

	if !bytes.Equal(decompressed, data) {
		t.Errorf("EncodeAll/DecodeAll round-trip failed: got %q, want %q", decompressed, data)
	}
}

func TestLargeData(t *testing.T) {
	t.Skip("skipping large data test in short mode")

	data := randomBytes(10 * 1024 * 1024) // 10 MB

	var buf bytes.Buffer
	encoder, _ := zstd.NewWriter(&buf)
	encoder.Write(data)
	encoder.Close()

	decoder, _ := zstd.NewReader(&buf)
	decompressed, _ := io.ReadAll(decoder)
	decoder.Close()

	if !bytes.Equal(decompressed, data) {
		t.Errorf("large data round-trip failed")
	}
}

func TestZeroLength(t *testing.T) {
	var buf bytes.Buffer
	encoder, _ := zstd.NewWriter(&buf)
	encoder.Close()

	decoder, _ := zstd.NewReader(&buf)
	defer decoder.Close()

	decompressed, err := io.ReadAll(decoder)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}

	if len(decompressed) != 0 {
		t.Errorf("expected 0 bytes, got %d", len(decompressed))
	}
}

func TestRepeatedZIMPattern(t *testing.T) {
	// Test with patterns commonly found in ZIM files.
	patterns := []string{
		"<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"UTF-8\">\n<title>Article</title>\n</head>\n<body>\n<article>\n<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit.</p>\n</article>\n</body>\n</html>\n",
		"C/article.html",
		"Test ZIM file - Main page",
	}

	for _, p := range patterns {
		data := []byte(strings.Repeat(p, 3))
		var buf bytes.Buffer
		encoder, _ := zstd.NewWriter(&buf)
		encoder.Write(data)
		encoder.Close()

		decoder, _ := zstd.NewReader(&buf)
		decompressed, _ := io.ReadAll(decoder)
		decoder.Close()

		if !bytes.Equal(decompressed, data) {
			t.Errorf("ZIM pattern round-trip failed for %q", truncateStr(p, 50))
		}
	}
}

func compressData(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	encoder, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		t.Fatalf("NewWriter error: %v", err)
	}
	_, err = encoder.Write(data)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	err = encoder.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}
	return buf.Bytes()
}

func decompressData(t *testing.T, data []byte) []byte {
	t.Helper()
	decoder, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewReader error: %v", err)
	}
	defer decoder.Close()

	result, err := io.ReadAll(decoder)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}
	return result
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
