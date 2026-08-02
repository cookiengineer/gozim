package tracers

import (
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
)

func describeCompression(c zim.Compression) string {
	switch c {
	case zim.CompressionNone:
		return "none"
	case zim.CompressionZstd:
		return "zstd"
	case zim.CompressionLzma:
		return "lzma/xz"
	default:
		return fmt.Sprintf("unknown(%d)", c)
	}
}

