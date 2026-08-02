package tracers

import (
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
)

func TraceTitleIndex(a *zim.Archive, r zim.Reader) {

	fmt.Println()
	fmt.Println("> Section: Title Index")
	fmt.Println()

	h := a.Header()
	titleIdx := a.TitleIndices()

	if titleIdx == nil {
		fmt.Println("  (no title index loaded)")
		fmt.Println()
		return
	}

	if h.HasTitleIndex() && h.TitleIdxPos != 0xFFFFFFFFFFFFFFFF {
		size := len(titleIdx) * 4
		rawBytes := readRaw(r, int64(h.TitleIdxPos), size)
		fmt.Printf("  v0 location: 0x%016X, %d entries (%d bytes)\n", h.TitleIdxPos, len(titleIdx), size)
		fmt.Printf("  first 64 bytes: %s\n", renderHex(truncateBytes(rawBytes, 64)))
	}

	fmt.Printf("  title entries: %d\n", len(titleIdx))
	maxShow := 20
	if len(titleIdx) > maxShow {
		fmt.Printf("  (showing first %d of %d)\n", maxShow, len(titleIdx))
	}

	dirents := a.Dirents()
	for i := 0; i < len(titleIdx) && i < maxShow; i++ {
		entryIdx := titleIdx[i]
		if int(entryIdx) < len(dirents) {
			d := dirents[entryIdx]
			fmt.Printf("  [%3d] entry %d: %q (title: %q)\n", i, entryIdx, d.Path, truncateString(d.Title, 50))
		} else {
			fmt.Printf("  [%3d] entry %d: (out of range)\n", i, entryIdx)
		}
	}
	if len(titleIdx) > maxShow {
		fmt.Printf("  ... (%d more)\n", len(titleIdx)-maxShow)
	}
	fmt.Println()
}


