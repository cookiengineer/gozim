package tracers

import (
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
)

func TraceMimeList(a *zim.Archive, r zim.Reader) {

	fmt.Println()
	fmt.Println("> Section: MIME List")
	fmt.Println()

	ml := a.MimeTypeList()
	h := a.Header()
	mimeListSize := int(h.UrlPtrPos - h.MimeListPos)
	mimeData := readRaw(r, int64(h.MimeListPos), mimeListSize)

	fmt.Printf("  location: 0x%016X, length: %d bytes\n", h.MimeListPos, mimeListSize)
	fmt.Printf("  first 64 bytes: %s\n", renderHex(truncateBytes(mimeData, 64)))
	fmt.Println()

	count := ml.Len()
	if count == 0 {
		fmt.Println("  (empty)")
		return
	}

	for i := 0; i < count; i++ {
		mime := ml.MimeType(uint16(i))
		if mime == "" {
			fmt.Printf("  [%3d] (null string)\n", i)
		} else {
			fmt.Printf("  [%3d] %q\n", i, mime)
		}
	}
	fmt.Printf("  (%d entries)\n\n", count)
}

