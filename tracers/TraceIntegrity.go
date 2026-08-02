package tracers

import (
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
)

func TraceIntegrity(filename string) {

	fmt.Println()
	fmt.Println("> Section: Integrity")
	fmt.Println()

	report, err := zim.CheckFile(filename, zim.CheckAll)
	if err != nil {
		fmt.Printf("  error: %v\n\n", err)
		return
	}

	checkMark := func(ok bool) string {
		if ok {
			return "✓"
		}
		return "✗"
	}

	fmt.Printf("  checksum:          %s\n", checkMark(report.ChecksumOK))
	fmt.Printf("  broken links:       %d\n", len(report.BrokenLinks))
	fmt.Printf("  redirect loops:     %d\n", len(report.RedirectLoops))
	fmt.Printf("  redundant pairs:    %d\n", len(report.Redundant))
	fmt.Printf("  other errors:       %d\n", len(report.Errors))

	if len(report.BrokenLinks) > 0 {
		maxShow := 10
		fmt.Printf("  broken link detail (first %d):\n", min(maxShow, len(report.BrokenLinks)))
		for i, link := range report.BrokenLinks {
			if i >= maxShow {
				break
			}
			fmt.Printf("    %s → %s (%s)\n", link.SourceEntry, link.TargetLink, link.Reason)
		}
	}

	if len(report.RedirectLoops) > 0 {
		fmt.Printf("  redirect loops:\n")
		for i, loop := range report.RedirectLoops {
			fmt.Printf("    loop %d: entries %v\n", i, loop)
		}
	}

	if len(report.Errors) > 0 {
		maxShow := 10
		fmt.Printf("  error details (first %d):\n", min(maxShow, len(report.Errors)))
		for i, errMsg := range report.Errors {
			if i >= maxShow {
				break
			}
			fmt.Printf("    %s\n", errMsg)
		}
	}

	fmt.Println()
}


