package le

import "encoding/binary"

type Indexer interface {
	~int | ~int32 | ~uint | ~uint32
}

func Load32[I Indexer](b []byte, i I) uint32 {
	return binary.LittleEndian.Uint32(b[i:])
}

func Load64[I Indexer](b []byte, i I) uint64 {
	return binary.LittleEndian.Uint64(b[i:])
}
