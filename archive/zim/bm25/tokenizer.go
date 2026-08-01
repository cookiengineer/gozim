package bm25

import (
	"unicode"
	"unicode/utf8"
)

// Token represents a normalized text token for indexing.
type Token struct {
	Text     string
	Position uint32
}

// Tokenizer splits text into normalized tokens suitable for full-text indexing.
type Tokenizer struct {
	stemmer Stemmer
}

// NewTokenizer creates a new Tokenizer with the given stemmer.
func NewTokenizer(stemmer Stemmer) *Tokenizer {
	return &Tokenizer{stemmer: stemmer}
}

// Tokenize splits text into tokens, applying lowercasing and stemming.
func (t *Tokenizer) Tokenize(text string) []Token {
	if text == "" {
		return nil
	}

	var tokens []Token
	var position uint32
	runes := []rune(text)

	i := 0
	for i < len(runes) {
		for i < len(runes) && !isWordRune(runes[i]) {
			i++
		}

		if i >= len(runes) {
			break
		}

		start := i

		if isCJK(runes[i]) {
			// CJK characters: generate bigrams.
			for i < len(runes) && isCJK(runes[i]) {
				i++
			}

			// Single char.
			cjkChars := runes[start:i]
			for j := 0; j < len(cjkChars); j++ {
				tokens = append(tokens, Token{
					Text:     string(cjkChars[j : j+1]),
					Position: position,
				})
				position++

				if j+1 < len(cjkChars) {
					tokens = append(tokens, Token{
						Text:     string(cjkChars[j : j+2]),
						Position: position,
					})
					position++
				}
			}
		} else {
			// Non-CJK: aggregate runes until whitespace/punctuation.
			for i < len(runes) && isWordRune(runes[i]) && !isCJK(runes[i]) {
				i++
			}

			word := string(runes[start:i])
			lower := toLower(word)

			stemmed := lower
			if t.stemmer != nil {
				stemmed = t.stemmer.Stem(lower)
			}

			tokens = append(tokens, Token{
				Text:     stemmed,
				Position: position,
			})
			position++
		}
	}

	return tokens
}

// isWordRune returns true if r is a word character (letter, digit, underscore, CJK).
func isWordRune(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		return true
	}
	return isCJK(r)
}

// isCJK returns true if r is in a CJK Unicode block.
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) || // Katakana
		(r >= 0xAC00 && r <= 0xD7AF)    // Hangul
}

// toLower converts a string to lowercase (ASCII only for speed).
func toLower(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		if r >= 'A' && r <= 'Z' {
			runes[i] = r + 32
		} else if r < utf8.RuneSelf {
			// Already lowercase or non-alpha ASCII.
		} else {
			runes[i] = unicode.ToLower(r)
		}
	}
	return string(runes)
}

// Stemmer is the interface for stemming algorithms.
type Stemmer interface {
	Stem(word string) string
}
