package tracers

import (
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
)

func TraceHeader(a *zim.Archive, r zim.Reader) {

	fmt.Println()
	fmt.Println("> Section: Header (80 bytes at offset 0)")
	fmt.Println()

	headerData := readRaw(r, 0, 80)
	fmt.Printf("  Raw bytes: %s\n", renderHex(headerData))
	fmt.Println()

	h := a.Header()
	fmt.Printf("  magicNumber:      0x%08X (%d)\n", h.MagicNumber, h.MagicNumber)
	fmt.Printf("  majorVersion:     %d\n", h.MajorVersion)
	fmt.Printf("  minorVersion:     %d\n", h.MinorVersion)
	fmt.Printf("  uuid:             %s\n", h.Uuid)
	fmt.Printf("  entryCount:       %d\n", h.EntryCount)
	fmt.Printf("  clusterCount:     %d\n", h.ClusterCount)
	fmt.Printf("  urlPtrPos:        0x%016X (%d)\n", h.UrlPtrPos, h.UrlPtrPos)
	fmt.Printf("  titleIdxPos:      0x%016X (%d)", h.TitleIdxPos, h.TitleIdxPos)
	if h.TitleIdxPos == 0xFFFFFFFFFFFFFFFF {
		fmt.Print(" = none")
	}
	fmt.Println()
	fmt.Printf("  clusterPtrPos:    0x%016X (%d)\n", h.ClusterPtrPos, h.ClusterPtrPos)
	fmt.Printf("  mimeListPos:      0x%016X (%d)\n", h.MimeListPos, h.MimeListPos)
	fmt.Printf("  mainPage:         %d", h.MainPage)
	if h.MainPage == 0xFFFFFFFF {
		fmt.Print(" = none")
	}
	fmt.Println()
	fmt.Printf("  layoutPage:       %d", h.LayoutPage)
	if h.LayoutPage == 0xFFFFFFFF {
		fmt.Print(" = none")
	}
	fmt.Println()
	fmt.Printf("  checksumPos:      0x%016X (%d)\n", h.ChecksumPos, h.ChecksumPos)
	fmt.Println()
	fmt.Printf("  hasChecksum:       %v\n", h.HasChecksum())
	fmt.Printf("  newNamespaceScheme: %v\n", h.HasNewNamespaceScheme())
	fmt.Printf("  hasTitleIndex:     %v\n", h.HasTitleIndex())
	fmt.Printf("  hasMainPage:       %v\n", h.HasMainPage())
	fmt.Println()

}


