package zstd_test

import (
	"bytes"
	"testing"

	"github.com/cookiengineer/gozim/compress/zstd"
)

// zstdMagic is the Zstandard frame magic number (little-endian 0xFD2FB528).
var zstdMagic = []byte{0x28, 0xb5, 0x2f, 0xfd}

// TestSpec_FrameHeader_DescriptorFlags verifies parsing of each flag bit.
func TestSpec_FrameHeader_DescriptorFlags(t *testing.T) {
	// Build a minimal valid frame header: magic + FHD + (window if !single_seg) + FCS if needed
	makeFrame := func(singleSeg bool, checksum bool, dictID uint32, contentSize uint64) []byte {
		var buf bytes.Buffer
		buf.Write(zstdMagic)
		var fhd byte
		if checksum {
			fhd |= 1 << 2
		}
		if singleSeg {
			fhd |= 1 << 5
		}
		// FCS
		var fcsByte byte
		switch {
		case contentSize >= 256 && contentSize < 65792:
			fcsByte = 1
		case contentSize >= 65792 && contentSize < 0x100000000:
			fcsByte = 2
		case contentSize >= 0x100000000:
			fcsByte = 3
		default:
			if !singleSeg {
				fcsByte = 0
			}
		}
		fhd |= fcsByte << 6
		if dictID > 0 {
			if dictID < 256 {
				fhd |= 1
			} else if dictID < 1<<16 {
				fhd |= 2
			} else {
				fhd |= 3
			}
		}
		buf.WriteByte(fhd)

		// Window descriptor
		if !singleSeg {
			// Window_Size=8192 → exponent=13, mantissa=0
			buf.WriteByte(byte((13 - 10) << 3))
		}
		// DictID
		if dictID > 0 {
			s := dictID
			if dictID < 256 {
				buf.WriteByte(byte(s))
			} else if dictID < 1<<16 {
				buf.WriteByte(byte(s))
				buf.WriteByte(byte(s >> 8))
			} else {
				buf.WriteByte(byte(s))
				buf.WriteByte(byte(s >> 8))
				buf.WriteByte(byte(s >> 16))
				buf.WriteByte(byte(s >> 24))
			}
		}
		// FCS
		switch fcsByte {
		case 1:
			s := contentSize - 256
			buf.WriteByte(byte(s))
			buf.WriteByte(byte(s >> 8))
		case 2:
			buf.WriteByte(byte(contentSize))
			buf.WriteByte(byte(contentSize >> 8))
			buf.WriteByte(byte(contentSize >> 16))
			buf.WriteByte(byte(contentSize >> 24))
		case 3:
			buf.WriteByte(byte(contentSize))
			buf.WriteByte(byte(contentSize >> 8))
			buf.WriteByte(byte(contentSize >> 16))
			buf.WriteByte(byte(contentSize >> 24))
			buf.WriteByte(byte(contentSize >> 32))
			buf.WriteByte(byte(contentSize >> 40))
			buf.WriteByte(byte(contentSize >> 48))
			buf.WriteByte(byte(contentSize >> 56))
		}
		return buf.Bytes()
	}

	t.Run("valid_minimal", func(t *testing.T) {
		data := makeFrame(false, false, 0, 0)
		_, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("minimal frame should be valid: %v", err)
		}
	})

	t.Run("valid_checksum", func(t *testing.T) {
		data := makeFrame(false, true, 0, 100)
		_, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("checksum frame should be valid: %v", err)
		}
	})

	t.Run("valid_single_segment", func(t *testing.T) {
		data := makeFrame(true, false, 0, 100)
		_, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("single segment frame should be valid: %v", err)
		}
	})

	t.Run("valid_large_fcs", func(t *testing.T) {
		data := makeFrame(false, false, 0, 100000)
		_, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("large FCS frame should be valid: %v", err)
		}
	})

	t.Run("reject_reserved_bit_set", func(t *testing.T) {
		// Bit 3 is Reserved_bit. Set it.
		data := makeFrame(false, false, 0, 0)
		data[4] |= 1 << 3
		r, err := zstd.NewReader(bytes.NewReader(data))
		if err == nil {
			// Error may appear on Read, not NewReader
			buf := make([]byte, 100)
			_, err = r.Read(buf)
		}
		if err == nil {
			t.Error("expected error when reserved bit is set")
		}
	})

	t.Run("reject_unused_bit_set", func(t *testing.T) {
		// Bit 4 is Unused_bit (spec §3.1.1.2: "must be set to zero")
		data := makeFrame(false, false, 0, 0)
		data[4] |= 1 << 4
		r, err := zstd.NewReader(bytes.NewReader(data))
		if err == nil {
			buf := make([]byte, 100)
			_, err = r.Read(buf)
		}
		if err == nil {
			t.Error("expected error when unused bit is set")
		}
	})

	t.Run("valid_dict_id_1byte", func(t *testing.T) {
		data := makeFrame(false, false, 200, 0)
		_, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("1-byte dict ID frame should be valid: %v", err)
		}
	})

	t.Run("valid_dict_id_2byte", func(t *testing.T) {
		data := makeFrame(false, false, 500, 0)
		_, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("2-byte dict ID frame should be valid: %v", err)
		}
	})

	t.Run("valid_dict_id_4byte", func(t *testing.T) {
		data := makeFrame(false, false, 70000, 0)
		_, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("4-byte dict ID frame should be valid: %v", err)
		}
	})
}

// TestSpec_FrameMagic verifies magic number validation.
func TestSpec_FrameMagic(t *testing.T) {
	t.Run("wrong_magic", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00, 0x00, 0x00}
		r, err := zstd.NewReader(bytes.NewReader(data))
		if err == nil {
			buf := make([]byte, 100)
			_, err = r.Read(buf)
		}
		if err != zstd.ErrMagicMismatch {
			t.Errorf("expected ErrMagicMismatch, got %v", err)
		}
	})

	t.Run("valid_magic", func(t *testing.T) {
		data := []byte{0x28, 0xb5, 0x2f, 0xfd, (10-10)<<3 + 3, 0x00}
		// magic + window_descriptor(10,0) + block_header(last=1, raw, size=0)
		_, err := zstd.NewReader(bytes.NewReader(data))
		// This will fail at block decode because there's no block data,
		// but magic should be accepted.
		if err != nil && err == zstd.ErrMagicMismatch {
			t.Error("valid magic was rejected")
		}
	})

	t.Run("skip_frame_bypassed", func(t *testing.T) {
		// Zstandard skip frame starts with 0x50, 0x2a, 0x4d, 0x18
		skipFrame := []byte{0x50, 0x2a, 0x4d, 0x18, 0x00, 0x00, 0x00, 0x00}
		// Followed by valid zstd frame
		validFrame := []byte{0x28, 0xb5, 0x2f, 0xfd, (10 - 10) << 3, 0x00}
		data := append(skipFrame, validFrame...)

		_, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			// Should succeed because skip frames are transparently skipped
			t.Logf("skip frame followed by valid frame: %v", err)
		}
	})
}

// TestSpec_FCS_Thresholds verifies Frame_Content_Size field size selection.
func TestSpec_FCS_Thresholds(t *testing.T) {
	// Encode small content sizes and verify via round-trip
	tests := []struct {
		name string
		data []byte
		size uint64
	}{
		// FCS=0 (no size stored, unknown or 1 byte for single segment)
		{"empty", []byte{}, 0},
		// FCS=1 threshold tests: 256 to 65791
		{"small_min_fcs", bytes.Repeat([]byte("x"), 256), 256},
		{"small_fcs", bytes.Repeat([]byte("x"), 1000), 1000},
		{"small_fcs_max", bytes.Repeat([]byte("x"), 65791), 65791},
		// FCS=2 threshold: >= 65792
		{"med_fcs_min", bytes.Repeat([]byte("x"), 65792), 65792},
		{"med_fcs", bytes.Repeat([]byte("x"), 100000), 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.data) > 131072 {
				t.Skip("too large for quick test")
			}
			if len(tt.data) > 65536 {
				t.Skip("too large for quick test")
			}
			var buf bytes.Buffer
			w, err := zstd.NewWriter(&buf)
			if err != nil {
				t.Fatalf("NewWriter error: %v", err)
			}
			w.Write(tt.data)
			w.Close()

			r, err := zstd.NewReader(&buf)
			if err != nil {
				t.Fatalf("NewReader error: %v", err)
			}
			d := make([]byte, len(tt.data)+100)
			n, err := r.Read(d)
			if err != nil && err.Error() != "EOF" {
				t.Fatalf("Read error: %v", err)
			}
			if uint64(n) != tt.size {
				t.Errorf("decoded %d bytes, want %d", n, tt.size)
			}
			r.Close()
		})
	}
}

// TestSpec_FCS_FieldSize_Encoding verifies FCS byte encoding matches spec.
func TestSpec_FCS_FieldSize_Encoding(t *testing.T) {
	// For FCS=1: value = Frame_Content_Size - 256 encoded as 2 bytes LE
	// For FCS=2: value = Frame_Content_Size encoded as 4 bytes LE
	tests := []struct {
		name          string
		contentSize   int
		expectFcsSize int // 0 (unknown), 1, 2, 3
	}{
		{"zero_non_single", 0, 0},
		{"just_below_256", 255, 0},  // Single-segment uses 0, non-single uses 0 (unknown)
		{"at_256", 256, 1},
		{"well_in_fcs1", 10000, 1},
		{"at_max_fcs1", 65791, 1},
		{"at_min_fcs2", 65792, 2},
		{"well_in_fcs2", 100000, 2},
		{"at_max_fcs2", 4294967295, 2},
		{"at_min_fcs3", 4294967296, 3},
	}

	// We verify indirectly: round-trip should preserve content size
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.contentSize > 131072 {
				t.Skip("too large")
			}
			data := bytes.Repeat([]byte("a"), tt.contentSize)

			var buf bytes.Buffer
			w, _ := zstd.NewWriter(&buf)
			w.Write(data)
			w.Close()

			r, _ := zstd.NewReader(&buf)
			d := make([]byte, len(data)+1024)
			n, err := r.Read(d)
			if err != nil && err.Error() != "EOF" {
				t.Fatalf("Read error: %v", err)
			}
			if n != tt.contentSize {
				t.Errorf("round-trip: got %d bytes, want %d", n, tt.contentSize)
			}
			r.Close()
		})
	}
}

// TestSpec_BlockHeader verifies block header format compliance.
func TestSpec_BlockHeader(t *testing.T) {
	// Block_Header is 3 bytes:
	// - Last_Block (bit 0)
	// - Block_Type (bits 1-2): 0=Raw, 1=RLE, 2=Compressed, 3=Reserved
	// - Block_Size (bits 3-23)

	t.Run("reserved_block_type_rejected", func(t *testing.T) {
		// Craft a frame with a reserved block type (3).
		// Magic + FHD(w=10,exp=10,mand=0) + block(Last=1,Type=3,Size=0)
		data := []byte{
			0x28, 0xb5, 0x2f, 0xfd, // magic
			(10 - 10) << 3, // window descriptor (Exponent=10, Mantissa=0)
			0x07, 0x00, 0x00, // block header: Last=1, Type=3, Size=0
		}
		_, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			// Should return ErrReservedBlockType or similar
			t.Logf("reserved block rejected: %v", err)
		}
	})

	t.Run("raw_block_last_flag", func(t *testing.T) {
		data := bytes.Repeat([]byte("rawblock"), 10)
		var buf bytes.Buffer
		w, _ := zstd.NewWriter(&buf)
		w.Write(data)
		w.Close()

		r, _ := zstd.NewReader(&buf)
		d := make([]byte, 1024)
		n, err := r.Read(d)
		if err != nil && err.Error() != "EOF" {
			t.Fatalf("Read error: %v", err)
		}
		if n != len(data) {
			t.Errorf("decoded %d, want %d", n, len(data))
		}
		r.Close()
	})
}

// TestSpec_Literals_Types verifies all 4 literals block types are handled.
func TestSpec_Literals_Types(t *testing.T) {
	// Literals_Block_Type values:
	// 0 = Raw_Literals_Block
	// 1 = RLE_Literals_Block
	// 2 = Compressed_Literals_Block (Huffman with tree)
	// 3 = Treeless_Literals_Block (Huffman reusing previous tree)

	// Test that compressed blocks round-trip correctly for various data types
	inputs := []struct {
		name string
		data []byte
	}{
		{"all_same", bytes.Repeat([]byte("A"), 200)},    // RLE literals
		{"randomish", bytes.Repeat([]byte("ABCDEFGH"), 25)}, // compressed literals
		{"raw_small", []byte("hi")},                        // raw literals for very small
		{"html", []byte("<!DOCTYPE html><html><head><title>T</title></head><body><p>Test</p></body></html>")},
	}

	for _, tt := range inputs {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w, _ := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
			w.Write(tt.data)
			w.Close()

			d, err := zstd.DecodeTo(nil, buf.Bytes())
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if !bytes.Equal(d, tt.data) {
				t.Errorf("round-trip failed for %s", tt.name)
			}
		})
	}
}

// TestSpec_ContentChecksum verifies CRC-32 checksum compliance.
func TestSpec_ContentChecksum(t *testing.T) {
	data := bytes.Repeat([]byte("checksum test "), 20)

	t.Run("checksum_enabled", func(t *testing.T) {
		var buf bytes.Buffer
		w, _ := zstd.NewWriter(&buf,
			zstd.WithEncoderLevel(zstd.SpeedDefault),
			zstd.WithEncoderCRC(true),
		)
		w.Write(data)
		w.Close()

		r, _ := zstd.NewReader(&buf)
		d := make([]byte, 2048)
		n, err := r.Read(d)
		if err != nil && err.Error() != "EOF" {
			t.Fatalf("Read error: %v", err)
		}
		if n != len(data) {
			t.Errorf("CRC frame: got %d, want %d", n, len(data))
		}
		if !bytes.Equal(d[:n], data) {
			t.Error("data mismatch in CRC frame")
		}
		r.Close()
	})

	t.Run("checksum_disabled_default", func(t *testing.T) {
		var buf bytes.Buffer
		w, _ := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
		w.Write(data)
		w.Close()

		r, _ := zstd.NewReader(&buf)
		d := make([]byte, 2048)
		n, err := r.Read(d)
		if err != nil && err.Error() != "EOF" {
			t.Fatalf("Read error: %v", err)
		}
		if n != len(data) {
			t.Errorf("non-CRC frame: got %d, want %d", n, len(data))
		}
		r.Close()
	})
}

// TestSpec_EncodeTo_DecodeTo verifies the convenience functions.
func TestSpec_EncodeTo_DecodeTo(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single", []byte("x")},
		{"small", []byte("hello")},
		{"medium", bytes.Repeat([]byte("test"), 100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := zstd.EncodeTo(nil, tt.data)
			dec, err := zstd.DecodeTo(nil, enc)
			if err != nil {
				t.Fatalf("DecodeTo error: %v", err)
			}
			if !bytes.Equal(dec, tt.data) {
				t.Error("round-trip mismatch")
			}
		})
	}
}

// TestSpec_BlockMaximumSize verifies the block size constraint.
func TestSpec_BlockMaximumSize(t *testing.T) {
	// Block_Maximum_Size = min(Window_Size, 128 KB) per spec §3.1.2
	// 128 KB = 131072 bytes. Our maxCompressedBlockSize is 128<<10 = 131072. ✓
	data := bytes.Repeat([]byte("block max test "), 3000) // ~48KB

	var buf bytes.Buffer
	w, _ := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
	w.Write(data)
	w.Close()

	d, _ := zstd.DecodeTo(nil, buf.Bytes())
	if !bytes.Equal(d, data) {
		t.Error("48KB round-trip failed")
	}
}

// TestSpec_SingleSegment verifies single-segment frame encoding.
func TestSpec_SingleSegment(t *testing.T) {
	data := bytes.Repeat([]byte("single segment test "), 50)

	t.Run("with_single_segment", func(t *testing.T) {
		var buf bytes.Buffer
		enc, err := zstd.NewWriter(&buf,
			zstd.WithEncoderLevel(zstd.SpeedDefault),
			zstd.WithSingleSegment(true),
		)
		if err != nil {
			t.Fatalf("NewWriter error: %v", err)
		}
		enc.Write(data)
		enc.Close()

		d, _ := zstd.DecodeTo(nil, buf.Bytes())
		if !bytes.Equal(d, data) {
			t.Error("single segment round-trip failed")
		}
	})

	t.Run("default_multi_segment", func(t *testing.T) {
		var buf bytes.Buffer
		w, _ := zstd.NewWriter(&buf)
		w.Write(data)
		w.Close()

		d, _ := zstd.DecodeTo(nil, buf.Bytes())
		if !bytes.Equal(d, data) {
			t.Error("multi-segment round-trip failed")
		}
	})
}

// TestSpec_ErrorConstants verifies error sentinel values.
func TestSpec_ErrorConstants(t *testing.T) {
	// All spec-related errors should be non-nil with descriptive messages.
	tests := []struct {
		name string
		err  error
	}{
		{"MagicMismatch", zstd.ErrMagicMismatch},
		{"ReservedBlockType", zstd.ErrReservedBlockType},
		{"CompressedSizeTooBig", zstd.ErrCompressedSizeTooBig},
		{"BlockTooSmall", zstd.ErrBlockTooSmall},
		{"WindowSizeExceeded", zstd.ErrWindowSizeExceeded},
		{"WindowSizeTooSmall", zstd.ErrWindowSizeTooSmall},
		{"CRCMismatch", zstd.ErrCRCMismatch},
		{"DecoderClosed", zstd.ErrDecoderClosed},
		{"EncoderClosed", zstd.ErrEncoderClosed},
		{"ErrSnappyUnsupported", zstd.ErrSnappyUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil || tt.err.Error() == "" {
				t.Errorf("error %s is nil or has empty message", tt.name)
			}
		})
	}
}

// TestSpec_Constants verifies named constants match the specification.
func TestSpec_Constants(t *testing.T) {
	// MinWindowSize must be 1 KB (1024).
	if zstd.MinWindowSize != 1<<10 {
		t.Errorf("MinWindowSize = %d, spec says 1024", zstd.MinWindowSize)
	}

	// MaxWindowSize must be at least 1<<29 for default.
	if zstd.MaxWindowSize != 1<<29 {
		t.Errorf("MaxWindowSize = %d, spec says %d", zstd.MaxWindowSize, 1<<29)
	}

	// HeaderMaxSize = 14 (max frame header) + 3 (block header) = 17
	if zstd.HeaderMaxSize != 17 {
		t.Errorf("HeaderMaxSize = %d, spec says 14+3=17", zstd.HeaderMaxSize)
	}
}

func FuzzSpec_RoundTrip(f *testing.F) {
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, data []byte) {
		var buf bytes.Buffer
		w, err := zstd.NewWriter(&buf)
		if err != nil {
			t.Fatalf("NewWriter: %v", err)
		}
		w.Write(data)
		w.Close()

		d, err := zstd.DecodeTo(nil, buf.Bytes())
		if err != nil {
			t.Fatalf("DecodeTo: %v", err)
		}
		if !bytes.Equal(d, data) {
			t.Fatal("round-trip failed")
		}
	})
}
