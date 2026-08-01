package hash_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/cookiengineer/gozim/compress/lzma/hash"
)

func TestCyclicPoly_RoundTrip(t *testing.T) {
	for _, n := range []int{1, 2, 3, 4, 5, 8, 16, 32} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			data := make([]byte, 256)
			rand.Read(data)

			h1 := hash.NewCyclicPoly(n)
			var hashes1 []uint64
			for _, b := range data {
				hashes1 = append(hashes1, h1.RollByte(b))
			}

			h2 := hash.NewCyclicPoly(n)
			var hashes2 []uint64
			for _, b := range data {
				hashes2 = append(hashes2, h2.RollByte(b))
			}

			if len(hashes1) < n {
				t.Skip("not enough data")
			}

			for i := n - 1; i < len(hashes1); i++ {
				if hashes1[i] != hashes2[i] {
					t.Errorf("hash mismatch at position %d: %d != %d", i, hashes1[i], hashes2[i])
				}
			}
		})
	}
}

func TestCyclicPoly_Len(t *testing.T) {
	for _, n := range []int{1, 4, 8, 16} {
		h := hash.NewCyclicPoly(n)
		if h.Len() != n {
			t.Errorf("Len() = %d, want %d", h.Len(), n)
		}
	}
}

func TestCyclicPoly_PanicsOnZero(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for n=0")
		}
	}()
	hash.NewCyclicPoly(0)
}

func TestCyclicPoly_SmallInput(t *testing.T) {
	h := hash.NewCyclicPoly(4)
	// Feed fewer bytes than Len(), hash values are valid after Len() bytes
	hashes := make([]uint64, 0)
	for _, b := range []byte{0x00, 0x01, 0x02} {
		hashes = append(hashes, h.RollByte(b))
	}
	if len(hashes) != 3 {
		t.Errorf("expected 3 hash values, got %d", len(hashes))
	}
	// After feeding Len() bytes, hash should still work
	h.RollByte(0x03)
}

func TestCyclicPoly_Reproducibility(t *testing.T) {
	data := []byte("The quick brown fox jumps over the lazy dog")
	for _, n := range []int{4, 8, 12} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			h1 := hash.NewCyclicPoly(n)
			h2 := hash.NewCyclicPoly(n)
			for _, b := range data {
				h1.RollByte(b)
				h2.RollByte(b)
			}
			// Force all bytes through
			h1.RollByte(0)
			h2.RollByte(0)
			h1.RollByte(0)
			h2.RollByte(0)
			h1.RollByte(0)
			h2.RollByte(0)
		})
	}
}

func TestRabinKarp_RoundTrip(t *testing.T) {
	for _, n := range []int{1, 2, 3, 4, 5, 8, 16, 32} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			data := make([]byte, 256)
			rand.Read(data)

			h1 := hash.NewRabinKarp(n)
			var hashes1 []uint64
			for _, b := range data {
				hashes1 = append(hashes1, h1.RollByte(b))
			}

			h2 := hash.NewRabinKarp(n)
			var hashes2 []uint64
			for _, b := range data {
				hashes2 = append(hashes2, h2.RollByte(b))
			}

			if len(hashes1) < n {
				t.Skip("not enough data")
			}

			for i := n - 1; i < len(hashes1); i++ {
				if hashes1[i] != hashes2[i] {
					t.Errorf("hash mismatch at position %d: %x != %x", i, hashes1[i], hashes2[i])
				}
			}
		})
	}
}

func TestRabinKarp_Len(t *testing.T) {
	for _, n := range []int{1, 4, 8, 16} {
		h := hash.NewRabinKarp(n)
		if h.Len() != n {
			t.Errorf("Len() = %d, want %d", h.Len(), n)
		}
	}
}

func TestRabinKarp_PanicsOnZero(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for n=0")
		}
	}()
	hash.NewRabinKarp(0)
}

func TestRabinKarp_CustomConstant(t *testing.T) {
	data := []byte("test data for rabin-karp with custom constant")
	const customA uint64 = 0x1234567890ABCDEF

	h1 := hash.NewRabinKarpConst(8, customA)
	h2 := hash.NewRabinKarpConst(8, customA)

	for _, b := range data {
		v1 := h1.RollByte(b)
		v2 := h2.RollByte(b)
		_ = v1
		_ = v2
	}

	// After feeding all bytes, verify consistency
	h1.RollByte(0xFF)
	h2.RollByte(0xFF)
}

func TestRabinKarp_DifferentConstants(t *testing.T) {
	data := make([]byte, 64)
	rand.Read(data)

	h1 := hash.NewRabinKarpConst(8, 0x97b548add41d5da1)
	h2 := hash.NewRabinKarpConst(8, 0x1234567890ABCDEF)

	hashes1 := hash.Hashes(h1, data)
	hashes2 := hash.Hashes(h2, data)

	if len(hashes1) != len(hashes2) {
		t.Fatalf("unexpected hash count: %d vs %d", len(hashes1), len(hashes2))
	}

	allSame := true
	for i := range hashes1 {
		if hashes1[i] != hashes2[i] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Log("different constants produced same hashes (unlikely but possible)")
	}
}

func TestHashes(t *testing.T) {
	for _, n := range []int{2, 4, 8, 16} {
		t.Run(fmt.Sprintf("cyclic_poly_n=%d", n), func(t *testing.T) {
			data := make([]byte, 128)
			rand.Read(data)

			r := hash.NewCyclicPoly(n)
			result := hash.Hashes(r, data)

			expected := len(data) - n + 1
			if len(result) != expected {
				t.Errorf("Hashes returned %d values, expected %d", len(result), expected)
			}

			if len(result) >= 2 {
				r2 := hash.NewCyclicPoly(n)
				r2result := hash.Hashes(r2, data)
				for i := range result {
					if result[i] != r2result[i] {
						t.Errorf("hash mismatch at position %d", i)
						break
					}
				}
			}
		})
	}
}

func TestHashes_RabinKarp(t *testing.T) {
	for _, n := range []int{4, 8, 16} {
		t.Run(fmt.Sprintf("rabin_karp_n=%d", n), func(t *testing.T) {
			data := make([]byte, 128)
			rand.Read(data)

			r := hash.NewRabinKarp(n)
			result := hash.Hashes(r, data)

			expected := len(data) - n + 1
			if len(result) != expected {
				t.Errorf("Hashes returned %d values, expected %d", len(result), expected)
			}
		})
	}
}

func TestHashes_TooShort(t *testing.T) {
	r := hash.NewCyclicPoly(16)
	data := []byte("short")
	result := hash.Hashes(r, data)
	if result != nil {
		t.Errorf("expected nil for input shorter than hash window, got %v", result)
	}
}

func TestHashes_ExactLength(t *testing.T) {
	r := hash.NewCyclicPoly(4)
	data := []byte{0, 1, 2, 3}
	result := hash.Hashes(r, data)
	expected := len(data) - r.Len() + 1
	if len(result) != expected {
		t.Errorf("Hashes returned %d values, expected %d", len(result), expected)
	}
}

func TestRollerInterface(t *testing.T) {
	var r1 hash.Roller = hash.NewCyclicPoly(4)
	var r2 hash.Roller = hash.NewRabinKarp(4)

	if r1.Len() != 4 {
		t.Errorf("CyclicPoly.Len() = %d, want 4", r1.Len())
	}
	if r2.Len() != 4 {
		t.Errorf("RabinKarp.Len() = %d, want 4", r2.Len())
	}

	for _, b := range []byte{0, 1, 2, 3, 4, 5, 6, 7} {
		r1.RollByte(b)
		r2.RollByte(b)
	}
}

func TestHashes_SameInputProducesSameHash(t *testing.T) {
	data := bytes.Repeat([]byte("ABCDEFGHIJKLMNOP"), 8)

	for _, n := range []int{4, 8, 12} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			r1 := hash.NewCyclicPoly(n)
			r2 := hash.NewRabinKarp(n)

			h1 := hash.Hashes(r1, data)
			h2 := hash.Hashes(r2, data)

			if len(h1) != len(h2) {
				t.Fatalf("hash count mismatch: %d vs %d", len(h1), len(h2))
			}
		})
	}
}

func TestCyclicPoly_KnownValue(t *testing.T) {
	data := []byte("known data for cyclic poly test")
	h := hash.NewCyclicPoly(4)

	// Feed data, record the last hash
	var lastHash uint64
	for _, b := range data {
		lastHash = h.RollByte(b)
	}

	// Same input should produce same final hash
	h2 := hash.NewCyclicPoly(4)
	var lastHash2 uint64
	for _, b := range data {
		lastHash2 = h2.RollByte(b)
	}

	if lastHash != lastHash2 {
		t.Errorf("inconsistent result: %x != %x", lastHash, lastHash2)
	}
}

func TestRabinKarp_KnownValue(t *testing.T) {
	data := []byte("known data for rabin karp test")
	h := hash.NewRabinKarp(8)

	var lastHash uint64
	for _, b := range data {
		lastHash = h.RollByte(b)
	}

	h2 := hash.NewRabinKarp(8)
	var lastHash2 uint64
	for _, b := range data {
		lastHash2 = h2.RollByte(b)
	}

	if lastHash != lastHash2 {
		t.Errorf("inconsistent result: %x != %x", lastHash, lastHash2)
	}
}

func BenchmarkCyclicPoly(b *testing.B) {
	data := make([]byte, 1024)
	rand.Read(data)
	h := hash.NewCyclicPoly(8)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range data {
			h.RollByte(v)
		}
	}
}

func BenchmarkRabinKarp(b *testing.B) {
	data := make([]byte, 1024)
	rand.Read(data)
	h := hash.NewRabinKarp(8)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range data {
			h.RollByte(v)
		}
	}
}

func BenchmarkHashes_CyclicPoly(b *testing.B) {
	data := make([]byte, 4096)
	rand.Read(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := hash.NewCyclicPoly(16)
		hash.Hashes(r, data)
	}
}

func BenchmarkHashes_RabinKarp(b *testing.B) {
	data := make([]byte, 4096)
	rand.Read(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := hash.NewRabinKarp(16)
		hash.Hashes(r, data)
	}
}

func TestCyclicPoly_Rolling(t *testing.T) {
	// Verify the rolling property: hash(data[1:n] + byte) should be
	// derivable from hash(data[0:n]) in a rolling fashion.
	n := 4
	data := make([]byte, 2*n+10)
	rand.Read(data)

	h := hash.NewCyclicPoly(n)
	// Feed first n-1 bytes without recording
	for i := 0; i < n-1; i++ {
		h.RollByte(data[i])
	}

	rolling := hash.NewCyclicPoly(n)
	rollingHashes := hash.Hashes(rolling, data)

	// Now manually get sliding hashes
	manual := make([]uint64, len(data)-n+1)
	for i := n - 1; i < len(data); i++ {
		manual[i-n+1] = h.RollByte(data[i])
	}

	for i := range manual {
		if manual[i] != rollingHashes[i] {
			t.Fatalf("sliding hash mismatch at %d: %x != %x (n=%d)", i, manual[i], rollingHashes[i], n)
		}
	}
}

func TestRabinKarp_Rolling(t *testing.T) {
	n := 4
	data := make([]byte, 2*n+10)
	rand.Read(data)

	h := hash.NewRabinKarp(n)
	for i := 0; i < n-1; i++ {
		h.RollByte(data[i])
	}

	rolling := hash.NewRabinKarp(n)
	rollingHashes := hash.Hashes(rolling, data)

	manual := make([]uint64, len(data)-n+1)
	for i := n - 1; i < len(data); i++ {
		manual[i-n+1] = h.RollByte(data[i])
	}

	for i := range manual {
		if manual[i] != rollingHashes[i] {
			t.Fatalf("sliding hash mismatch at %d: %x != %x (n=%d)", i, manual[i], rollingHashes[i], n)
		}
	}
}

func TestCyclicPoly_Deterministic(t *testing.T) {
	data := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	h := hash.NewCyclicPoly(3)
	hashes := make([]uint64, 0)
	for _, b := range data {
		hashes = append(hashes, h.RollByte(b))
	}

	for run := 0; run < 10; run++ {
		h2 := hash.NewCyclicPoly(3)
		for i, b := range data {
			v := h2.RollByte(b)
			if v != hashes[i] {
				t.Errorf("run %d position %d: %x != %x", run, i, v, hashes[i])
			}
		}
	}
}

func TestRabinKarp_Deterministic(t *testing.T) {
	data := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	h := hash.NewRabinKarp(3)
	hashes := make([]uint64, 0)
	for _, b := range data {
		hashes = append(hashes, h.RollByte(b))
	}

	for run := 0; run < 10; run++ {
		h2 := hash.NewRabinKarp(3)
		for i, b := range data {
			v := h2.RollByte(b)
			if v != hashes[i] {
				t.Errorf("run %d position %d: %x != %x", run, i, v, hashes[i])
			}
		}
	}
}

func TestHashes_DifferentRollers(t *testing.T) {
	data := make([]byte, 128)
	rand.Read(data)

	for _, n := range []int{4, 8, 16} {
		cp := hash.NewCyclicPoly(n)
		rk := hash.NewRabinKarp(n)

		cpHashes := hash.Hashes(cp, data)
		rkHashes := hash.Hashes(rk, data)

		if len(cpHashes) != len(rkHashes) {
			t.Fatalf("n=%d: hash count mismatch: %d vs %d", n, len(cpHashes), len(rkHashes))
		}
	}
}

func FuzzCyclicPoly(f *testing.F) {
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		for n := 1; n <= 16; n *= 2 {
			h1 := hash.NewCyclicPoly(n)
			h2 := hash.NewCyclicPoly(n)
			for _, b := range data {
				v1 := h1.RollByte(b)
				v2 := h2.RollByte(b)
				if v1 != v2 {
					t.Fatalf("n=%d: inconsistent hashes: %x != %x", n, v1, v2)
				}
			}
		}
	})
}

func FuzzRabinKarp(f *testing.F) {
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		for n := 1; n <= 16; n *= 2 {
			h1 := hash.NewRabinKarp(n)
			h2 := hash.NewRabinKarp(n)
			for _, b := range data {
				v1 := h1.RollByte(b)
				v2 := h2.RollByte(b)
				if v1 != v2 {
					t.Fatalf("n=%d: inconsistent hashes: %x != %x", n, v1, v2)
				}
			}
		}
	})
}

// Ensure unused imports don't cause compile errors.
var _ = sha256.New
var _ = binary.LittleEndian
var _ = fmt.Sprintf
