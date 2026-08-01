package lzma_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/cookiengineer/gozim/compress/lzma"
)

func TestLZMA_RoundTrip_VariousData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single_byte", []byte("x")},
		{"small_text", []byte("Hello, World!")},
		{"medium_text", []byte("The quick brown fox jumps over the lazy dog. This is a test.")},
		{"html", []byte("<!DOCTYPE html><html><head><title>Test</title></head><body><p>Hello World</p></body></html>")},
		{"json", []byte(`{"key":"value","nested":{"array":[1,2,3,4,5,6,7,8,9,10]}}`)},
		{"repeated_short", bytes.Repeat([]byte("abc"), 100)},
		{"repeated_long", bytes.Repeat([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ"), 200)},
		{"binary_mixed", bytes.Repeat([]byte{0x00, 0xFF, 0x55, 0xAA}, 128)},
		{"ascii_article", []byte(strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 50))},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("lzma1_%s", tt.name), func(t *testing.T) {
			var compressed bytes.Buffer
			writer, err := lzma.NewWriter(&compressed)
			if err != nil {
				t.Fatalf("NewWriter error: %v", err)
			}
			_, err = writer.Write(tt.data)
			if err != nil {
				t.Fatalf("Write error: %v", err)
			}
			err = writer.Close()
			if err != nil {
				t.Fatalf("Close error: %v", err)
			}

			if len(tt.data) > 0 {
				t.Logf("lzma1: %d -> %d bytes (%.1f%%)", len(tt.data), compressed.Len(),
					100*float64(compressed.Len())/float64(len(tt.data)))
			}

			reader, err := lzma.NewReader(&compressed)
			if err != nil {
				t.Fatalf("NewReader error: %v", err)
			}
			decompressed, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("ReadAll error: %v", err)
			}

			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("lzma1 round-trip failed: got %d bytes, want %d bytes", len(decompressed), len(tt.data))
			}
		})

		t.Run(fmt.Sprintf("lzma2_%s", tt.name), func(t *testing.T) {
			var compressed bytes.Buffer
			writer, err := lzma.NewWriter2(&compressed)
			if err != nil {
				t.Fatalf("NewWriter2 error: %v", err)
			}
			_, err = writer.Write(tt.data)
			if err != nil {
				t.Fatalf("Write error: %v", err)
			}
			err = writer.Close()
			if err != nil {
				t.Fatalf("Close error: %v", err)
			}

			if len(tt.data) > 0 {
				t.Logf("lzma2: %d -> %d bytes (%.1f%%)", len(tt.data), compressed.Len(),
					100*float64(compressed.Len())/float64(len(tt.data)))
			}

			reader, err := lzma.NewReader2(&compressed)
			if err != nil {
				t.Fatalf("NewReader2 error: %v", err)
			}
			decompressed, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("ReadAll error: %v", err)
			}

			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("lzma2 round-trip failed: got %d bytes, want %d bytes", len(decompressed), len(tt.data))
			}
		})
	}
}

func TestLZMA_VaryingSizes(t *testing.T) {
	sizes := []int{0, 1, 2, 3, 4, 10, 64, 128, 256, 512, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("lzma1_size=%d", size), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)

			var compressed bytes.Buffer
			writer, _ := lzma.NewWriter(&compressed)
			writer.Write(data)
			writer.Close()

			reader, _ := lzma.NewReader(&compressed)
			decompressed, _ := io.ReadAll(reader)

			if !bytes.Equal(decompressed, data) {
				t.Errorf("size %d round-trip failed", size)
			}
		})

		t.Run(fmt.Sprintf("lzma2_size=%d", size), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)

			var compressed bytes.Buffer
			writer, _ := lzma.NewWriter2(&compressed)
			writer.Write(data)
			writer.Close()

			reader, _ := lzma.NewReader2(&compressed)
			decompressed, _ := io.ReadAll(reader)

			if !bytes.Equal(decompressed, data) {
				t.Errorf("size %d round-trip failed", size)
			}
		})
	}
}

func TestLZMA_SHA256(t *testing.T) {
	data := make([]byte, 100000)
	rand.Read(data)
	origHash := sha256.Sum256(data)

	var compressed bytes.Buffer
	writer, _ := lzma.NewWriter(&compressed)
	writer.Write(data)
	writer.Close()

	t.Logf("100KB: compressed to %d bytes (%.1f%%)", compressed.Len(), 100*float64(compressed.Len())/float64(len(data)))

	reader, _ := lzma.NewReader(&compressed)
	decompressed, _ := io.ReadAll(reader)

	decompressedHash := sha256.Sum256(decompressed)
	if decompressedHash != origHash {
		t.Error("SHA256 mismatch after round-trip")
	}
}

func TestLZMA_StreamingWrite(t *testing.T) {
	data := make([]byte, 10000)
	rand.Read(data)

	chunkSizes := []int{1, 3, 7, 13, 64, 128, 500, 1024}
	for _, chunkSize := range chunkSizes {
		t.Run(fmt.Sprintf("chunk=%d", chunkSize), func(t *testing.T) {
			var compressed bytes.Buffer
			writer, _ := lzma.NewWriter(&compressed)

			for i := 0; i < len(data); i += chunkSize {
				end := i + chunkSize
				if end > len(data) {
					end = len(data)
				}
				n, err := writer.Write(data[i:end])
				if err != nil {
					t.Fatalf("Write error at offset %d: %v", i, err)
				}
				if n != end-i {
					t.Errorf("Write returned %d, want %d", n, end-i)
				}
			}
			writer.Close()

			reader, _ := lzma.NewReader(&compressed)
			decompressed, _ := io.ReadAll(reader)

			if !bytes.Equal(decompressed, data) {
				t.Errorf("streaming write chunk=%d round-trip failed", chunkSize)
			}
		})
	}
}

func TestLZMA2_StreamingWrite(t *testing.T) {
	data := make([]byte, 10000)
	rand.Read(data)

	chunkSizes := []int{1, 3, 7, 13, 64, 128}
	for _, chunkSize := range chunkSizes {
		t.Run(fmt.Sprintf("chunk=%d", chunkSize), func(t *testing.T) {
			var compressed bytes.Buffer
			writer, _ := lzma.NewWriter2(&compressed)

			for i := 0; i < len(data); i += chunkSize {
				end := i + chunkSize
				if end > len(data) {
					end = len(data)
				}
				writer.Write(data[i:end])
			}
			writer.Close()

			reader, _ := lzma.NewReader2(&compressed)
			decompressed, _ := io.ReadAll(reader)

			if !bytes.Equal(decompressed, data) {
				t.Errorf("streaming write chunk=%d round-trip failed", chunkSize)
			}
		})
	}
}

func TestLZMA_StreamingRead(t *testing.T) {
	data := make([]byte, 10000)
	rand.Read(data)

	var compressed bytes.Buffer
	writer, _ := lzma.NewWriter(&compressed)
	writer.Write(data)
	writer.Close()

	chunkSizes := []int{1, 3, 13, 64, 128}
	for _, chunkSize := range chunkSizes {
		t.Run(fmt.Sprintf("lzma1_chunk=%d", chunkSize), func(t *testing.T) {
			reader, _ := lzma.NewReader(bytes.NewReader(compressed.Bytes()))
			var decompressed bytes.Buffer
			buf := make([]byte, chunkSize)
			for {
				n, err := reader.Read(buf)
				if n > 0 {
					decompressed.Write(buf[:n])
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("Read error: %v", err)
				}
			}
			if !bytes.Equal(decompressed.Bytes(), data) {
				t.Errorf("streaming read chunk=%d round-trip failed", chunkSize)
			}
		})
	}
}

func TestLZMA_Repeatability(t *testing.T) {
	data := []byte("repeatability test data for lzma compression")

	var firstCompressed []byte
	for i := 0; i < 10; i++ {
		var buf bytes.Buffer
		writer, _ := lzma.NewWriter(&buf)
		writer.Write(data)
		writer.Close()

		if i == 0 {
			firstCompressed = append([]byte{}, buf.Bytes()...)
		} else {
			if !bytes.Equal(buf.Bytes(), firstCompressed) {
				t.Fatalf("compression not repeatable at iteration %d", i)
			}
		}

		reader, _ := lzma.NewReader(&buf)
		decompressed, _ := io.ReadAll(reader)
		if !bytes.Equal(decompressed, data) {
			t.Fatalf("round-trip failed at iteration %d", i)
		}
	}
}

func TestLZMA2_Repeatability(t *testing.T) {
	data := []byte("repeatability test data for lzma2 compression")

	var firstCompressed []byte
	for i := 0; i < 10; i++ {
		var buf bytes.Buffer
		writer, _ := lzma.NewWriter2(&buf)
		writer.Write(data)
		writer.Close()

		if i == 0 {
			firstCompressed = append([]byte{}, buf.Bytes()...)
		} else {
			if !bytes.Equal(buf.Bytes(), firstCompressed) {
				t.Fatalf("compression not repeatable at iteration %d", i)
			}
		}

		reader, _ := lzma.NewReader2(&buf)
		decompressed, _ := io.ReadAll(reader)
		if !bytes.Equal(decompressed, data) {
			t.Fatalf("round-trip failed at iteration %d", i)
		}
	}
}

func TestLZMA_ZIMPatterns(t *testing.T) {
	patterns := []struct {
		name string
		data []byte
	}{
		{
			"article_html",
			[]byte("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"UTF-8\">\n<title>Article Title</title>\n</head>\n<body>\n<article>\n<h1>Article Heading</h1>\n<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</p>\n</article>\n</body>\n</html>"),
		},
		{
			"url_paths",
			[]byte("C/article.html\nA/another_page.html\nC/subdir/index.html\nI/logo.png\n"),
		},
		{
			"mime_list",
			[]byte("text/html\x00application/javascript\x00text/css\x00image/png\x00image/jpeg\x00"),
		},
	}

	for _, tt := range patterns {
		t.Run(fmt.Sprintf("lzma1_%s", tt.name), func(t *testing.T) {
			var compressed bytes.Buffer
			writer, _ := lzma.NewWriter(&compressed)
			writer.Write(tt.data)
			writer.Close()

			reader, _ := lzma.NewReader(&compressed)
			decompressed, _ := io.ReadAll(reader)

			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("lzma1 %s round-trip failed", tt.name)
			}
		})

		t.Run(fmt.Sprintf("lzma2_%s", tt.name), func(t *testing.T) {
			var compressed bytes.Buffer
			writer, _ := lzma.NewWriter2(&compressed)
			writer.Write(tt.data)
			writer.Close()

			reader, _ := lzma.NewReader2(&compressed)
			decompressed, _ := io.ReadAll(reader)

			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("lzma2 %s round-trip failed", tt.name)
			}
		})
	}
}

func TestLZMA_LargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large data test in short mode")
	}

	data := make([]byte, 1<<20) // 1 MB
	rand.Read(data)

	var compressed bytes.Buffer
	writer, _ := lzma.NewWriter(&compressed)
	writer.Write(data)
	writer.Close()

	t.Logf("1MB lzma1: compressed to %d bytes (%.1f%%)", compressed.Len(), 100*float64(compressed.Len())/float64(len(data)))

	reader, _ := lzma.NewReader(&compressed)
	decompressed, _ := io.ReadAll(reader)

	if !bytes.Equal(decompressed, data) {
		t.Error("large data round-trip failed")
	}
}

func TestLZMA2_LargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large data test in short mode")
	}

	data := make([]byte, 1<<20) // 1 MB
	rand.Read(data)

	var compressed bytes.Buffer
	writer, _ := lzma.NewWriter2(&compressed)
	writer.Write(data)
	writer.Close()

	t.Logf("1MB lzma2: compressed to %d bytes (%.1f%%)", compressed.Len(), 100*float64(compressed.Len())/float64(len(data)))

	reader, _ := lzma.NewReader2(&compressed)
	decompressed, _ := io.ReadAll(reader)

	if !bytes.Equal(decompressed, data) {
		t.Error("large data round-trip failed")
	}
}

func TestLZMA_ReaderRead_ShortBuffer(t *testing.T) {
	data := bytes.Repeat([]byte("ABCDEFGHIJ"), 100)
	var compressed bytes.Buffer
	writer, _ := lzma.NewWriter(&compressed)
	writer.Write(data)
	writer.Close()

	reader, _ := lzma.NewReader(&compressed)

	var decompressed bytes.Buffer
	buf := make([]byte, 1) // single byte reads
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			decompressed.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
	}

	if !bytes.Equal(decompressed.Bytes(), data) {
		t.Error("single-byte read round-trip failed")
	}
}

func TestLZMA_NewReader_InvalidData(t *testing.T) {
	invalidData := [][]byte{
		nil,
		{},
		{0x00},
		[]byte("not valid lzma data at all"),
	}

	for i, d := range invalidData {
		_, err := lzma.NewReader(bytes.NewReader(d))
		if err == nil {
			t.Errorf("case %d: expected error for invalid data", i)
		}
	}
}

func TestLZMA_CrossFormatCompatibility(t *testing.T) {
	// LZMA1 compressed data should NOT be decodable by LZMA2 reader
	data := []byte("cross format test data")
	var compressed bytes.Buffer
	writer, _ := lzma.NewWriter(&compressed)
	writer.Write(data)
	writer.Close()

	// LZMA2 reader should fail on LZMA1 data
	_, err := lzma.NewReader2(&compressed)
	if err == nil {
		t.Log("LZMA2 reader accepted LZMA1 data (unusual but possible)")
	}
}

func TestLZMA_WriteRead_Empty(t *testing.T) {
	t.Run("lzma1", func(t *testing.T) {
		var compressed bytes.Buffer
		writer, _ := lzma.NewWriter(&compressed)
		writer.Close()

		reader, err := lzma.NewReader(&compressed)
		if err != nil {
			t.Fatalf("NewReader error: %v", err)
		}
		decompressed, _ := io.ReadAll(reader)
		if len(decompressed) != 0 {
			t.Errorf("expected 0 bytes, got %d", len(decompressed))
		}
	})
	t.Run("lzma2", func(t *testing.T) {
		var compressed bytes.Buffer
		writer, _ := lzma.NewWriter2(&compressed)
		writer.Close()

		reader, err := lzma.NewReader2(&compressed)
		if err != nil {
			t.Fatalf("NewReader2 error: %v", err)
		}
		decompressed, _ := io.ReadAll(reader)
		if len(decompressed) != 0 {
			t.Errorf("expected 0 bytes, got %d", len(decompressed))
		}
	})
}

func BenchmarkLZMA1_RoundTrip_4K(b *testing.B) {
	data := make([]byte, 4096)
	rand.Read(data)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w, _ := lzma.NewWriter(&buf)
		w.Write(data)
		w.Close()
		r, _ := lzma.NewReader(&buf)
		io.ReadAll(r)
	}
}

func BenchmarkLZMA2_RoundTrip_4K(b *testing.B) {
	data := make([]byte, 4096)
	rand.Read(data)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w, _ := lzma.NewWriter2(&buf)
		w.Write(data)
		w.Close()
		r, _ := lzma.NewReader2(&buf)
		io.ReadAll(r)
	}
}

func FuzzLZMA1_RoundTrip(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("hello"))
	f.Add(bytes.Repeat([]byte("abc"), 100))
	f.Fuzz(func(t *testing.T, data []byte) {
		var compressed bytes.Buffer
		writer, err := lzma.NewWriter(&compressed)
		if err != nil {
			t.Fatalf("NewWriter error: %v", err)
		}
		_, err = writer.Write(data)
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		err = writer.Close()
		if err != nil {
			t.Fatalf("Close error: %v", err)
		}

		reader, err := lzma.NewReader(&compressed)
		if err != nil {
			t.Fatalf("NewReader error: %v", err)
		}
		decompressed, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll error: %v", err)
		}

		if !bytes.Equal(decompressed, data) {
			t.Fatalf("round-trip failed")
		}
	})
}

func FuzzLZMA2_RoundTrip(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, data []byte) {
		var compressed bytes.Buffer
		writer, err := lzma.NewWriter2(&compressed)
		if err != nil {
			t.Fatalf("NewWriter2 error: %v", err)
		}
		_, err = writer.Write(data)
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		err = writer.Close()
		if err != nil {
			t.Fatalf("Close error: %v", err)
		}

		reader, err := lzma.NewReader2(&compressed)
		if err != nil {
			t.Fatalf("NewReader2 error: %v", err)
		}
		decompressed, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll error: %v", err)
		}

		if !bytes.Equal(decompressed, data) {
			t.Fatalf("round-trip failed")
		}
	})
}
