package tracers

import (
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
)

func TraceUrlPointers(a *zim.Archive, r zim.Reader) {

	fmt.Println()
	fmt.Println("> Section: URL Pointer List")
	fmt.Println()

	offsets := a.UrlPtrOffsets()
	h := a.Header()
	count := len(offsets)

	ptrData := readRaw(r, int64(h.UrlPtrPos), count*8)
	fmt.Printf("  location: 0x%016X, %d entries (%d bytes)\n", h.UrlPtrPos, count, count*8)
	fmt.Printf("  first 64 bytes: %s\n", renderHex(truncateBytes(ptrData, 64)))
	fmt.Println()

	maxShow := 30
	if count > maxShow {
		fmt.Printf("  (showing first %d of %d entries)\n", maxShow, count)
	}
	for i := 0; i < count && i < maxShow; i++ {
		off := offsets[i]
		fmt.Printf("  [%3d] → dirent at 0x%016X (%d)\n", i, off, off)
	}
	if count > maxShow {
		fmt.Printf("  ... (%d more entries)\n", count-maxShow)
	}
	fmt.Println()
}

