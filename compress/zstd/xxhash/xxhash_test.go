package xxhash_test

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"hash"
	"testing"

	"github.com/cookiengineer/gozim/compress/zstd/xxhash"
)

func TestSum64(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single_byte", []byte{0x00}},
		{"hello", []byte("hello world")},
		{"ascii_all", []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")},
		{"repeated", bytes.Repeat([]byte("x"), 100)},
		{"binary", bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03}, 64)},
		{"32_bytes", make([]byte, 32)},
		{"64_bytes", []byte("the quick brown fox jumps over the lazy dog, the quick brown fox")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h1 := xxhash.Sum64(tt.data)

			d := xxhash.New()
			d.Write(tt.data)
			h2 := d.Sum64()

			if h1 != h2 {
				t.Errorf("Sum64=%x, Digest.Sum64()=%x", h1, h2)
			}
		})
	}
}

func TestSum64_Deterministic(t *testing.T) {
	data := []byte("deterministic test data")
	expected := xxhash.Sum64(data)
	for i := 0; i < 100; i++ {
		if got := xxhash.Sum64(data); got != expected {
			t.Errorf("iteration %d: %x != %x", i, got, expected)
		}
	}
}

func TestSum64String(t *testing.T) {
	testCases := []string{
		"",
		"hello",
		"the quick brown fox",
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"日本語テスト",
	}

	for _, s := range testCases {
		t.Run(fmt.Sprintf("len=%d", len(s)), func(t *testing.T) {
			h1 := xxhash.Sum64String(s)
			h2 := xxhash.Sum64([]byte(s))
			if h1 != h2 {
				t.Errorf("Sum64String=%x, Sum64=%x", h1, h2)
			}
		})
	}
}

func TestDigest_Write(t *testing.T) {
	tests := []struct {
		name    string
		chunks  [][]byte
		combined []byte
	}{
		{
			"one_chunk",
			[][]byte{[]byte("hello world")},
			[]byte("hello world"),
		},
		{
			"two_chunks",
			[][]byte{[]byte("hello "), []byte("world")},
			[]byte("hello world"),
		},
		{
			"many_small",
			[][]byte{
				{0},
				{1},
				{2},
				{3},
				{4},
				{5},
				{6},
				{7},
			},
			[]byte{0, 1, 2, 3, 4, 5, 6, 7},
		},
		{
			"cross_block_boundary",
			[][]byte{
				bytes.Repeat([]byte("a"), 30),
				bytes.Repeat([]byte("b"), 30),
			},
			bytes.Join([][]byte{bytes.Repeat([]byte("a"), 30), bytes.Repeat([]byte("b"), 30)}, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d1 := xxhash.New()
			for _, chunk := range tt.chunks {
				n, err := d1.Write(chunk)
				if err != nil {
					t.Fatalf("Write error: %v", err)
				}
				if n != len(chunk) {
					t.Errorf("Write returned %d, want %d", n, len(chunk))
				}
			}
			h1 := d1.Sum64()

			d2 := xxhash.New()
			d2.Write(tt.combined)
			h2 := d2.Sum64()

			if h1 != h2 {
				t.Errorf("chunked %x != combined %x", h1, h2)
			}
		})
	}
}

func TestDigest_Size(t *testing.T) {
	d := xxhash.New()
	if d.Size() != 8 {
		t.Errorf("Size() = %d, want 8", d.Size())
	}
}

func TestDigest_BlockSize(t *testing.T) {
	d := xxhash.New()
	if d.BlockSize() != 32 {
		t.Errorf("BlockSize() = %d, want 32", d.BlockSize())
	}
}

func TestDigest_Reset(t *testing.T) {
	data := []byte("test data for reset")
	d := xxhash.New()

	d.Write(data)
	h1 := d.Sum64()

	d.Reset()
	d.Write(data)
	h2 := d.Sum64()

	if h1 != h2 {
		t.Errorf("after Reset: %x != %x", h1, h2)
	}
}

func TestDigest_Sum(t *testing.T) {
	d := xxhash.New()
	d.Write([]byte("hello"))
	sum := d.Sum(nil)

	if len(sum) != 8 {
		t.Errorf("Sum returned %d bytes, want 8", len(sum))
	}

	digestVal := binary.BigEndian.Uint64(sum)
	if digestVal != d.Sum64() {
		t.Errorf("Sum bytes != Sum64: %x vs %x", digestVal, d.Sum64())
	}
}

func TestDigest_Sum_Append(t *testing.T) {
	d := xxhash.New()
	d.Write([]byte("hello"))
	prefix := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	result := d.Sum(prefix)

	if len(result) != len(prefix)+8 {
		t.Errorf("Sum with prefix returned %d bytes, want %d", len(result), len(prefix)+8)
	}
	if !bytes.Equal(result[:4], prefix) {
		t.Errorf("prefix not preserved: %x != %x", result[:4], prefix)
	}
}

func TestDigest_MarshalBinary(t *testing.T) {
	data := []byte("test data for marshal")
	d := xxhash.New()
	d.Write(data)
	h1 := d.Sum64()

	b, err := d.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary error: %v", err)
	}

	d2 := xxhash.New()
	err = d2.UnmarshalBinary(b)
	if err != nil {
		t.Fatalf("UnmarshalBinary error: %v", err)
	}

	h2 := d2.Sum64()
	if h1 != h2 {
		t.Errorf("after marshal/unmarshal: %x != %x", h1, h2)
	}
}

func TestDigest_MarshalBinary_AddMore(t *testing.T) {
	data := []byte("first chunk of data")
	more := []byte(" and more")

	d := xxhash.New()
	d.Write(data)
	b, _ := d.MarshalBinary()

	d2 := xxhash.New()
	d2.UnmarshalBinary(b)
	d2.Write(more)

	d.Write(more)

	if d.Sum64() != d2.Sum64() {
		t.Errorf("marshaled+added != original+added: %x != %x", d.Sum64(), d2.Sum64())
	}
}

func TestDigest_UnmarshalBinary_Invalid(t *testing.T) {
	invalidTests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too_short", []byte("xxh\x06")},
		{"wrong_magic", bytes.Repeat([]byte{0xFF}, 62)},
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			d := xxhash.New()
			err := d.UnmarshalBinary(tt.data)
			if err == nil {
				t.Error("expected error for invalid data")
			}
		})
	}
}

func TestDigest_HashInterface(t *testing.T) {
	var h hash.Hash = xxhash.New()
	h.Write([]byte("interface test"))
	_ = h.Sum(nil)
	h.Reset()
	h.Write([]byte("interface test after reset"))
	_ = h.Sum(nil)
}

func TestDigest_Hash64Interface(t *testing.T) {
	var h hash.Hash64 = xxhash.New()
	data := make([]byte, 1024)
	rand.Read(data)

	h.Write(data)
	sum := h.Sum64()

	h.Reset()
	h.Write(data)
	sum2 := h.Sum64()

	if sum != sum2 {
		t.Errorf("Reset produced different hash: %x != %x", sum, sum2)
	}
}

func TestSum64_VaryingLengths(t *testing.T) {
	for length := 0; length <= 256; length++ {
		data := make([]byte, length)
		rand.Read(data)
		h := xxhash.Sum64(data)
		_ = h
	}
}

func TestSum64_ZeroValue(t *testing.T) {
	// xxhash should handle all-zero data
	for _, size := range []int{0, 1, 4, 8, 16, 32, 64, 128} {
		data := make([]byte, size)
		h := xxhash.Sum64(data)
		// Just verify it doesn't panic and produces consistent results
		if xxhash.Sum64(data) != h {
			t.Errorf("inconsistent hash for zero data of size %d", size)
		}
	}
}

func TestDigest_Write_Large(t *testing.T) {
	data := make([]byte, 100000)
	rand.Read(data)

	d := xxhash.New()
	n, err := d.Write(data)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}

	sum := d.Sum64()
	if sum == 0 {
		t.Error("hash of random data should not be zero")
	}
}

func TestDigest_MultipleMarshal(t *testing.T) {
	data := bytes.Repeat([]byte("ABCD"), 100)
	d := xxhash.New()

	for i := 0; i < 5; i++ {
		chunk := data[i*10 : (i+1)*10]
		d.Write(chunk)
		b, _ := d.MarshalBinary()
		d2 := xxhash.New()
		d2.UnmarshalBinary(b)
		if d.Sum64() != d2.Sum64() {
			t.Errorf("marshal inconsistency at step %d", i)
		}
	}
}

func TestWriteString(t *testing.T) {
	s := "test write string"
	d := xxhash.New()
	n, err := d.WriteString(s)
	if err != nil {
		t.Fatalf("WriteString error: %v", err)
	}
	if n != len(s) {
		t.Errorf("WriteString returned %d, want %d", n, len(s))
	}
}

func BenchmarkSum64_64(b *testing.B) {
	data := []byte("0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF")
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xxhash.Sum64(data)
	}
}

func BenchmarkSum64_4K(b *testing.B) {
	data := make([]byte, 4096)
	rand.Read(data)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xxhash.Sum64(data)
	}
}

func FuzzSum64(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, data []byte) {
		h := xxhash.Sum64(data)
		if xxhash.Sum64(data) != h {
			t.Fatal("inconsistent hash")
		}
	})
}

func FuzzDigest(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3})
	f.Fuzz(func(t *testing.T, data []byte) {
		d := xxhash.New()
		d.Write(data)
		h1 := d.Sum64()
		h2 := xxhash.Sum64(data)
		if h1 != h2 {
			t.Fatalf("Digest mismatch: %x != %x", h1, h2)
		}
	})
}
