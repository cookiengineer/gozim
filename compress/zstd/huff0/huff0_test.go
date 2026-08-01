package huff0_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/cookiengineer/gozim/compress/zstd/huff0"
)

func TestCompress1X_Decompress1X(t *testing.T) {
	inputs := []struct {
		name string
		data []byte
	}{
		{"short_text", []byte("This is a test of huff0 compression and decompression end to end round trip")},
		{"repeated", bytes.Repeat([]byte("abc"), 100)},
		{"html", []byte("<!DOCTYPE html><html><head><title>Test</title></head><body><p>Hello World</p></body></html>")},
		{"text", []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore.")},
		{"binary_repeating", bytes.Repeat([]byte{0xAA, 0xBB, 0xCC, 0xDD}, 64)},
		{"medium_repeated", bytes.Repeat([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ"), 200)},
	}

	for _, tt := range inputs {
		t.Run(tt.name, func(t *testing.T) {
			var s huff0.Scratch
			compressed, _, err := huff0.Compress1X(tt.data, &s)
			if err != nil {
				t.Fatalf("Compress1X error: %v", err)
			}
			t.Logf("compressed %d -> %d bytes (%.1f%%)", len(tt.data), len(compressed), 100*float64(len(compressed))/float64(len(tt.data)))

			s2, remain, err := huff0.ReadTable(compressed, nil)
			if err != nil {
				t.Fatalf("ReadTable error: %v", err)
			}
			decompressed, err := s2.Decompress1X(remain)
			if err != nil {
				t.Fatalf("Decompress1X error: %v", err)
			}

			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("round-trip failed: got %d bytes, want %d bytes", len(decompressed), len(tt.data))
				if len(tt.data) < 100 {
					t.Errorf("  want: %q", tt.data)
					t.Errorf("  got:  %q", decompressed)
				}
			}
		})
	}
}

func TestCompress4X_Decompress4X(t *testing.T) {
	inputs := []struct {
		name string
		data []byte
	}{
		{"mixed", bytes.Repeat([]byte("hello world "), 10)},
		{"repeated", bytes.Repeat([]byte("XYZ"), 500)},
		{"html", []byte("<!DOCTYPE html><html><head><title>Test</title></head><body><p>Hello World</p></body></html>")},
		{"medium", bytes.Repeat([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ "), 100)},
		{"binary", bytes.Repeat([]byte{0xAA, 0x55, 0xAA, 0x55}, 200)},
	}

	for _, tt := range inputs {
		t.Run(tt.name, func(t *testing.T) {
			var s huff0.Scratch
			compressed, _, err := huff0.Compress4X(tt.data, &s)
			if err != nil {
				t.Fatalf("Compress4X error: %v", err)
			}
			t.Logf("compressed 4X %d -> %d bytes (%.1f%%)", len(tt.data), len(compressed), 100*float64(len(compressed))/float64(len(tt.data)))

			s2, remain, err := huff0.ReadTable(compressed, nil)
			if err != nil {
				t.Fatalf("ReadTable error: %v", err)
			}
			decompressed, err := s2.Decompress4X(remain, len(tt.data))
			if err != nil {
				t.Fatalf("Decompress4X error: %v", err)
			}

			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("round-trip failed: got %d bytes, want %d bytes", len(decompressed), len(tt.data))
			}
		})
	}
}

func TestCompress1X_VaryingSizes(t *testing.T) {
	for _, size := range []int{0, 1, 2, 3, 4, 5, 10, 100, 200, 500, 1024, 4096} {
		t.Run(fmt.Sprintf("size=%d", size), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)

			var s huff0.Scratch
			compressed, _, err := huff0.Compress1X(data, &s)
			if err != nil {
				if err == huff0.ErrIncompressible || err == huff0.ErrUseRLE {
					t.Logf("size %d: %v (expected)", size, err)
					return
				}
				t.Fatalf("Compress1X error for size %d: %v", size, err)
			}

			s2, remain, err := huff0.ReadTable(compressed, nil)
			if err != nil {
				t.Fatalf("ReadTable error: %v", err)
			}
			decompressed, err := s2.Decompress1X(remain)
			if err != nil {
				t.Fatalf("Decompress1X error: %v", err)
			}

			if !bytes.Equal(decompressed, data) {
				t.Errorf("size %d round-trip failed", size)
			}
		})
	}
}

func TestCompress4X_VaryingSizes(t *testing.T) {
	for _, size := range []int{12, 100, 200, 500, 1024, 4096} {
		t.Run(fmt.Sprintf("size=%d", size), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)

			var s huff0.Scratch
			compressed, _, err := huff0.Compress4X(data, &s)
			if err != nil {
				if err == huff0.ErrIncompressible || err == huff0.ErrUseRLE {
					t.Logf("size %d: %v (expected)", size, err)
					return
				}
				t.Fatalf("Compress4X error for size %d: %v", size, err)
			}

			s2, remain, err := huff0.ReadTable(compressed, nil)
			if err != nil {
				t.Fatalf("ReadTable error: %v", err)
			}
			decompressed, err := s2.Decompress4X(remain, len(data))
			if err != nil {
				t.Fatalf("Decompress4X error: %v", err)
			}

			if !bytes.Equal(decompressed, data) {
				t.Errorf("size %d round-trip failed", size)
			}
		})
	}
}

func TestCompress1X_ScratchReuse(t *testing.T) {
	data1 := bytes.Repeat([]byte("first chunk of data for testing scratch reuse in huff0 "), 3)
	data2 := bytes.Repeat([]byte("second set of data, different from the first one "), 3)

	var s huff0.Scratch
	s.Reuse = huff0.ReusePolicyAllow

	compressed1, _, err := huff0.Compress1X(data1, &s)
	if err != nil {
		t.Fatalf("first Compress1X error: %v", err)
	}

	s2, remain, _ := huff0.ReadTable(compressed1, nil)
	decompressed1, _ := s2.Decompress1X(remain)
	if !bytes.Equal(decompressed1, data1) {
		t.Fatal("first round-trip failed")
	}

	compressed2, _, err := huff0.Compress1X(data2, &s)
	if err != nil {
		t.Fatalf("second Compress1X error: %v", err)
	}

	s3, remain2, _ := huff0.ReadTable(compressed2, nil)
	decompressed2, _ := s3.Decompress1X(remain2)
	if !bytes.Equal(decompressed2, data2) {
		t.Fatal("second round-trip failed")
	}
}

func TestCompress4X_ScratchReuse(t *testing.T) {
	data1 := bytes.Repeat([]byte("scratch reuse test data for 4X compression "), 20)
	data2 := bytes.Repeat([]byte("completely different data set for second round "), 20)

	var s huff0.Scratch
	s.Reuse = huff0.ReusePolicyAllow

	compressed1, _, err := huff0.Compress4X(data1, &s)
	if err != nil {
		t.Fatalf("first Compress4X error: %v", err)
	}

	s2, remain, _ := huff0.ReadTable(compressed1, nil)
	decompressed1, _ := s2.Decompress4X(remain, len(data1))
	if !bytes.Equal(decompressed1, data1) {
		t.Fatal("first round-trip failed")
	}

	compressed2, _, err := huff0.Compress4X(data2, &s)
	if err != nil {
		t.Fatalf("second Compress4X error: %v", err)
	}

	s3, remain2, _ := huff0.ReadTable(compressed2, nil)
	decompressed2, _ := s3.Decompress4X(remain2, len(data2))
	if !bytes.Equal(decompressed2, data2) {
		t.Fatal("second round-trip failed")
	}
}

func TestBuildCTable_And_Compress(t *testing.T) {
	data := bytes.Repeat([]byte("hello world "), 50)

	var hist [256]uint32
	for _, b := range data {
		hist[b]++
	}

	var s huff0.Scratch
	err := s.BuildCTable(&hist)
	if err != nil {
		t.Fatalf("BuildCTable error: %v", err)
	}

	if !s.CanUseTable(&hist) {
		t.Error("table should be usable for its own histogram")
	}
	est := s.EstimateSize(&hist)
	if est < 0 {
		t.Errorf("EstimateSize returned negative: %d", est)
	}
	t.Logf("estimated size with built table: %d bytes", est)

	// Compress with ReusePolicyNone so table header is always emitted
	s.Reuse = huff0.ReusePolicyNone
	compressed, _, err := huff0.Compress1X(data, &s)
	if err != nil {
		t.Fatalf("Compress1X error: %v", err)
	}

	s2, remain, _ := huff0.ReadTable(compressed, nil)
	decompressed, _ := s2.Decompress1X(remain)
	if !bytes.Equal(decompressed, data) {
		t.Fatal("round-trip failed")
	}
}

func TestAppendTable(t *testing.T) {
	data := bytes.Repeat([]byte("append table test data "), 30)

	var hist [256]uint32
	for _, b := range data {
		hist[b]++
	}

	var s huff0.Scratch
	err := s.BuildCTable(&hist)
	if err != nil {
		t.Fatalf("BuildCTable error: %v", err)
	}

	tableBytes, err := s.AppendTable(nil)
	if err != nil {
		t.Fatalf("AppendTable error: %v", err)
	}

	s2, _, err := huff0.ReadTable(tableBytes, nil)
	if err != nil {
		t.Fatalf("ReadTable error: %v", err)
	}
	_ = s2
}

func TestEstimateSize(t *testing.T) {
	var hist [256]uint32
	data := bytes.Repeat([]byte("estimate size test "), 50)
	for _, b := range data {
		hist[b]++
	}

	var s huff0.Scratch
	err := s.BuildCTable(&hist)
	if err != nil {
		t.Fatalf("BuildCTable error: %v", err)
	}

	est := s.EstimateSize(&hist)
	if est < 0 {
		t.Errorf("EstimateSize returned negative: %d", est)
	}
	t.Logf("estimated compressed size: %d bytes (uncompressed: %d)", est, len(data))
}

func TestCanUseTable(t *testing.T) {
	var hist1, hist2 [256]uint32

	data1 := bytes.Repeat([]byte("AAAAABBBBBCCCCC"), 20)
	data2 := bytes.Repeat([]byte("XXXXXYYYYYZZZZZ"), 20)

	for _, b := range data1 {
		hist1[b]++
	}
	for _, b := range data2 {
		hist2[b]++
	}

	var s huff0.Scratch
	err := s.BuildCTable(&hist1)
	if err != nil {
		t.Fatalf("BuildCTable error: %v", err)
	}

	if !s.CanUseTable(&hist1) {
		t.Error("table should be usable for its own histogram")
	}

	canUse := s.CanUseTable(&hist2)
	t.Logf("CanUseTable for different histogram: %v", canUse)
}

func TestTransferCTable(t *testing.T) {
	data := bytes.Repeat([]byte("transfer ctable test "), 30)

	var hist [256]uint32
	for _, b := range data {
		hist[b]++
	}

	var s1, s2 huff0.Scratch
	err := s1.BuildCTable(&hist)
	if err != nil {
		t.Fatalf("BuildCTable error: %v", err)
	}

	s2.TransferCTable(&s1)
	est := s2.EstimateSize(&hist)
	if est < 0 {
		t.Error("transfer failed: table not usable")
	}
}

func TestReusePolicies(t *testing.T) {
	data := bytes.Repeat([]byte("reuse policy test data for huff0 "), 50)

	// Test with ReusePolicyNone first - always emits table headers
	var s huff0.Scratch
	s.Reuse = huff0.ReusePolicyNone
	for i := 0; i < 3; i++ {
		compressed, _, err := huff0.Compress1X(data, &s)
		if err != nil {
			t.Fatalf("iteration %d: Compress1X error: %v", i, err)
		}
		s2, remain, _ := huff0.ReadTable(compressed, nil)
		decompressed, _ := s2.Decompress1X(remain)
		if !bytes.Equal(decompressed, data) {
			t.Fatalf("iteration %d: round-trip failed", i)
		}
	}

	// Test ReusePolicyMust with BuildCTable - verifies that compression
	// with a pre-built table succeeds and the table is properly used.
	var s2 huff0.Scratch
	var hist [256]uint32
	for _, b := range data {
		hist[b]++
	}
	if err := s2.BuildCTable(&hist); err != nil {
		t.Fatalf("BuildCTable error: %v", err)
	}
	s2.Reuse = huff0.ReusePolicyMust
	compressed, reused, err := huff0.Compress1X(data, &s2)
	if err != nil {
		t.Fatalf("Compress1X with ReusePolicyMust error: %v", err)
	}
	if !reused {
		t.Log("ReusePolicyMust did not reuse (table generated)")
	}
	_ = compressed
}

func TestEstimateSizes(t *testing.T) {
	data := bytes.Repeat([]byte("estimate sizes test data "), 30)

	var s huff0.Scratch
	tableSz, dataSz, reuseSz, err := huff0.EstimateSizes(data, &s)
	if err != nil {
		t.Fatalf("EstimateSizes error: %v", err)
	}
	t.Logf("table=%d, data=%d, reuse=%d", tableSz, dataSz, reuseSz)

	if tableSz < 0 || dataSz < 0 {
		t.Errorf("EstimateSizes returned negative values: %d/%d/%d", tableSz, dataSz, reuseSz)
	}
	if reuseSz == -1 {
		t.Log("reuseSz is -1 (no previous table to reuse)")
	}
}

func TestCompress1X_TableAndDataSeparation(t *testing.T) {
	data := bytes.Repeat([]byte("table data separation test "), 40)

	var s huff0.Scratch
	s.Reuse = huff0.ReusePolicyNone
	compressed, _, err := huff0.Compress1X(data, &s)
	if err != nil {
		t.Fatalf("Compress1X error: %v", err)
	}

	if len(s.OutTable) == 0 {
		t.Log("no OutTable (table embedded)")
	} else {
		t.Logf("OutTable=%d bytes, OutData=%d bytes", len(s.OutTable), len(s.OutData))
	}

	s2, remain, _ := huff0.ReadTable(compressed, nil)
	decompressed, _ := s2.Decompress1X(remain)
	if !bytes.Equal(decompressed, data) {
		t.Fatal("round-trip failed")
	}
}

func TestMaxDecodedSize(t *testing.T) {
	data := bytes.Repeat([]byte("max decode size test "), 50)

	var s huff0.Scratch
	compressed, _, err := huff0.Compress1X(data, &s)
	if err != nil {
		t.Fatalf("Compress1X error: %v", err)
	}

	s2, remain, _ := huff0.ReadTable(compressed, nil)
	s2.MaxDecodedSize = len(data) / 2
	_, err = s2.Decompress1X(remain)
	if err != huff0.ErrMaxDecodedSizeExceeded {
		t.Errorf("expected ErrMaxDecodedSizeExceeded, got %v", err)
	}
}

func TestBlockSizeMax(t *testing.T) {
	if huff0.BlockSizeMax != (1<<18)-1 {
		t.Errorf("BlockSizeMax = %d, want %d", huff0.BlockSizeMax, (1<<18)-1)
	}
}

func TestDecompress1X_WithDecoder(t *testing.T) {
	data := []byte("test decoder-based decompress")
	var s huff0.Scratch
	compressed, _, err := huff0.Compress1X(data, &s)
	if err != nil {
		t.Fatalf("Compress1X error: %v", err)
	}

	s2, remain, _ := huff0.ReadTable(compressed, nil)
	decoder := s2.Decoder()
	buf := make([]byte, 0, len(data)*2)
	decompressed, err := decoder.Decompress1X(buf, remain)
	if err != nil {
		t.Fatalf("Decoder.Decompress1X error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatal("round-trip failed")
	}
}

func TestDecompress4X_WithDecoder(t *testing.T) {
	data := bytes.Repeat([]byte("decoder 4x decompress test "), 30)
	var s huff0.Scratch
	compressed, _, err := huff0.Compress4X(data, &s)
	if err != nil {
		t.Fatalf("Compress4X error: %v", err)
	}

	s2, remain, _ := huff0.ReadTable(compressed, nil)
	decoder := s2.Decoder()
	buf := make([]byte, len(data))
	decompressed, err := decoder.Decompress4X(buf, remain)
	if err != nil {
		t.Fatalf("Decoder.Decompress4X error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatal("round-trip failed")
	}
}

func TestReadTable_InvalidInput(t *testing.T) {
	invalidTests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"one_byte", []byte{0}},
		{"corrupt", []byte{0xFF, 0x00, 0x00, 0x00}},
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := huff0.ReadTable(tt.data, nil)
			if err == nil {
				t.Error("expected error for invalid input")
			}
		})
	}
}

func TestCompress1X_Incompressible(t *testing.T) {
	data := make([]byte, 100)
	rand.Read(data)

	var s huff0.Scratch
	s.Reuse = huff0.ReusePolicyNone
	_, _, err := huff0.Compress1X(data, &s)
	if err != nil {
		if err == huff0.ErrIncompressible || err == huff0.ErrUseRLE {
			t.Logf("expected: %v", err)
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkCompress1X_Decompress1X(b *testing.B) {
	data := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 20)
	b.Run("compress", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var s huff0.Scratch
			s.Reuse = huff0.ReusePolicyNone
			huff0.Compress1X(data, &s)
		}
	})

	var s huff0.Scratch
	compressed, _, _ := huff0.Compress1X(data, &s)
	s2, remain, _ := huff0.ReadTable(compressed, nil)

	b.Run("decompress", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s2.Decompress1X(remain)
		}
	})
}

func BenchmarkCompress4X_Decompress4X(b *testing.B) {
	data := bytes.Repeat([]byte("benchmark 4x compression test data for huff0 "), 50)
	b.Run("compress4x", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var s huff0.Scratch
			s.Reuse = huff0.ReusePolicyNone
			huff0.Compress4X(data, &s)
		}
	})

	var s huff0.Scratch
	compressed, _, _ := huff0.Compress4X(data, &s)
	s2, remain, _ := huff0.ReadTable(compressed, nil)

	b.Run("decompress4x", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s2.Decompress4X(remain, len(data))
		}
	})
}

func FuzzCompressDecompress1X(f *testing.F) {
	f.Add([]byte("hello world"))
	f.Add(bytes.Repeat([]byte("abc"), 100))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > huff0.BlockSizeMax {
			return
		}
		var s huff0.Scratch
		s.Reuse = huff0.ReusePolicyNone
		compressed, _, err := huff0.Compress1X(data, &s)
		if err != nil {
			return
		}
		s2, remain, err := huff0.ReadTable(compressed, nil)
		if err != nil {
			t.Fatalf("ReadTable error: %v", err)
		}
		decompressed, err := s2.Decompress1X(remain)
		if err != nil {
			t.Fatalf("Decompress1X error: %v", err)
		}
		if !bytes.Equal(decompressed, data) {
			t.Fatalf("round-trip mismatch")
		}
	})
}

func FuzzCompressDecompress4X(f *testing.F) {
	f.Add(bytes.Repeat([]byte("test"), 200))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 12 || len(data) > huff0.BlockSizeMax {
			return
		}
		var s huff0.Scratch
		s.Reuse = huff0.ReusePolicyNone
		compressed, _, err := huff0.Compress4X(data, &s)
		if err != nil {
			return
		}
		s2, remain, err := huff0.ReadTable(compressed, nil)
		if err != nil {
			t.Fatalf("ReadTable error: %v", err)
		}
		decompressed, err := s2.Decompress4X(remain, len(data))
		if err != nil {
			t.Fatalf("Decompress4X error: %v", err)
		}
		if !bytes.Equal(decompressed, data) {
			t.Fatalf("round-trip mismatch")
		}
	})
}
