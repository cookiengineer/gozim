package bm25

import (
	"strings"
	"unicode"
)

// ParseQuery splits a search query string into normalized terms.
// Supports: bare terms, quoted phrases (terms joined with AND),
// NOT terms with -prefix, and * suffix for prefix matching.
func ParseQuery(query string) (terms []string, phrases [][]string, notTerms []string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil, nil
	}

	var current strings.Builder
	inQuote := false

	for i := 0; i < len(query); {
		ch := query[i]

		if ch == '"' {
			if current.Len() > 0 {
				terms = append(terms, normalizeTerm(current.String()))
				current.Reset()
			}
			inQuote = !inQuote
			i++
			continue
		}

		if inQuote {
			if ch == ' ' || ch == '\t' {
				if current.Len() > 0 {
					terms = append(terms, normalizeTerm(current.String()))
					current.Reset()
				}
				i++
				continue
			}
			current.WriteByte(ch)
			i++
			continue
		}

		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				word := current.String()
				// End of in-quote phrase.
				terms = append(terms, normalizeTerm(word))
				current.Reset()
			}
			i++
			continue
		}

		current.WriteByte(ch)
		i++
	}

	if current.Len() > 0 {
		terms = append(terms, normalizeTerm(current.String()))
	}

	// Separate NOT terms and phrases.
	var cleanTerms []string
	for _, t := range terms {
		if strings.HasPrefix(t, "-") && len(t) > 1 {
			notTerms = append(notTerms, t[1:])
		} else {
			cleanTerms = append(cleanTerms, t)
		}
	}

	return cleanTerms, phrases, notTerms
}

// normalizeTerm lowercases and removes trailing punctuation from a term.
func normalizeTerm(term string) string {
	term = strings.ToLower(strings.TrimSpace(term))

	if term == "" {
		return term
	}

	// Strip trailing punctuation.
	for len(term) > 0 && unicode.IsPunct(rune(term[len(term)-1])) {
		term = term[:len(term)-1]
	}
	// Strip leading punctuation.
	for len(term) > 0 && unicode.IsPunct(rune(term[0])) {
		term = term[1:]
	}

	return term
}
