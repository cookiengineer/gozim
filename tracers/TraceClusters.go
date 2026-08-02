package tracers

import (
	"encoding/binary"
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
)

func TraceClusters(a *zim.Archive, r zim.Reader) {

	fmt.Println()
	fmt.Println("> Section: Clusters")
	fmt.Println()

	h := a.Header()
	count := int(h.ClusterCount)

	ptrData := readRaw(r, int64(h.ClusterPtrPos), count*8)
	var clusterOffsets []uint64
	for i := 0; i < count; i++ {
		off := binary.LittleEndian.Uint64(ptrData[i*8:])
		clusterOffsets = append(clusterOffsets, off)
	}

	maxShow := 10
	if count > maxShow {
		fmt.Printf("  (showing first %d of %d clusters)\n\n", maxShow, count)
	}

	for i := 0; i < count && i < maxShow; i++ {
		off := clusterOffsets[i]
		endOff := uint64(r.Size())
		if i+1 < len(clusterOffsets) {
			endOff = clusterOffsets[i+1]
		}
		if h.HasChecksum() && i+1 >= len(clusterOffsets) {
			endOff = h.ChecksumPos
		}

		clusterBytes := readRaw(r, int64(off), int(endOff-off))
		TraceClusterDetail(i, off, clusterBytes)
	}

	if count > maxShow {
		fmt.Printf("  ... (%d more clusters)\n\n", count-maxShow)
	}

}

