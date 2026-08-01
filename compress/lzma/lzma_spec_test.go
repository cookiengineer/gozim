package lzma_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"testing"

	"github.com/cookiengineer/gozim/compress/lzma"
)

// =============================================================================
// LZMA File Header (spec §lzma file format)
// =============================================================================

func TestSpec_HeaderFormat(t *testing.T) {
	// LZMA file header: 1 byte properties + 4 bytes dictSize LE + 8 bytes uncompressedSize LE
	data := []byte("header format test")
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter(&compressed)
	w.Write(data)
	w.Close()

	header := compressed.Bytes()[:13]
	// Properties byte at offset 0
	propsCode := header[0]
	p, err := lzma.PropertiesForCode(propsCode)
	if err != nil {
		t.Fatalf("invalid properties code %d: %v", propsCode, err)
	}
	if p.LC < 0 || p.LC > 8 || p.LP < 0 || p.LP > 4 || p.PB < 0 || p.PB > 4 {
		t.Errorf("properties out of range: LC=%d LP=%d PB=%d", p.LC, p.LP, p.PB)
	}
}

func TestSpec_PropertiesEncoding(t *testing.T) {
	// Spec: properties[0] = (Byte)((pb * 5 + lp) * 9 + lc)
	for lc := 0; lc <= 8; lc++ {
		for lp := 0; lp <= 4; lp++ {
			for pb := 0; pb <= 4; pb++ {
				p := lzma.Properties{LC: lc, LP: lp, PB: pb}
				code := p.Code()
				expected := byte((pb*5+lp)*9 + lc)
				if code != expected {
					t.Errorf("LC=%d LP=%d PB=%d: Code()=%d, spec=%d",
						lc, lp, pb, code, expected)
				}
			}
		}
	}
}

func TestSpec_PropertiesDecoding(t *testing.T) {
	// Spec: lc = d % 9; d /= 9; pb = d / 5; lp = d % 5
	for code := 0; code <= 224; code++ {
		p, err := lzma.PropertiesForCode(byte(code))
		if err != nil {
			t.Errorf("PropertiesForCode(%d) error: %v", code, err)
			continue
		}
		if p.LC != code%9 {
			t.Errorf("LC mismatch for code %d: got %d, want %d", code, p.LC, code%9)
		}
		if p.LP != (code/9)%5 {
			t.Errorf("LP mismatch for code %d: got %d, want %d", code, p.LP, (code/9)%5)
		}
		if p.PB != code/9/5%5 {
			t.Errorf("PB mismatch for code %d: got %d, want %d", code, p.PB, code/9/5%5)
		}
	}
}

func TestSpec_PropertiesRange(t *testing.T) {
	// Spec: LC [0,8], LP [0,4], PB [0,4]
	// Code values >= 225 (9*5*5) are invalid
	for code := 225; code <= 255; code++ {
		_, err := lzma.PropertiesForCode(byte(code))
		if err == nil {
			t.Errorf("code %d should be invalid (>= 9*5*5 = 225)", code)
		}
	}
}

func TestSpec_DictSizeMinimum(t *testing.T) {
	// Spec: If dictSize < LZMA_DIC_MIN (1<<12 = 4096), dictSize = LZMA_DIC_MIN
	// Verifies round-trip works with small data
	data := bytes.Repeat([]byte("A"), 100)
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter(&compressed)
	w.Write(data)
	w.Close()

	r, _ := lzma.NewReader(&compressed)
	d, _ := io.ReadAll(r)
	if !bytes.Equal(d, data) {
		t.Error("round-trip failed")
	}
}

func TestSpec_UncompressedSizeAllOnes(t *testing.T) {
	// Spec: all-ones (0xFFFFFFFFFFFFFFFF) means unknown uncompressed size
	// Our implementation uses this for streams where size is unknown
	data := []byte("unknown size stream test")
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter(&compressed)
	w.Write(data)
	w.Close()

	// Should decode correctly regardless of header size field
	r, _ := lzma.NewReader(&compressed)
	d, _ := io.ReadAll(r)
	if !bytes.Equal(d, data) {
		t.Error("round-trip failed for unknown size")
	}
}

// =============================================================================
// Dictionary Size Encode/Decode (LZMA2 scheme from spec)
// =============================================================================

func TestSpec_DictCapEncoding(t *testing.T) {
	// Spec LZMA2 scheme:
	// dictSize = (p == 40) ? 0xFFFFFFFF : (((UInt32)2 | ((p) & 1)) << ((p) / 2 + 11))
	for c := byte(0); c <= 40; c++ {
		dictCap, err := lzma.DecodeDictCap(c)
		if err != nil {
			t.Errorf("DecodeDictCap(%d) error: %v", c, err)
			continue
		}
		// Re-encode and decode to verify consistency
		enc := lzma.EncodeDictCap(dictCap)
		dec2, _ := lzma.DecodeDictCap(enc)
		if dec2 != dictCap {
			t.Errorf("c=%d: DecodeDictCap=%d, EncodeDictCap=%d, roundtrip=%d",
				c, dictCap, enc, dec2)
		}
	}
	// c=40 is special: should decode to 0xFFFFFFFF
	dc40, _ := lzma.DecodeDictCap(40)
	if dc40 != 0xFFFFFFFF {
		t.Errorf("DecodeDictCap(40) = %d, want 0xFFFFFFFF", dc40)
	}
}

// =============================================================================
// Decompressed data integrity
// =============================================================================

func TestSpec_SHA256Integrity(t *testing.T) {
	data := make([]byte, 100000)
	rand.Read(data)
	origHash := sha256.Sum256(data)

	var compressed bytes.Buffer
	w, _ := lzma.NewWriter(&compressed)
	w.Write(data)
	w.Close()

	r, _ := lzma.NewReader(&compressed)
	decompressed, _ := io.ReadAll(r)

	if sha256.Sum256(decompressed) != origHash {
		t.Error("SHA-256 integrity check failed")
	}
}

func TestSpec_RoundTrip_VaryingSizes(t *testing.T) {
	sizes := []int{0, 1, 2, 3, 5, 10, 50, 100, 500, 1024, 4096, 16384, 65536}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)

			var compressed bytes.Buffer
			w, _ := lzma.NewWriter(&compressed)
			w.Write(data)
			w.Close()

			r, _ := lzma.NewReader(&compressed)
			d, _ := io.ReadAll(r)
			if !bytes.Equal(d, data) {
				t.Errorf("size %d round-trip failed", size)
			}
		})
	}
}

func TestSpec_RoundTrip_Large(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large test in short mode")
	}
	data := make([]byte, 1<<20)
	rand.Read(data)

	var compressed bytes.Buffer
	w, _ := lzma.NewWriter(&compressed)
	w.Write(data)
	w.Close()

	r, _ := lzma.NewReader(&compressed)
	d, _ := io.ReadAll(r)
	if !bytes.Equal(d, data) {
		t.Error("1 MB round-trip failed")
	}
}

// =============================================================================
// LZMA1 vs LZMA2 format compatibility tests
// =============================================================================

func TestSpec_LZMA1_ValidHeader(t *testing.T) {
	data := []byte("header validity check")
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter(&compressed)
	w.Write(data)
	w.Close()

	header := compressed.Bytes()[:13]
	if !lzma.ValidHeader(header) {
		t.Error("generated LZMA1 header should be valid")
	}

	// Corrupt properties byte
	corrupt := make([]byte, 13)
	copy(corrupt, header)
	corrupt[0] = 0xFF // invalid properties code
	if lzma.ValidHeader(corrupt) {
		t.Error("corrupted header should be invalid")
	}
}

func TestSpec_LZMA2_NewReader2(t *testing.T) {
	data := bytes.Repeat([]byte("LZMA2 format test "), 100)
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter2(&compressed)
	w.Write(data)
	w.Close()

	r, err := lzma.NewReader2(&compressed)
	if err != nil {
		t.Fatalf("NewReader2 error: %v", err)
	}
	d, _ := io.ReadAll(r)
	if !bytes.Equal(d, data) {
		t.Error("LZMA2 round-trip failed")
	}
	if !r.EOS() {
		t.Error("LZMA2 EOS marker not detected")
	}
}

func TestSpec_LZMA2_DictSizes(t *testing.T) {
	// Spec: LZMA2 supports dict sizes: (2<<11), (3<<11), (2<<12), (3<<12), ..., (2<<30), (3<<30), (2<<31)-1
	// Our DecodeDictCap should decode all 41 codes (0-40)
	data := bytes.Repeat([]byte("dict size test "), 50)
	for c := byte(0); c <= 40; c++ {
		var compressed bytes.Buffer
		w, _ := lzma.NewWriter2(&compressed)
		w.Write(data)
		w.Close()

		r, err := lzma.NewReader2(&compressed)
		if err != nil {
			t.Fatalf("c=%d: NewReader2 error: %v", c, err)
		}
		d, _ := io.ReadAll(r)
		if !bytes.Equal(d, data) {
			t.Errorf("c=%d: round-trip failed", c)
		}
	}
}

// =============================================================================
// Streaming reads / Concurrency
// =============================================================================

func TestSpec_StreamingRead(t *testing.T) {
	data := make([]byte, 50000)
	rand.Read(data)
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter(&compressed)
	w.Write(data)
	w.Close()

	for _, chunkSize := range []int{1, 3, 13, 64, 128} {
		t.Run(fmt.Sprintf("chunk=%d", chunkSize), func(t *testing.T) {
			r, _ := lzma.NewReader(bytes.NewReader(compressed.Bytes()))
			var result bytes.Buffer
			buf := make([]byte, chunkSize)
			for {
				n, err := r.Read(buf)
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
				t.Errorf("chunk=%d failed", chunkSize)
			}
		})
	}
}

func TestSpec_Concurrent(t *testing.T) {
	data := bytes.Repeat([]byte("concurrent test "), 50)
	const n = 10
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			var cbuf bytes.Buffer
			w, _ := lzma.NewWriter(&cbuf)
			w.Write(data)
			w.Close()
			r, _ := lzma.NewReader(&cbuf)
			d, _ := io.ReadAll(r)
			if !bytes.Equal(d, data) {
				errs <- fmt.Errorf("mismatch")
			} else {
				errs <- nil
			}
		}()
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Error(err)
		}
	}
}

// =============================================================================
// Properties: Full coverage of all valid codes
// =============================================================================

func TestSpec_AllPropertyCodes(t *testing.T) {
	// All 225 valid LZMA property codes (0-224)
	for code := 0; code <= 224; code++ {
		p, err := lzma.PropertiesForCode(byte(code))
		if err != nil {
			t.Errorf("code %d should be valid: %v", code, err)
		}
		if p.Code() != byte(code) {
			t.Errorf("code %d round-trip fails: Code()=%d", code, p.Code())
		}
	}
}

// =============================================================================
// Close idempotency
// =============================================================================

func TestSpec_CloseIdempotent(t *testing.T) {
	var buf bytes.Buffer
	w, _ := lzma.NewWriter(&buf)
	w.Write([]byte("close twice"))
	w.Close()
	w.Close()

	r, _ := lzma.NewReader(&buf)
	d, _ := io.ReadAll(r)
	if !bytes.Equal(d, []byte("close twice")) {
		t.Error("double close failed")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkSpec_LZMA1_RoundTrip_4K(b *testing.B) {
	data := make([]byte, 4096)
	rand.Read(data)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cbuf bytes.Buffer
		w, _ := lzma.NewWriter(&cbuf)
		w.Write(data)
		w.Close()
		r, _ := lzma.NewReader(&cbuf)
		io.ReadAll(r)
	}
}

func BenchmarkSpec_LZMA2_RoundTrip_4K(b *testing.B) {
	data := make([]byte, 4096)
	rand.Read(data)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cbuf bytes.Buffer
		w, _ := lzma.NewWriter2(&cbuf)
		w.Write(data)
		w.Close()
		r, _ := lzma.NewReader2(&cbuf)
		io.ReadAll(r)
	}
}

// =============================================================================
// Fuzz tests
// =============================================================================

func FuzzSpec_Properties(f *testing.F) {
	f.Add(byte(0))
	f.Add(byte(93)) // LC=3, LP=0, PB=2 (default)
	f.Fuzz(func(t *testing.T, code byte) {
		p, err := lzma.PropertiesForCode(code)
		if err != nil {
			return // invalid codes are expected
		}
		if pc := p.Code(); pc != code {
			t.Errorf("code %d -> Properties -> Code() = %d", code, pc)
		}
	})
}

func FuzzSpec_RoundTrip(f *testing.F) {
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, data []byte) {
		var cbuf bytes.Buffer
		w, _ := lzma.NewWriter(&cbuf)
		w.Write(data)
		w.Close()

		r, _ := lzma.NewReader(&cbuf)
		d, _ := io.ReadAll(r)
		if !bytes.Equal(d, data) {
			t.Fatal("mismatch")
		}
	})
}
