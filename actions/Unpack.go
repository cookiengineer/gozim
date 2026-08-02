package actions

import (
	"fmt"
	"os"
	"strings"
	"github.com/cookiengineer/gozim/archive/zim"
)

func Unpack(args []string) {

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gozim unpack <archive.zim> --output <dir> [--ns=C]")
		os.Exit(1)
	}

	filename := args[0]
	outputDir := "."
	nameSpace := "C"

	for i, arg := range args {
		if arg == "--output" || arg == "-o" {
			if i+1 < len(args) {
				outputDir = args[i+1]
			}
		}
		if arg == "--ns" {
			if i+1 < len(args) {
				nameSpace = args[i+1]
			}
		}
	}

	archive, err := zim.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening archive: %v\n", err)
		os.Exit(1)
	}
	defer archive.Close()

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output directory: %v\n", err)
		os.Exit(1)
	}

	count := 0
	for entry := range archive.IterateByPath() {
		if nameSpace != "" && string(entry.Namespace()) != nameSpace {
			continue
		}

		item, err := entry.Item(false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", entry.Path(), err)
			continue
		}

		outPath := outputDir + "/" + entry.Path()

		dir := outPath[:strings.LastIndex(outPath, "/")]
		if dir != "" {
			os.MkdirAll(dir, 0755)
		}

		data, err := item.DataAll()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: writing %s: %v\n", outPath, err)
			continue
		}
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "warning: writing %s: %v\n", outPath, err)
			continue
		}

		count++
	}

	fmt.Printf("unpacked %d entries to %s\n", count, outputDir)
}

