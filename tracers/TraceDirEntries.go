package tracers

import (
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
)

func TraceDirEntries(a *zim.Archive, r zim.Reader) {

	fmt.Println()
	fmt.Println("> Section: Directory Entries")
	fmt.Println()

	dirents := a.Dirents()
	urlPtrs := a.UrlPtrOffsets()
	ml := a.MimeTypeList()

	fmt.Printf("  %d entries\n\n", len(dirents))

	maxShow := 40
	if len(dirents) > maxShow {
		fmt.Printf("  (showing first %d of %d entries)\n\n", maxShow, len(dirents))
	}

	for i := 0; i < len(dirents) && i < maxShow; i++ {
		d := dirents[i]
		direntOff := urlPtrs[i]

		rawBytes := readRaw(r, int64(direntOff), 512)
		_, direntSize, _ := zim.ParseDirent(rawBytes)
		direntData := rawBytes[:min(direntSize, len(rawBytes))]

		fmt.Printf("  [%3d] offset 0x%X (%d), %d raw bytes\n", i, direntOff, direntOff, direntSize)
		fmt.Printf("        namespace:    %c (%s)\n", d.Namespace, describeNamespace(d.Namespace))
		fmt.Printf("        path:         %q\n", d.Path)
		fmt.Printf("        title:        %q\n", truncateString(d.Title, 60))
		fmt.Printf("        mime index:   %d", d.MimeTypeIndex)
		if mime := ml.MimeType(d.MimeTypeIndex); mime != "" {
			fmt.Printf(" (%s)", mime)
		}
		fmt.Println()

		switch {
		case d.IsRedirect():
			fmt.Printf("        type:         redirect → entry %d\n", d.RedirectIndex)
		case d.IsLinkTarget():
			fmt.Printf("        type:         linktarget (cluster %d, blob %d)\n", d.ClusterNumber, d.BlobNumber)
		case d.IsDeleted():
			fmt.Printf("        type:         deleted\n")
		default:
			fmt.Printf("        type:         content (cluster %d, blob %d)\n", d.ClusterNumber, d.BlobNumber)
		}

		if len(d.Extra) > 0 {
			fmt.Printf("        extra params: %d bytes: %s\n", len(d.Extra), renderHex(d.Extra))
		}

		hexStr := renderHex(truncateBytes(direntData, 48))
		fmt.Printf("        raw hex:      %s\n", hexStr)
		fmt.Println()
	}

	if len(dirents) > maxShow {
		fmt.Printf("  ... (%d more entries)\n\n", len(dirents)-maxShow)
	}

}

