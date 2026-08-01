package zstd_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/cookiengineer/gozim/compress/zstd"
)

func TestRoundTrip_SHA256(t *testing.T) {
	sizes := []int{0, 1, 10, 100, 1000, 10000, 100000}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("size=%d", size), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)

			origHash := sha256.Sum256(data)

			var compressed bytes.Buffer
			encoder, err := zstd.NewWriter(&compressed)
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

			t.Logf("size=%d: %d -> %d bytes (%.1f%%)", size, len(data), compressed.Len(),
				100*float64(compressed.Len())/float64(max(len(data), 1)))

			decoder, err := zstd.NewReader(&compressed)
			if err != nil {
				t.Fatalf("NewReader error: %v", err)
			}
			defer decoder.Close()

			decompressed, err := io.ReadAll(decoder)
			if err != nil {
				t.Fatalf("ReadAll error: %v", err)
			}

			decompressedHash := sha256.Sum256(decompressed)
			if decompressedHash != origHash {
				t.Errorf("SHA256 mismatch for size %d", size)
			}
		})
	}
}

func TestStreamingWrite(t *testing.T) {
	chunkSizes := []int{1, 3, 7, 13, 64, 128, 1024}
	for _, chunkSize := range chunkSizes {
		t.Run(fmt.Sprintf("chunk=%d", chunkSize), func(t *testing.T) {
			data := make([]byte, 5000)
			rand.Read(data)

			var compressed bytes.Buffer
			encoder, err := zstd.NewWriter(&compressed)
			if err != nil {
				t.Fatalf("NewWriter error: %v", err)
			}

			for i := 0; i < len(data); i += chunkSize {
				end := i + chunkSize
				if end > len(data) {
					end = len(data)
				}
				n, err := encoder.Write(data[i:end])
				if err != nil {
					t.Fatalf("Write error at offset %d: %v", i, err)
				}
				if n != end-i {
					t.Errorf("Write returned %d, want %d", n, end-i)
				}
			}
			err = encoder.Close()
			if err != nil {
				t.Fatalf("Close error: %v", err)
			}

			decoder, err := zstd.NewReader(&compressed)
			if err != nil {
				t.Fatalf("NewReader error: %v", err)
			}
			defer decoder.Close()

			decompressed, err := io.ReadAll(decoder)
			if err != nil {
				t.Fatalf("ReadAll error: %v", err)
			}

			if !bytes.Equal(decompressed, data) {
				t.Errorf("streaming write round-trip failed")
			}
		})
	}
}

func TestStreamingRead(t *testing.T) {
	chunkSizes := []int{1, 3, 13, 64, 128}
	for _, chunkSize := range chunkSizes {
		t.Run(fmt.Sprintf("chunk=%d", chunkSize), func(t *testing.T) {
			data := make([]byte, 5000)
			rand.Read(data)

			var compressed bytes.Buffer
			encoder, _ := zstd.NewWriter(&compressed)
			encoder.Write(data)
			encoder.Close()

			decoder, _ := zstd.NewReader(&compressed)
			defer decoder.Close()

			var decompressed bytes.Buffer
			buf := make([]byte, chunkSize)
			for {
				n, err := decoder.Read(buf)
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
				t.Errorf("streaming read round-trip failed")
			}
		})
	}
}

func TestAllEncoderLevels(t *testing.T) {
	data := []byte(strings.Repeat("the quick brown fox jumps over the lazy dog. ", 50))

	levels := []zstd.EncoderLevel{
		zstd.SpeedFastest,
		zstd.SpeedDefault,
		zstd.SpeedBetterCompression,
		zstd.SpeedBestCompression,
	}

	for _, level := range levels {
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
			t.Logf("level %s: %d -> %d bytes (%.1f%%)", level, len(data), buf.Len(),
				100*float64(buf.Len())/float64(len(data)))
		})
	}
}

func TestWithWindowSize(t *testing.T) {
	data := bytes.Repeat([]byte("window size test data for zstd compression "), 200)

	windowSizes := []int{1 << 18, 1 << 19, 1 << 20}
	for _, ws := range windowSizes {
		t.Run(fmt.Sprintf("window=%d", ws), func(t *testing.T) {
			var buf bytes.Buffer
			encoder, err := zstd.NewWriter(&buf,
				zstd.WithEncoderLevel(zstd.SpeedDefault),
				zstd.WithWindowSize(ws),
			)
			if err != nil {
				t.Fatalf("NewWriter error: %v", err)
			}
			encoder.Write(data)
			encoder.Close()

			decoder, _ := zstd.NewReader(&buf)
			decompressed, _ := io.ReadAll(decoder)
			decoder.Close()

			if !bytes.Equal(decompressed, data) {
				t.Errorf("window=%d round-trip failed", ws)
			}
			t.Logf("window=%d: %d -> %d bytes", ws, len(data), buf.Len())
		})
	}
}

func TestEncodeAll_DecodeAll_Varying(t *testing.T) {
	inputs := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single", []byte("x")},
		{"small", []byte("hello world")},
		{"medium", bytes.Repeat([]byte("ABCDEFGH"), 1000)},
		{"html", []byte("<!DOCTYPE html><html><head><title>Test</title></head><body><p>Content</p></body></html>")},
		{"json", []byte(`{"articles":[{"title":"One","body":"Lorem ipsum"},{"title":"Two","body":"Dolor sit"}]}`)},
	}

	for _, tt := range inputs {
		t.Run(tt.name, func(t *testing.T) {
			encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
			compressed := encoder.EncodeAll(tt.data, nil)
			encoder.Close()

			decoder, _ := zstd.NewReader(nil)
			decompressed, err := decoder.DecodeAll(compressed, nil)
			decoder.Close()
			if err != nil {
				t.Fatalf("DecodeAll error: %v", err)
			}

			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("EncodeAll/DecodeAll round-trip failed")
			}
		})
	}
}

func TestEncodeAll_SameEncoder(t *testing.T) {
	dataSets := [][]byte{
		[]byte("first data block for encode all"),
		[]byte("second block, completely different"),
		[]byte("third block, also different"),
	}

	encoder, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	defer encoder.Close()

	for i, data := range dataSets {
		compressed := encoder.EncodeAll(data, nil)

		decoder, _ := zstd.NewReader(nil)
		decompressed, err := decoder.DecodeAll(compressed, nil)
		decoder.Close()
		if err != nil {
			t.Fatalf("block %d DecodeAll error: %v", i, err)
		}

		if !bytes.Equal(decompressed, data) {
			t.Errorf("block %d round-trip failed", i)
		}
	}
}

func TestZIMPatterns(t *testing.T) {
	patterns := []struct {
		name string
		data []byte
	}{
		{
			"article_html",
			[]byte("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"UTF-8\">\n<title>Article Title</title>\n</head>\n<body>\n<article>\n<h1>Article Heading</h1>\n<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.</p>\n<p>Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.</p>\n</article>\n</body>\n</html>"),
		},
		{
			"url_paths",
			[]byte(strings.Repeat("C/article.html\nC/another_page.html\nC/subdir/index.html\n", 10)),
		},
		{
			"mime_list",
			[]byte("text/html\x00application/javascript\x00text/css\x00image/png\x00image/jpeg\x00application/json\x00text/plain\x00"),
		},
		{
			"metadata",
			[]byte("Title: Wikipedia EN\nCreator: mwoffliner 1.11.0\nPublisher: openZIM\nDate: 2024-01-01\nLanguage: eng\n"),
		},
	}

	for _, tt := range patterns {
		t.Run(tt.name, func(t *testing.T) {
			for _, level := range []zstd.EncoderLevel{
				zstd.SpeedFastest,
				zstd.SpeedDefault,
				zstd.SpeedBetterCompression,
			} {
				var buf bytes.Buffer
				encoder, _ := zstd.NewWriter(&buf, zstd.WithEncoderLevel(level))
				encoder.Write(tt.data)
				encoder.Close()

				decoder, _ := zstd.NewReader(&buf)
				decompressed, _ := io.ReadAll(decoder)
				decoder.Close()

				if !bytes.Equal(decompressed, tt.data) {
					t.Errorf("%s level=%s round-trip failed", tt.name, level)
				}
			}
		})
	}
}

func TestMultiBlockBoundary(t *testing.T) {
	// Test data sizes around typical block boundaries (128KB zstd default)
	sizes := []int{
		1 << 16,        // 64KB
		1 << 17,        // 128KB
		1<<17 - 1,      // 128KB - 1
		1<<17 + 1,      // 128KB + 1
		1 << 18,        // 256KB
		3 * (1 << 16),  // 192KB
	}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("size=%d", size), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)

			var buf bytes.Buffer
			encoder, _ := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
			encoder.Write(data)
			encoder.Close()

			decoder, _ := zstd.NewReader(&buf)
			decompressed, _ := io.ReadAll(decoder)
			decoder.Close()

			if !bytes.Equal(decompressed, data) {
				t.Errorf("size %d round-trip failed", size)
			}
		})
	}
}

func TestCompressedSize(t *testing.T) {
	data := bytes.Repeat([]byte("AAAAAAAAAAAAAAAAAAAA"), 1000)

	var buf bytes.Buffer
	encoder, _ := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	encoder.Write(data)
	encoder.Close()

	ratio := float64(buf.Len()) / float64(len(data))
	t.Logf("repeated compression ratio: %.1f%% (%d -> %d)", 100*ratio, len(data), buf.Len())

	if ratio > 0.1 {
		t.Logf("compression ratio for repeated data is higher than expected")
	}

	// Verify round trip
	decoder, _ := zstd.NewReader(&buf)
	decompressed, _ := io.ReadAll(decoder)
	decoder.Close()
	if !bytes.Equal(decompressed, data) {
		t.Error("round-trip failed")
	}
}

func TestReaderClose(t *testing.T) {
	var buf bytes.Buffer
	encoder, _ := zstd.NewWriter(&buf)
	encoder.Write([]byte("test data for close"))
	encoder.Close()

	decoder, _ := zstd.NewReader(&buf)
	decompressed1, _ := io.ReadAll(decoder)
	decoder.Close()

	if !bytes.Equal(decompressed1, []byte("test data for close")) {
		t.Error("data mismatch")
	}
}

func TestWriterClose_NoData(t *testing.T) {
	var buf bytes.Buffer
	encoder, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter error: %v", err)
	}
	err = encoder.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}

	decoder, _ := zstd.NewReader(&buf)
	defer decoder.Close()
	decompressed, _ := io.ReadAll(decoder)
	if len(decompressed) != 0 {
		t.Errorf("expected 0 bytes, got %d", len(decompressed))
	}
}

func TestLargeMultiBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large data test in short mode")
	}

	data := make([]byte, 1<<20) // 1 MB
	rand.Read(data)

	var buf bytes.Buffer
	encoder, _ := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
	encoder.Write(data)
	encoder.Close()

	t.Logf("1MB: compressed to %d bytes (%.1f%%)", buf.Len(), 100*float64(buf.Len())/float64(len(data)))

	decoder, _ := zstd.NewReader(&buf)
	decompressed, _ := io.ReadAll(decoder)
	decoder.Close()

	if !bytes.Equal(decompressed, data) {
		t.Error("large multi-block round-trip failed")
	}
}

func TestEncoderDecoder_Concurrent(t *testing.T) {
	data := bytes.Repeat([]byte("concurrent test data for zstd "), 100)
	const numGoroutines = 10

	errCh := make(chan error, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			var buf bytes.Buffer
			encoder, _ := zstd.NewWriter(&buf)
			encoder.Write(data)
			encoder.Close()

			decoder, _ := zstd.NewReader(&buf)
			decompressed, _ := io.ReadAll(decoder)
			decoder.Close()

			if !bytes.Equal(decompressed, data) {
				errCh <- fmt.Errorf("goroutine %d: round-trip failed", id)
			} else {
				errCh <- nil
			}
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		if err := <-errCh; err != nil {
			t.Error(err)
		}
	}
}

func TestPipePattern(t *testing.T) {
	data := bytes.Repeat([]byte("pipe pattern test data for zstd compression "), 50)

	pr, pw := io.Pipe()

	go func() {
		encoder, _ := zstd.NewWriter(pw)
		encoder.Write(data)
		encoder.Close()
		pw.Close()
	}()

	decoder, _ := zstd.NewReader(pr)
	decompressed, _ := io.ReadAll(decoder)
	decoder.Close()

	if !bytes.Equal(decompressed, data) {
		t.Error("pipe pattern round-trip failed")
	}
}

func TestRepeatability(t *testing.T) {
	data := []byte("repeatability is important for ZIM files because we use content hashing for deduplication")

	var firstCompressed []byte
	for i := 0; i < 10; i++ {
		var buf bytes.Buffer
		encoder, _ := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
		encoder.Write(data)
		encoder.Close()

		if i == 0 {
			firstCompressed = append([]byte{}, buf.Bytes()...)
		} else {
			if !bytes.Equal(buf.Bytes(), firstCompressed) {
				t.Fatalf("compression not repeatable at iteration %d", i)
			}
		}

		decoder, _ := zstd.NewReader(&buf)
		decompressed, _ := io.ReadAll(decoder)
		decoder.Close()

		if !bytes.Equal(decompressed, data) {
			t.Fatalf("round-trip failed at iteration %d", i)
		}
	}
}

func BenchmarkRoundTrip_1K(b *testing.B) {
	data := make([]byte, 1024)
	rand.Read(data)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cbuf bytes.Buffer
		w, _ := zstd.NewWriter(&cbuf)
		w.Write(data)
		w.Close()
		r, _ := zstd.NewReader(&cbuf)
		io.ReadAll(r)
		r.Close()
	}
}

func BenchmarkRoundTrip_64K(b *testing.B) {
	data := make([]byte, 65536)
	rand.Read(data)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cbuf bytes.Buffer
		w, _ := zstd.NewWriter(&cbuf)
		w.Write(data)
		w.Close()
		r, _ := zstd.NewReader(&cbuf)
		io.ReadAll(r)
		r.Close()
	}
}

func FuzzRoundTrip(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("hello"))
	f.Add(bytes.Repeat([]byte("abc"), 100))
	f.Fuzz(func(t *testing.T, data []byte) {
		var buf bytes.Buffer
		encoder, err := zstd.NewWriter(&buf)
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

		decoder, err := zstd.NewReader(&buf)
		if err != nil {
			t.Fatalf("NewReader error: %v", err)
		}
		decompressed, err := io.ReadAll(decoder)
		decoder.Close()
		if err != nil {
			t.Fatalf("ReadAll error: %v", err)
		}

		if !bytes.Equal(decompressed, data) {
			t.Fatalf("round-trip failed")
		}
	})
}

func FuzzEncodeAllDecodeAll(f *testing.F) {
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, data []byte) {
		encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
		if err != nil {
			t.Fatalf("NewWriter error: %v", err)
		}
		compressed := encoder.EncodeAll(data, nil)
		encoder.Close()

		decoder, err := zstd.NewReader(nil)
		if err != nil {
			t.Fatalf("NewReader error: %v", err)
		}
		decompressed, err := decoder.DecodeAll(compressed, nil)
		decoder.Close()
		if err != nil {
			t.Fatalf("DecodeAll error: %v", err)
		}

		if !bytes.Equal(decompressed, data) {
			t.Fatal("round-trip failed")
		}
	})
}
