package actions

import (
	"fmt"
	"os"
	"github.com/cookiengineer/gozim/archive/zim"
	"github.com/cookiengineer/gozim/tracers"
)

func Trace(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: gozim trace <archive.zim>")
		os.Exit(1)
	}

	filename := args[0]

	archive, err := zim.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening archive: %v\n", err)
		os.Exit(1)
	}
	defer archive.Close()

	reader := archive.RawReader()

	fmt.Printf("\n")
	fmt.Printf("> ZIM Trace: %s\n", filename)
	fmt.Printf("> File size: %d bytes\n", reader.Size())
	fmt.Printf("\n")

	tracers.TraceHeader(archive, reader)
	tracers.TraceMimeList(archive, reader)
	tracers.TraceUrlPointers(archive, reader)
	tracers.TraceDirEntries(archive, reader)
	tracers.TraceTitleIndex(archive, reader)
	tracers.TraceClusterPointers(archive, reader)
	tracers.TraceClusters(archive, reader)
	tracers.TraceChecksum(archive, reader)
	tracers.TraceIntegrity(filename)

}

