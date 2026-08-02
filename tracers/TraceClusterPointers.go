package tracers

import (
	"encoding/binary"
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
)

func TraceClusterPointers(a *zim.Archive, r zim.Reader) {

	fmt.Println()
	fmt.Println("> Section: Cluster Pointer List")
	fmt.Println()

	h := a.Header()
	count := int(h.ClusterCount)
	ptrData := readRaw(r, int64(h.ClusterPtrPos), count*8)

	fmt.Printf("  location: 0x%016X, %d entries (%d bytes)\n", h.ClusterPtrPos, count, count*8)
	fmt.Printf("  first 64 bytes: %s\n", renderHex(truncateBytes(ptrData, 64)))
	fmt.Println()

	var clusterOffsets []uint64
	for i := 0; i < count; i++ {
		off := binary.LittleEndian.Uint64(ptrData[i*8:])
		clusterOffsets = append(clusterOffsets, off)
	}

	for i, off := range clusterOffsets {
		endOff := uint64(r.Size())
		if i+1 < len(clusterOffsets) {
			endOff = clusterOffsets[i+1]
		}
		if h.HasChecksum() && i+1 >= len(clusterOffsets) {
			endOff = h.ChecksumPos
		}
		size := endOff - off
		fmt.Printf("  [%3d] offset 0x%X (%d), size %d bytes (ends at 0x%X)\n", i, off, off, size, endOff)
	}
	fmt.Println()
}

