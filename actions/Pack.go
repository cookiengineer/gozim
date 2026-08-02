package actions

import (
	"fmt"
	"os"
	"github.com/cookiengineer/gozim/archive/zim"
	"github.com/cookiengineer/gozim/utils"
)

func Pack(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: gozim pack <input-dir> --output <archive.zim> [--title <title>] [--language <code>]")
		os.Exit(1)
	}

	inputDir := args[0]
	outputFile := "output.zim"
	title := "Untitled"
	language := "eng"

	for i, arg := range args {
		if arg == "--output" || arg == "-o" {
			if i+1 < len(args) {
				outputFile = args[i+1]
			}
		}
		if arg == "--title" {
			if i+1 < len(args) {
				title = args[i+1]
			}
		}
		if arg == "--language" {
			if i+1 < len(args) {
				language = args[i+1]
			}
		}
	}

	writer := zim.NewWriter().
		SetCompression(zim.CompressionZstd).
		SetClusterSize(2 * zim.Megabyte).
		SetIndexing(true, language).
		SetMainPath("index.html")

	if err := writer.Create(outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "error creating archive: %v\n", err)
		os.Exit(1)
	}

	writer.AddMetadata("Title", title)
	writer.AddMetadata("Language", language)

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		os.Exit(1)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := inputDir + "/" + entry.Name()
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", entry.Name(), err)
			continue
		}

		mimeType := utils.DetectMimeType(entry.Name())
		item := zim.NewBytesItem(entry.Name(), mimeType, entry.Name(), data)
		if err := writer.AddItem(item); err != nil {
			fmt.Fprintf(os.Stderr, "error adding %s: %v\n", entry.Name(), err)
			continue
		}

		count++
	}

	if err := writer.Finish(); err != nil {
		fmt.Fprintf(os.Stderr, "error finishing archive: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("packed %d files into %s\n", count, outputFile)
}
