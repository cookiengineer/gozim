package zim

import (
	"bytes"
	"testing"
)

func TestParseClusterInfo(t *testing.T) {
	tests := []struct {
		byteVal     byte
		compression Compression
		extended    bool
	}{
		{0x01, CompressionNone, false},
		{0x05, CompressionZstd, false},
		{0x04, CompressionLzma, false},
		{0x11, CompressionNone, true},
		{0x15, CompressionZstd, true},
	}

	for _, tt := range tests {
		info := ParseClusterInfo(tt.byteVal)
		if info.Compression != tt.compression {
			t.Errorf("ParseClusterInfo(0x%02x).Compression = %d, want %d", tt.byteVal, info.Compression, tt.compression)
		}
		if info.IsExtended != tt.extended {
			t.Errorf("ParseClusterInfo(0x%02x).IsExtended = %v, want %v", tt.byteVal, info.IsExtended, tt.extended)
		}
	}
}

func TestEncodeClusterInfo(t *testing.T) {
	tests := []struct {
		info    ClusterInfo
		byteVal byte
	}{
		{ClusterInfo{CompressionNone, false}, 0x01},
		{ClusterInfo{CompressionZstd, false}, 0x05},
		{ClusterInfo{CompressionZstd, true}, 0x15},
	}

	for _, tt := range tests {
		encoded := EncodeClusterInfo(tt.info)
		if encoded != tt.byteVal {
			t.Errorf("EncodeClusterInfo(%+v) = 0x%02x, want 0x%02x", tt.info, encoded, tt.byteVal)
		}
	}
}

func TestParseClusterUncompressed(t *testing.T) {
	factory := NewCompressorFactory()
	noneComp, _ := factory.CompressorForType(CompressionNone)

	// Create a cluster with two uncompressed blobs.
	blob1 := []byte("hello")
	blob2 := []byte(" world")

	clusterBytes, err := EncodeCluster(CompressionNone, [][]byte{blob1, blob2}, noneComp)
	if err != nil {
		t.Fatalf("EncodeCluster() error: %v", err)
	}

	cluster, err := ParseCluster(clusterBytes, factory)
	if err != nil {
		t.Fatalf("ParseCluster() error: %v", err)
	}

	if cluster.BlobCount() != 2 {
		t.Errorf("BlobCount() = %d, want 2", cluster.BlobCount())
	}

	if cluster.Compression() != CompressionNone {
		t.Errorf("Compression() = %d, want %d", cluster.Compression(), CompressionNone)
	}

	// Read first blob.
	data, err := cluster.Blob(0)
	if err != nil {
		t.Fatalf("Blob(0) error: %v", err)
	}
	if !bytes.Equal(data, blob1) {
		t.Errorf("Blob(0) = %q, want %q", data, blob1)
	}

	// Read second blob.
	data, err = cluster.Blob(1)
	if err != nil {
		t.Fatalf("Blob(1) error: %v", err)
	}
	if !bytes.Equal(data, blob2) {
		t.Errorf("Blob(1) = %q, want %q", data, blob2)
	}

	// Out of range blob.
	_, err = cluster.Blob(5)
	if err == nil {
		t.Error("expected error for out-of-range blob")
	}
}

func TestParseClusterZstd(t *testing.T) {
	factory := NewCompressorFactory()
	zstdComp, _ := factory.CompressorForType(CompressionZstd)

	// Generate some compressible text.
	blob1 := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 100)
	blob2 := []byte("short text")

	clusterBytes, err := EncodeCluster(CompressionZstd, [][]byte{blob1, blob2}, zstdComp)
	if err != nil {
		t.Fatalf("EncodeCluster() error: %v", err)
	}

	cluster, err := ParseCluster(clusterBytes, factory)
	if err != nil {
		t.Fatalf("ParseCluster() error: %v", err)
	}

	if cluster.BlobCount() != 2 {
		t.Errorf("BlobCount() = %d, want 2", cluster.BlobCount())
	}

	if cluster.Compression() != CompressionZstd {
		t.Errorf("Compression() = %d, want %d", cluster.Compression(), CompressionZstd)
	}

	// Verify first blob decompresses correctly.
	data, err := cluster.Blob(0)
	if err != nil {
		t.Fatalf("Blob(0) error: %v", err)
	}
	if !bytes.Equal(data, blob1) {
		t.Errorf("Blob(0) length = %d, want %d", len(data), len(blob1))
	}

	// Verify second blob.
	data, err = cluster.Blob(1)
	if err != nil {
		t.Fatalf("Blob(1) error: %v", err)
	}
	if !bytes.Equal(data, blob2) {
		t.Errorf("Blob(1) = %q, want %q", data, blob2)
	}
}

func TestClusterBlobRange(t *testing.T) {
	factory := NewCompressorFactory()
	noneComp, _ := factory.CompressorForType(CompressionNone)
	blob := []byte("0123456789abcdef")

	clusterBytes, err := EncodeCluster(CompressionNone, [][]byte{blob}, noneComp)
	if err != nil {
		t.Fatalf("EncodeCluster() error: %v", err)
	}

	cluster, err := ParseCluster(clusterBytes, factory)
	if err != nil {
		t.Fatalf("ParseCluster() error: %v", err)
	}

	// Read range from middle.
	data, err := cluster.BlobRange(0, 5, 5)
	if err != nil {
		t.Fatalf("BlobRange() error: %v", err)
	}
	if !bytes.Equal(data, []byte("56789")) {
		t.Errorf("BlobRange(0, 5, 5) = %q, want \"56789\"", data)
	}

	// Read past end.
	data, err = cluster.BlobRange(0, 14, 10)
	if err != nil {
		t.Fatalf("BlobRange() error: %v", err)
	}
	if !bytes.Equal(data, []byte("ef")) {
		t.Errorf("BlobRange(0, 14, 10) = %q, want \"ef\"", data)
	}
}

func TestParseClusterEmpty(t *testing.T) {
	factory := NewCompressorFactory()

	_, err := ParseCluster(nil, factory)
	if err == nil {
		t.Error("expected error for nil cluster data")
	}

	_, err = ParseCluster([]byte{0x01}, factory)
	if err == nil {
		t.Error("expected error for too-short cluster")
	}
}

func TestClusterRoundTrip(t *testing.T) {
	factory := NewCompressorFactory()

	tests := []struct {
		name        string
		compression Compression
		blobs       [][]byte
	}{
		{
			name:        "none_small",
			compression: CompressionNone,
			blobs:       [][]byte{[]byte("hello"), []byte("world")},
		},
		{
			name:        "zstd_single",
			compression: CompressionZstd,
			blobs:       [][]byte{bytes.Repeat([]byte("data "), 50)},
		},
		{
			name:        "none_many",
			compression: CompressionNone,
			blobs:       [][]byte{[]byte("a"), []byte("bb"), []byte("ccc"), []byte("dddd")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var compressor Compressor
			if tt.compression == CompressionZstd {
				compressor, _ = factory.CompressorForType(CompressionZstd)
			} else {
				compressor, _ = factory.CompressorForType(CompressionNone)
			}

			encoded, err := EncodeCluster(tt.compression, tt.blobs, compressor)
			if err != nil {
				t.Fatalf("EncodeCluster() error: %v", err)
			}

			decoded, err := ParseCluster(encoded, factory)
			if err != nil {
				t.Fatalf("ParseCluster() error: %v", err)
			}

			if decoded.BlobCount() != len(tt.blobs) {
				t.Errorf("BlobCount() = %d, want %d", decoded.BlobCount(), len(tt.blobs))
			}

			for i, expected := range tt.blobs {
				data, err := decoded.Blob(uint32(i))
				if err != nil {
					t.Errorf("Blob(%d) error: %v", i, err)
				}
				if !bytes.Equal(data, expected) {
					t.Errorf("Blob(%d) = %q, want %q", i, data, expected)
				}
			}
		})
	}
}
