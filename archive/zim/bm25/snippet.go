package bm25

import (
	"strings"
)

// GenerateSnippet extracts a relevant text snippet from document content
// based on the given query terms. The snippet highlights matching terms
// with ** markers.
func GenerateSnippet(content string, queryTerms []string, maxLen int) string {
	if content == "" || len(queryTerms) == 0 {
		if len(content) > maxLen {
			return content[:maxLen] + "..."
		}
		return content
	}

	if maxLen <= 0 {
		maxLen = 200
	}

	lower := strings.ToLower(content)

	// Find all occurrences of query terms.
	type match struct {
		start int
		end   int
	}

	var matches []match
	for _, term := range queryTerms {
		termLower := strings.ToLower(term)
		pos := 0
		for {
			idx := strings.Index(lower[pos:], termLower)
			if idx < 0 {
				break
			}
			matches = append(matches, match{start: pos + idx, end: pos + idx + len(term)})
			pos += idx + len(term)
		}
	}

	if len(matches) == 0 {
		if len(content) > maxLen {
			return content[:maxLen] + "..."
		}
		return content
	}

	// Find the window with the most matches.
	bestStart := 0
	bestCount := 0

	contentRunes := []rune(content)
	contentLen := len(contentRunes)

	for start := 0; start < contentLen; start++ {
		end := start + maxLen
		if end > contentLen {
			end = contentLen
		}

		count := 0
		for _, m := range matches {
			if m.start >= start && m.start < end {
				count++
			}
		}

		if count > bestCount {
			bestCount = count
			bestStart = start
		}
	}

	// Extract window.
	end := bestStart + maxLen
	if end > contentLen {
		end = contentLen
	}

	snippet := string(contentRunes[bestStart:end])

	// Highlight matching terms.
	for _, term := range queryTerms {
		termLower := strings.ToLower(term)
		snippetLower := strings.ToLower(snippet)

		pos := 0
		var result strings.Builder
		for {
			idx := strings.Index(snippetLower[pos:], termLower)
			if idx < 0 {
				result.WriteString(snippet[pos:])
				break
			}

			result.WriteString(snippet[pos : pos+idx])
			result.WriteString("**")
			result.WriteString(snippet[pos+idx : pos+idx+len(term)])
			result.WriteString("**")
			pos += idx + len(term)
			snippetLower = strings.ToLower(snippet[pos:])
		}
		snippet = result.String()
	}

	if end < contentLen {
		snippet += "..."
	}
	if bestStart > 0 {
		snippet = "..." + snippet
	}

	return snippet
}
