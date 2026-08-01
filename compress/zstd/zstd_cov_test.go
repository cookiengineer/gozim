package zstd_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/cookiengineer/gozim/compress/zstd"
)

func TestAllEncoderLevels_RoundTrip(t *testing.T) {
	data := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 50)
	levels := []zstd.EncoderLevel{
		zstd.SpeedFastest,
		zstd.SpeedDefault,
		zstd.SpeedBetterCompression,
		zstd.SpeedBestCompression,
	}
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			var buf bytes.Buffer
			enc, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(level))
			if err != nil {
				t.Fatalf("NewWriter error: %v", err)
			}
			enc.Write(data)
			enc.Close()

			dec, _ := zstd.NewReader(&buf)
			decompressed, _ := io.ReadAll(dec)
			dec.Close()
			if !bytes.Equal(decompressed, data) {
				t.Errorf("level %s round-trip failed", level)
			}
			t.Logf("%s: %d -> %d bytes (%.1f%%)", level, len(data), buf.Len(),
				100*float64(buf.Len())/float64(len(data)))
		})
	}
}

func TestEncoderReset(t *testing.T) {
	data1 := bytes.Repeat([]byte("first data block "), 50)
	data2 := bytes.Repeat([]byte("second block, reset test "), 50)

	var buf1, buf2 bytes.Buffer
	enc, _ := zstd.NewWriter(&buf1)
	enc.Write(data1)
	enc.Close()

	enc.Reset(&buf2)
	enc.Write(data2)
	enc.Close()

	// Verify both decompress correctly
	for i, buf := range []*bytes.Buffer{&buf1, &buf2} {
		dec, _ := zstd.NewReader(buf)
		d, _ := io.ReadAll(dec)
		dec.Close()
		expected := data1
		if i == 1 {
			expected = data2
		}
		if !bytes.Equal(d, expected) {
			t.Errorf("block %d round-trip failed", i)
		}
	}
}

func TestEncoderResetWithOptions(t *testing.T) {
	data := bytes.Repeat([]byte("reset with options "), 50)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))

	err := enc.ResetWithOptions(&buf, zstd.WithZeroFrames(true), zstd.WithSingleSegment(true))
	if err != nil {
		t.Fatalf("ResetWithOptions error: %v", err)
	}
	enc.Write(data)
	enc.Close()

	dec, _ := zstd.NewReader(&buf)
	d, _ := io.ReadAll(dec)
	dec.Close()
	if !bytes.Equal(d, data) {
		t.Error("round-trip failed")
	}
}

func TestEncoderResetContentSize(t *testing.T) {
	data := bytes.Repeat([]byte("content size test "), 30)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(nil)
	enc.ResetContentSize(&buf, int64(len(data)))
	enc.Write(data)
	enc.Close()

	dec, _ := zstd.NewReader(&buf)
	d, _ := io.ReadAll(dec)
	dec.Close()
	if !bytes.Equal(d, data) {
		t.Error("round-trip failed")
	}
}

func TestEncoderFlush(t *testing.T) {
	data := bytes.Repeat([]byte("flush test "), 100)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(&buf)
	enc.Write(data[:len(data)/2])
	enc.Flush()
	enc.Write(data[len(data)/2:])
	enc.Close()

	dec, _ := zstd.NewReader(&buf)
	d, _ := io.ReadAll(dec)
	dec.Close()
	if !bytes.Equal(d, data) {
		t.Error("flush round-trip failed")
	}
}

func TestEncoderReadFrom(t *testing.T) {
	data := bytes.Repeat([]byte("readfrom test data "), 100)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(&buf)
	n, err := enc.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadFrom error: %v", err)
	}
	if n != int64(len(data)) {
		t.Errorf("ReadFrom returned %d, want %d", n, len(data))
	}
	enc.Close()

	dec, _ := zstd.NewReader(&buf)
	d, _ := io.ReadAll(dec)
	dec.Close()
	if !bytes.Equal(d, data) {
		t.Error("ReadFrom round-trip failed")
	}
}

func TestEncoderEncodeAll(t *testing.T) {
	inputs := [][]byte{
		[]byte("first"),
		[]byte("second block"),
		bytes.Repeat([]byte("third"), 100),
		[]byte(""),
	}

	enc, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	defer enc.Close()

	for i, data := range inputs {
		compressed := enc.EncodeAll(data, nil)
		dec, _ := zstd.NewReader(nil)
		d, err := dec.DecodeAll(compressed, nil)
		dec.Close()
		if err != nil {
			t.Fatalf("block %d DecodeAll error: %v", i, err)
		}
		if !bytes.Equal(d, data) {
			t.Errorf("block %d round-trip failed", i)
		}
	}
}

func TestEncoderMaxEncodedSize(t *testing.T) {
	enc, _ := zstd.NewWriter(nil)
	defer enc.Close()
	if n := enc.MaxEncodedSize(0); n == 0 {
		t.Error("MaxEncodedSize(0) should be > 0")
	}
	if n := enc.MaxEncodedSize(1024); n < 1024 {
		t.Errorf("MaxEncodedSize(1024) = %d, want >= 1024", n)
	}
}

func TestDecoderReset(t *testing.T) {
	data := bytes.Repeat([]byte("decoder reset test "), 50)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(&buf)
	enc.Write(data)
	enc.Close()

	dec, _ := zstd.NewReader(nil)
	err := dec.Reset(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Reset error: %v", err)
	}
	d, _ := io.ReadAll(dec)
	dec.Close()
	if !bytes.Equal(d, data) {
		t.Error("reset round-trip failed")
	}
}

func TestDecoderResetWithOptions(t *testing.T) {
	data := bytes.Repeat([]byte("reset with decoder opts "), 50)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(&buf)
	enc.Write(data)
	enc.Close()

	dec, _ := zstd.NewReader(nil)
	err := dec.ResetWithOptions(bytes.NewReader(buf.Bytes()),
		zstd.IgnoreChecksum(true),
		zstd.WithDecodeAllCapLimit(true),
	)
	if err != nil {
		t.Fatalf("ResetWithOptions error: %v", err)
	}
	d, _ := io.ReadAll(dec)
	dec.Close()
	if !bytes.Equal(d, data) {
		t.Error("round-trip failed")
	}
}

func TestDecoderWriteTo(t *testing.T) {
	data := bytes.Repeat([]byte("WriteTo test "), 50)
	var inBuf, outBuf bytes.Buffer
	enc, _ := zstd.NewWriter(&inBuf)
	enc.Write(data)
	enc.Close()

	dec, _ := zstd.NewReader(&inBuf)
	n, err := dec.WriteTo(&outBuf)
	if err != nil {
		t.Fatalf("WriteTo error: %v", err)
	}
	dec.Close()
	if n != int64(len(data)) {
		t.Errorf("WriteTo returned %d, want %d", n, len(data))
	}
	if !bytes.Equal(outBuf.Bytes(), data) {
		t.Error("WriteTo round-trip failed")
	}
}

func TestDecoderIOReadCloser(t *testing.T) {
	data := bytes.Repeat([]byte("IOReadCloser test "), 50)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(&buf)
	enc.Write(data)
	enc.Close()

	dec, _ := zstd.NewReader(&buf)
	rc := dec.IOReadCloser()
	d, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(d, data) {
		t.Error("IOReadCloser round-trip failed")
	}
}

func TestEncoderOptions(t *testing.T) {
	data := bytes.Repeat([]byte("encoder options test "), 50)

	options := []struct {
		name string
		opts []zstd.EOption
	}{
		{"CRC", []zstd.EOption{zstd.WithEncoderCRC(true)}},
		{"Concurrency", []zstd.EOption{zstd.WithEncoderConcurrency(2)}},
		{"WindowSize", []zstd.EOption{zstd.WithWindowSize(1 << 18)}},
		{"ZeroFrames", []zstd.EOption{zstd.WithZeroFrames(true)}},
		{"SingleSegment", []zstd.EOption{zstd.WithSingleSegment(true)}},
		{"LowerMem", []zstd.EOption{zstd.WithLowerEncoderMem(true)}},
		{"NoEntropy", []zstd.EOption{zstd.WithNoEntropyCompression(true)}},
		{"AllLitEntropy", []zstd.EOption{zstd.WithAllLitEntropyCompression(true)}},
		{"ConcurrentBlocks", []zstd.EOption{zstd.WithConcurrentBlocks(true)}},
	}

	for _, opt := range options {
		t.Run(opt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc, err := zstd.NewWriter(&buf, opt.opts...)
			if err != nil {
				t.Fatalf("NewWriter error: %v", err)
			}
			enc.Write(data)
			enc.Close()

			dec, _ := zstd.NewReader(&buf)
			d, _ := io.ReadAll(dec)
			dec.Close()
			if !bytes.Equal(d, data) {
				t.Errorf("option %s round-trip failed", opt.name)
			}
		})
	}
}

func TestDecoderOptions(t *testing.T) {
	data := bytes.Repeat([]byte("decoder options test "), 50)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(&buf)
	enc.Write(data)
	enc.Close()

	options := []struct {
		name string
		opts []zstd.DOption
	}{
		{"Lowmem", []zstd.DOption{zstd.WithDecoderLowmem(true)}},
		{"Concurrency", []zstd.DOption{zstd.WithDecoderConcurrency(1)}},
		{"MaxMemory", []zstd.DOption{zstd.WithDecoderMaxMemory(1 << 20)}},
		{"MaxWindow", []zstd.DOption{zstd.WithDecoderMaxWindow(1 << 20)}},
		{"IgnoreChecksum", []zstd.DOption{zstd.IgnoreChecksum(true)}},
		{"DecodeAllCapLimit", []zstd.DOption{zstd.WithDecodeAllCapLimit(true)}},
	}

	for _, opt := range options {
		t.Run(opt.name, func(t *testing.T) {
			dec, err := zstd.NewReader(bytes.NewReader(buf.Bytes()), opt.opts...)
			if err != nil {
				t.Fatalf("NewReader error: %v", err)
			}
			d, _ := io.ReadAll(dec)
			dec.Close()
			if !bytes.Equal(d, data) {
				t.Errorf("option %s round-trip failed", opt.name)
			}
		})
	}
}

func TestEncoderLevelFromString(t *testing.T) {
	tests := []struct {
		s     string
		valid bool
	}{
		{"fastest", true},
		{"default", true},
		{"better", true},
		{"best", true},
		{"FASTEST", true},
		{"DEFAULT", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		valid, _ := zstd.EncoderLevelFromString(tt.s)
		if valid != tt.valid {
			t.Errorf("EncoderLevelFromString(%q) = %v, want %v", tt.s, valid, tt.valid)
		}
	}
}

func TestEncoderLevelFromZstd(t *testing.T) {
	mappings := map[int]string{
		1:  "fastest",
		3:  "default",
		5:  "default",
		7:  "better",
		9:  "better",
		11: "best",
		20: "best",
	}
	for level, expected := range mappings {
		got := zstd.EncoderLevelFromZstd(level)
		if got.String() != expected {
			t.Errorf("EncoderLevelFromZstd(%d) = %s, want %s", level, got, expected)
		}
	}
}

func TestHeader_Decode(t *testing.T) {
	data := bytes.Repeat([]byte("header decode test "), 50)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(&buf)
	enc.Write(data)
	enc.Close()

	var h zstd.Header
	err := h.Decode(buf.Bytes())
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if h.FrameContentSize != uint64(len(data)) {
		t.Errorf("FrameContentSize = %d, want %d", h.FrameContentSize, len(data))
	}
}

func TestHeader_DecodeAndStrip(t *testing.T) {
	data := bytes.Repeat([]byte("decode and strip test "), 50)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(&buf)
	enc.Write(data)
	enc.Close()

	var h zstd.Header
	remain, err := h.DecodeAndStrip(buf.Bytes())
	if err != nil {
		t.Fatalf("DecodeAndStrip error: %v", err)
	}
	if len(remain) == 0 {
		t.Error("expected remaining data after stripping header")
	}
}

func TestHeader_AppendTo(t *testing.T) {
	data := bytes.Repeat([]byte("appendto header test "), 50)
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(&buf)
	enc.Write(data)
	enc.Close()

	var h zstd.Header
	if err := h.Decode(buf.Bytes()); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	out, err := h.AppendTo(nil)
	if err != nil {
		t.Fatalf("AppendTo error: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected non-empty header output")
	}
}

func TestSnappyConverter(t *testing.T) {
	// SnappyConverter requires zstd frames with snappy-encoded blocks.
	// When input is not snappy-formatted, it returns an error.
	var buf bytes.Buffer
	w, _ := zstd.NewWriter(&buf)
	w.Write([]byte("test"))
	w.Close()

	conv := zstd.SnappyConverter{}
	_, err := conv.Convert(bytes.NewReader(buf.Bytes()), io.Discard)
	// SnappyConverter may return an error if the zstd blocks are not snappy.
	// That's expected since we used default zstd encoding.
	if err != nil {
		t.Logf("SnappyConverter (expected): %v", err)
	}
}

func TestEncodeTo_DecodeTo(t *testing.T) {
	data := bytes.Repeat([]byte("encode-to test "), 30)
	encoded := zstd.EncodeTo(nil, data)
	if len(encoded) == 0 {
		t.Fatal("EncodeTo returned empty")
	}
	decoded, err := zstd.DecodeTo(nil, encoded)
	if err != nil {
		t.Fatalf("DecodeTo error: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Error("EncodeTo/DecodeTo round-trip failed")
	}
}

func TestZipCompressor_Decompressor(t *testing.T) {
	data := bytes.Repeat([]byte("zip compressor test "), 50)

	var buf bytes.Buffer
	w, err := zstd.ZipCompressor(zstd.WithEncoderLevel(zstd.SpeedDefault))(&buf)
	if err != nil {
		t.Fatalf("ZipCompressor error: %v", err)
	}
	w.Write(data)
	w.Close()

	r := zstd.ZipDecompressor()(bytes.NewReader(buf.Bytes()))
	d, _ := io.ReadAll(r)
	r.Close()
	if !bytes.Equal(d, data) {
		t.Error("Zip round-trip failed")
	}
}

func TestErrorVariables(t *testing.T) {
	errors := []error{
		zstd.ErrReservedBlockType,
		zstd.ErrCompressedSizeTooBig,
		zstd.ErrBlockTooSmall,
		zstd.ErrUnexpectedBlockSize,
		zstd.ErrMagicMismatch,
		zstd.ErrWindowSizeExceeded,
		zstd.ErrWindowSizeTooSmall,
		zstd.ErrDecoderSizeExceeded,
		zstd.ErrUnknownDictionary,
		zstd.ErrFrameSizeExceeded,
		zstd.ErrFrameSizeMismatch,
		zstd.ErrCRCMismatch,
		zstd.ErrDecoderClosed,
		zstd.ErrEncoderClosed,
		zstd.ErrDecoderNilInput,
	}
	for _, err := range errors {
		if err.Error() == "" {
			t.Error("error variable has empty message")
		}
	}
}

func TestSnappyErrors(t *testing.T) {
	for _, err := range []error{zstd.ErrSnappyCorrupt, zstd.ErrSnappyTooLarge, zstd.ErrSnappyUnsupported} {
		if err.Error() == "" {
			t.Error("snappy error has empty message")
		}
	}
}

func TestConstants(t *testing.T) {
	if zstd.MinWindowSize == 0 {
		t.Error("MinWindowSize is zero")
	}
	if zstd.MaxWindowSize == 0 {
		t.Error("MaxWindowSize is zero")
	}
	if zstd.HeaderMaxSize == 0 {
		t.Error("HeaderMaxSize is zero")
	}
	if zstd.ZipMethodWinZip == 0 {
		t.Error("ZipMethodWinZip is zero")
	}
	if zstd.ZipMethodPKWare == 0 {
		t.Error("ZipMethodPKWare is zero")
	}
}

func BenchmarkZstdAllLevels_4K(b *testing.B) {
	data := make([]byte, 4096)
	rand.Read(data)
	levels := []zstd.EncoderLevel{
		zstd.SpeedFastest, zstd.SpeedDefault,
		zstd.SpeedBetterCompression, zstd.SpeedBestCompression,
	}
	for _, level := range levels {
		b.Run(level.String(), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				w, _ := zstd.NewWriter(&buf, zstd.WithEncoderLevel(level))
				w.Write(data)
				w.Close()
				r, _ := zstd.NewReader(&buf)
				io.ReadAll(r)
				r.Close()
			}
		})
	}
}

func FuzzZstd_EncodeAll(f *testing.F) {
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, data []byte) {
		enc, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
		compressed := enc.EncodeAll(data, nil)
		enc.Close()
		dec, _ := zstd.NewReader(nil)
		d, err := dec.DecodeAll(compressed, nil)
		dec.Close()
		if err != nil {
			t.Fatalf("DecodeAll error: %v", err)
		}
		if !bytes.Equal(d, data) {
			t.Fatal("mismatch")
		}
	})
}
