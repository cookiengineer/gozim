package tracers

import (
	"encoding/hex"
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
)

func TraceChecksum(a *zim.Archive, r zim.Reader) {

	fmt.Println()
	fmt.Println("> Section: Checksum")
	fmt.Println()

	h := a.Header()

	if !h.HasChecksum() {
		fmt.Println("  (no checksum in this archive)")
		fmt.Println()
		return
	}

	stored := readRaw(r, int64(h.ChecksumPos), 16)
	fmt.Printf("  location: 0x%016X\n", h.ChecksumPos)
	fmt.Printf("  raw bytes: %s\n", renderHex(stored))

	computed, err := zim.ComputeChecksum(r, h.ChecksumPos)
	if err != nil {
		fmt.Printf("  compute error: %v\n", err)
	} else if computed != nil {
		fmt.Printf("  stored:      %s\n", hex.EncodeToString(stored))
		fmt.Printf("  computed:    %s\n", hex.EncodeToString(computed))
		if hex.EncodeToString(stored) == hex.EncodeToString(computed) {
			fmt.Println("  match:       ✓ (OK)")
		} else {
			fmt.Println("  match:       ✗ (MISMATCH)")
		}
	}
	fmt.Println()

}

