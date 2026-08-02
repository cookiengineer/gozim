package actions

import (
	"fmt"
	"os"
	"github.com/cookiengineer/gozim/archive/zim"
)

func Search(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gozim search <archive.zim> <query> [--limit N]")
		os.Exit(1)
	}

	filename := args[0]
	query := args[1]
	limit := 20

	for i, arg := range args {
		if arg == "--limit" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &limit)
		}
	}

	archive, err := zim.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening archive: %v\n", err)
		os.Exit(1)
	}
	defer archive.Close()

	searcher := zim.NewSearcher(archive)
	results, err := searcher.Search(query, 0, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error searching: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("estimated matches: %d\n\n", results.EstimatedMatches())

	for i, result := range results.Results() {
		fmt.Printf("%d. %s (score: %.2f)\n", i+1, result.Title, result.Score)
		fmt.Printf("   path: %s\n", result.Path)
		if result.Snippet != "" {
			fmt.Printf("   snippet: %s\n", result.Snippet)
		}
		fmt.Println()
	}
}

