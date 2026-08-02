package actions

import (
	"fmt"
	"os"
	"github.com/cookiengineer/gozim/archive/zim"
)

func Dump(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: gozim dump <archive.zim> [--list] [--ns=C]")
		os.Exit(1)
	}

	filename := args[0]
	listMode := false
	nameSpace := ""

	for i, arg := range args {
		if arg == "--list" {
			listMode = true
		}
		if arg == "--ns" && i+1 < len(args) {
			nameSpace = args[i+1]
		}
	}

	archive, err := zim.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening archive: %v\n", err)
		os.Exit(1)
	}
	defer archive.Close()

	for entry := range archive.IterateByPath() {
		if nameSpace != "" && string(entry.Namespace()) != nameSpace {
			continue
		}

		if listMode {
			fmt.Printf("%s\n", entry.Path())
		} else {
			item, err := entry.Item(false)
			if err != nil {
				fmt.Printf("[%s] %s (error: %v)\n", entry.Namespace(), entry.Path(), err)
				continue
			}
			fmt.Printf("[%s] %s  title=%q  mime=%s  size=%d\n",
				entry.Namespace(), entry.Path(), entry.Title(),
				item.MimeType(), item.Size())
		}
	}
}

