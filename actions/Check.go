package actions

import (
	"fmt"
	"os"
	"github.com/cookiengineer/gozim/archive/zim"
)

func Check(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: gozim check <archive.zim>")
		os.Exit(1)
	}

	filename := args[0]
	checks := zim.CheckAll

	report, err := zim.CheckFile(filename, checks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error checking archive: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("checksum: %v\n", report.ChecksumOK)
	fmt.Printf("broken links: %d\n", len(report.BrokenLinks))
	fmt.Printf("redirect loops: %d\n", len(report.RedirectLoops))
	fmt.Printf("redundant entries: %d\n", len(report.Redundant))
	fmt.Printf("errors: %d\n", len(report.Errors))

	for _, errMsg := range report.Errors {
		fmt.Printf("  - %s\n", errMsg)
	}
	for _, link := range report.BrokenLinks {
		fmt.Printf("  broken link: %s -> %s (%s)\n", link.SourceEntry, link.TargetLink, link.Reason)
	}
}
