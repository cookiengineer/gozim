package main

import (
	"fmt"
	"os"
	"strings"
	"github.com/cookiengineer/gozim/actions"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := strings.ToLower(os.Args[1])
	args := os.Args[2:]

	switch command {
	case "unpack":
		actions.Unpack(args)
	case "pack":
		actions.Pack(args)
	case "search":
		actions.Search(args)
	case "check":
		actions.Check(args)
	case "dump":
		actions.Dump(args)
	case "info":
		actions.Info(args)
	case "trace":
		actions.Trace(args)
	case "help", "-h", "--help":
		printUsage()
	case "version", "--version":
		fmt.Println("gozim v0.1.1")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`gozim - ZIM file toolkit

Usage:
  gozim <command> [arguments]

Commands:
  unpack    Extract ZIM contents to a directory
  pack      Create a ZIM file from a directory
  search    Full-text search a ZIM file
  check     Verify ZIM file integrity
  dump      Dump entries/blobs from a ZIM file
  info      Print ZIM file metadata and structure
  trace     Print detailed internal structure for debugging
  version   Print version information
  help      Show this help message

Examples:
  gozim unpack wikipedia.zim --output ./out/
  gozim pack ./content/ --output archive.zim --title "My Wiki"
  gozim search archive.zim "quantum physics" --limit 20
  gozim check archive.zim
  gozim info archive.zim
  gozim trace archive.zim`)
}

