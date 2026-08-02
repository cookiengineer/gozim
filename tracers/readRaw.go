package tracers

import (
	"github.com/cookiengineer/gozim/archive/zim"
)

func readRaw(r zim.Reader, offset int64, size int) []byte {
	if size <= 0 || offset < 0 {
		return nil
	}
	buf := make([]byte, size)
	n, _ := r.ReadAt(buf, offset)
	return buf[:n]
}

