package actions

import (
	"fmt"
	"os"
	"github.com/cookiengineer/gozim/archive/zim"
)

func Info(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: gozim info <archive.zim>")
		os.Exit(1)
	}

	filename := args[0]
	archive, err := zim.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening archive: %v\n", err)
		os.Exit(1)
	}
	defer archive.Close()

	fmt.Printf("Archive:        %s\n", filename)
	fmt.Printf("UUID:           %s\n", archive.Uuid())
	fmt.Printf("Version:        %d.%d\n", zim.MajorVersion, zim.MinorVersion)
	fmt.Printf("Entries:        %d\n", archive.EntryCount())
	fmt.Printf("Articles:       %d\n", archive.ArticleCount())
	fmt.Printf("Media:          %d\n", archive.MediaCount())
	fmt.Printf("Clusters:       %d\n", archive.ClusterCount())
	fmt.Printf("Checksum:       %s\n", archive.Checksum())
	fmt.Printf("Checksum OK:    %v\n", archive.HasChecksum())
	fmt.Printf("Fulltext Index: %v\n", archive.HasFulltextIndex())
	fmt.Printf("Title Index:    %v\n", archive.HasTitleIndex())
	fmt.Printf("New Namespace:  %v\n", archive.HasNewNamespaceScheme())
	fmt.Printf("Multipart:      %v\n", archive.IsMultiPart())

	if title, ok := archive.Metadata("Title"); ok {
		fmt.Printf("Title:          %s\n", title)
	}
	if lang, ok := archive.Metadata("Language"); ok {
		fmt.Printf("Language:       %s\n", lang)
	}
	if creator, ok := archive.Metadata("Creator"); ok {
		fmt.Printf("Creator:        %s\n", creator)
	}
}

