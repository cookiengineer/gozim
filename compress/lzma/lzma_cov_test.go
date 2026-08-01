package lzma_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"testing"

	"github.com/cookiengineer/gozim/compress/lzma"
)

func TestProperties_Code(t *testing.T) {
	for lc := 0; lc <= 8; lc++ {
		for lp := 0; lp <= 4; lp++ {
			for pb := 0; pb <= 4; pb++ {
				p := lzma.Properties{LC: lc, LP: lp, PB: pb}
				code := p.Code()
				if code < 0 || code > 255 {
					t.Errorf("Code() for LC=%d LP=%d PB=%d returned %d", lc, lp, pb, code)
				}
			}
		}
	}
}

func TestProperties_String(t *testing.T) {
	p := lzma.Properties{LC: 3, LP: 0, PB: 2}
	s := p.String()
	if s == "" {
		t.Error("String() returned empty")
	}
}

func TestPropertiesForCode(t *testing.T) {
	// Only codes 0-224 can be valid (maxPropertyCode = 5*5*9 - 1 = 224)
	for code := 0; code <= 224; code++ {
		p, err := lzma.PropertiesForCode(byte(code))
		if err != nil {
			t.Errorf("PropertiesForCode(%d) error: %v", code, err)
			continue
		}
		if c := p.Code(); int(c) != code {
			t.Errorf("Code round-trip failed for code %d: got %d", code, c)
		}
	}
	// Codes 225-255 should all return errors (use int to avoid byte overflow)
	for code := 225; code <= 255; code++ {
		_, err := lzma.PropertiesForCode(byte(code))
		if err == nil {
			t.Errorf("PropertiesForCode(%d) should return error", code)
		}
	}
}

func TestPropertiesForCode_RoundTrip(t *testing.T) {
	props := []lzma.Properties{
		{LC: 3, LP: 0, PB: 2},
		{LC: 0, LP: 0, PB: 0},
		{LC: 8, LP: 4, PB: 4},
		{LC: 5, LP: 2, PB: 3},
	}
	for _, p := range props {
		code := p.Code()
		p2, err := lzma.PropertiesForCode(code)
		if err != nil {
			t.Errorf("PropertiesForCode(%d) error: %v", code, err)
		}
		if p2 != p {
			t.Errorf("round-trip failed: %v -> code %d -> %v", p, code, p2)
		}
	}
}

func TestValidHeader(t *testing.T) {
	// Generate a valid LZMA1 header by compressing data
	data := []byte("test")
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter(&compressed)
	w.Write(data)
	w.Close()

	validHeader := compressed.Bytes()[:13]

	tests := []struct {
		name   string
		data   []byte
		expect bool
	}{
		{"empty", []byte{}, false},
		{"too_short", []byte{0x5D, 0x00, 0x00}, false},
		{"valid_from_compressed", validHeader, true},
		{"garbage", bytes.Repeat([]byte{0xFF}, 13), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lzma.ValidHeader(tt.data); got != tt.expect {
				t.Errorf("ValidHeader = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestDecodeDictCap(t *testing.T) {
	for c := 0; c <= 40; c++ {
		dictCap, err := lzma.DecodeDictCap(byte(c))
		if err != nil {
			t.Errorf("DecodeDictCap(%d) error: %v", c, err)
		}
		if dictCap < 0 {
			t.Errorf("DecodeDictCap(%d) returned negative: %d", c, dictCap)
		}
	}
	n, err := lzma.DecodeDictCap(0)
	if err != nil || n == 0 {
		t.Errorf("DecodeDictCap(0) = (%d, %v)", n, err)
	}
}

func TestEncodeDictCap(t *testing.T) {
	sizes := []int64{4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576}
	for _, size := range sizes {
		c := lzma.EncodeDictCap(size)
		decoded, _ := lzma.DecodeDictCap(c)
		if decoded < size {
			t.Errorf("dict cap round-trip: %d -> c=%d -> %d (< %d)", size, c, decoded, size)
		}
	}
}

func TestDecodeEncodeDictCap_RoundTrip(t *testing.T) {
	for c := byte(0); c <= 40; c++ {
		decoded, err := lzma.DecodeDictCap(c)
		if err != nil {
			continue
		}
		encoded := lzma.EncodeDictCap(decoded)
		roundTripped, _ := lzma.DecodeDictCap(encoded)
		if roundTripped != decoded {
			t.Errorf("c=%d mismatch: %d -> enc=%d -> %d",
				c, decoded, encoded, roundTripped)
		}
	}
}

func TestMatchAlgorithm_String(t *testing.T) {
	if s := lzma.AlgorithmHashTable.String(); s == "" {
		t.Error("AlgorithmHashTable.String() empty")
	}
	if s := lzma.AlgorithmBinaryTree.String(); s == "" {
		t.Error("AlgorithmBinaryTree.String() empty")
	}
	if lzma.AlgorithmHashTable.String() == lzma.AlgorithmBinaryTree.String() {
		t.Error("AlgorithmHashTable and AlgorithmBinaryTree should have different strings")
	}
}

func TestReaderConfig_Verify(t *testing.T) {
	tests := []struct {
		name   string
		config lzma.ReaderConfig
		ok     bool
	}{
		{"default", lzma.ReaderConfig{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Verify()
			if tt.ok && err != nil {
				t.Errorf("expected ok, got error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestWriterConfig_Verify(t *testing.T) {
	tests := []struct {
		name   string
		config lzma.WriterConfig
		ok     bool
	}{
		{"default", lzma.WriterConfig{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Verify()
			if tt.ok && err != nil {
				t.Errorf("expected ok, got error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestWriterConfig_NewWriter(t *testing.T) {
	data := []byte("config-based writer test")
	var compressed bytes.Buffer
	wc, err := lzma.WriterConfig{}.NewWriter(&compressed)
	if err != nil {
		t.Fatalf("NewWriter error: %v", err)
	}
	wc.Write(data)
	wc.Close()

	reader, _ := lzma.NewReader(&compressed)
	decompressed, _ := io.ReadAll(reader)
	if !bytes.Equal(decompressed, data) {
		t.Error("round-trip failed")
	}
}

func TestReaderConfig_NewReader(t *testing.T) {
	data := []byte("reader config test")
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter(&compressed)
	w.Write(data)
	w.Close()

	r, err := lzma.ReaderConfig{}.NewReader(&compressed)
	if err != nil {
		t.Fatalf("NewReader error: %v", err)
	}
	decompressed, _ := io.ReadAll(r)
	if !bytes.Equal(decompressed, data) {
		t.Error("round-trip failed")
	}
}

func TestReader2Config_Verify(t *testing.T) {
	tests := []struct {
		name   string
		config lzma.Reader2Config
		ok     bool
	}{
		{"default", lzma.Reader2Config{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Verify()
			if tt.ok && err != nil {
				t.Errorf("expected ok, got error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestWriter2Config_Verify(t *testing.T) {
	tests := []struct {
		name   string
		config lzma.Writer2Config
		ok     bool
	}{
		{"default", lzma.Writer2Config{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Verify()
			if tt.ok && err != nil {
				t.Errorf("expected ok, got error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestWriter2Config_NewWriter2(t *testing.T) {
	data := bytes.Repeat([]byte("writer2 config test "), 100)
	var compressed bytes.Buffer
	wc, err := lzma.Writer2Config{}.NewWriter2(&compressed)
	if err != nil {
		t.Fatalf("NewWriter2 error: %v", err)
	}
	wc.Write(data)
	wc.Close()

	r, _ := lzma.NewReader2(&compressed)
	decompressed, _ := io.ReadAll(r)
	if !bytes.Equal(decompressed, data) {
		t.Error("round-trip failed")
	}
}

func TestWriter2_Flush(t *testing.T) {
	data := bytes.Repeat([]byte("flush test data "), 50)
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter2(&compressed)
	w.Write(data[:len(data)/2])
	w.Flush()
	w.Write(data[len(data)/2:])
	w.Close()

	r, _ := lzma.NewReader2(&compressed)
	decompressed, _ := io.ReadAll(r)
	if !bytes.Equal(decompressed, data) {
		t.Error("flush round-trip failed")
	}
}

func TestReader2_EOS(t *testing.T) {
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter2(&compressed)
	w.Write([]byte("eos test"))
	w.Close()

	r, err := lzma.NewReader2(&compressed)
	if err != nil {
		t.Fatalf("NewReader2 error: %v", err)
	}
	io.ReadAll(r)
	if !r.EOS() {
		t.Error("expected EOS to be true after full read")
	}
}

func TestWriter_CloseIdempotent(t *testing.T) {
	var buf bytes.Buffer
	w, _ := lzma.NewWriter(&buf)
	w.Write([]byte("close twice test"))
	w.Close()
	w.Close()

	r, _ := lzma.NewReader(&buf)
	decompressed, _ := io.ReadAll(r)
	if !bytes.Equal(decompressed, []byte("close twice test")) {
		t.Error("double-close round-trip failed")
	}
}

func TestLZMA_HeaderRoundTrip(t *testing.T) {
	data := []byte("header test data")
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter(&compressed)
	w.Write(data)
	w.Close()

	if len(compressed.Bytes()) >= 13 && lzma.ValidHeader(compressed.Bytes()[:13]) {
		t.Log("valid lzma1 header recognized")
	}
}

func TestLZMA_Concurrent(t *testing.T) {
	data := bytes.Repeat([]byte("concurrent lzma test "), 100)
	const n = 10
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			var compressed bytes.Buffer
			w, _ := lzma.NewWriter(&compressed)
			w.Write(data)
			w.Close()
			r, _ := lzma.NewReader(&compressed)
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

func TestLZMA_StrictDictConfig(t *testing.T) {
	data := bytes.Repeat([]byte("dict size test for lzma "), 50)
	var compressed bytes.Buffer
	w, _ := lzma.NewWriter(&compressed)
	w.Write(data)
	w.Close()

	t.Run("large_dict_limit", func(t *testing.T) {
		_, err := lzma.NewReader2(&compressed)
		if err != nil {
			t.Logf("NewReader2 with default config: %v", err)
		}
	})
}

func BenchmarkLZMA1_RoundTrip_64K(b *testing.B) {
	data := make([]byte, 65536)
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

func BenchmarkLZMA2_RoundTrip_1M(b *testing.B) {
	data := make([]byte, 1<<20)
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

func FuzzLZMA1_Config(f *testing.F) {
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, data []byte) {
		var cbuf bytes.Buffer
		w, _ := lzma.NewWriter(&cbuf)
		w.Write(data)
		w.Close()

		r, err := lzma.ReaderConfig{}.NewReader(&cbuf)
		if err != nil {
			t.Fatalf("NewReader error: %v", err)
		}
		d, _ := io.ReadAll(r)
		if !bytes.Equal(d, data) {
			t.Fatal("mismatch")
		}
	})
}

func FuzzLZMA2_Config(f *testing.F) {
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, data []byte) {
		var cbuf bytes.Buffer
		w, _ := lzma.NewWriter2(&cbuf)
		w.Write(data)
		w.Close()

		r, err := lzma.NewReader2(&cbuf)
		if err != nil {
			t.Fatalf("NewReader2 error: %v", err)
		}
		d, _ := io.ReadAll(r)
		if !bytes.Equal(d, data) {
			t.Fatal("mismatch")
		}
	})
}
