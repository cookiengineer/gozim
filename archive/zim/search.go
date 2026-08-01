package zim

import (
	"strings"

	"github.com/cookiengineer/gozim/archive/zim/bm25"
)

// Searcher performs full-text search across one or more ZIM archives.
type Searcher struct {
	archives    []*Archive
	bm25Index   *bm25.Index
	bm25Built   bool
	tokenizer   *bm25.Tokenizer
}

// NewSearcher creates a new Searcher for the given archives.
func NewSearcher(archives ...*Archive) *Searcher {
	s := &Searcher{
		archives:  archives,
		tokenizer: bm25.NewTokenizer(bm25.NewPorterStemmer()),
	}

	return s
}

// AddArchive adds an archive to the searcher.
func (s *Searcher) AddArchive(a *Archive) {
	s.archives = append(s.archives, a)
}

// Search performs a full-text search across all archives.
// If an archive has an embedded Xapian/Glass database, it uses that.
// Otherwise, it falls back to in-memory BM25 indexing.
func (s *Searcher) Search(query string, offset, limit int) (*SearchResultSet, error) {
	if limit <= 0 {
		limit = 20
	}

	// Build BM25 index from all archives (fallback).
	if !s.bm25Built {
		s.buildBM25Index()
	}

	// Parse query.
	queryTerms, _, _ := bm25.ParseQuery(query)

	if len(queryTerms) == 0 {
		return &SearchResultSet{estimated: 0}, nil
	}

	// Use BM25 for now. Glass integration will be added when
	// the Xapian database reading is fully implemented.
	results := s.bm25Index.Search(queryTerms, limit+offset)

	// Generate snippets.
	stemmer := bm25.NewPorterStemmer()
	for i := range results {
		if doc, ok := s.bm25Index.GetDocument(results[i].DocumentID); ok {
			// Re-tokenize query terms through the stemmer for snippet matching.
			var stemmedTerms []string
			for _, t := range queryTerms {
				stemmedTerms = append(stemmedTerms, stemmer.Stem(strings.ToLower(t)))
			}
			results[i].Snippet = bm25.GenerateSnippet(doc.Content, stemmedTerms, 200)
		}
	}

	// Apply pagination.
	if offset >= len(results) {
		return &SearchResultSet{estimated: len(results)}, nil
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return &SearchResultSet{
		results:   results[offset:end],
		estimated: len(results),
	}, nil
}

// buildBM25Index indexes all entries from all archives for fallback search.
func (s *Searcher) buildBM25Index() {
	if s.bm25Built {
		return
	}

	s.bm25Index = bm25.NewIndex()
	var docID uint32

	for _, archive := range s.archives {
		for entry := range archive.IterateByPath() {
			if entry.IsRedirect() || entry.Namespace().IsMetadata() || entry.Namespace().IsIndex() {
				continue
			}

			item, err := entry.Item(false)
			if err != nil {
				continue
			}

			itemData, itemErr := item.DataAll()
			if itemErr != nil {
				continue
			}
			content := string(itemData)

			tokens := s.tokenizer.Tokenize(content)

			info := &bm25.DocumentInfo{
				Path:    entry.Path(),
				Title:   entry.Title(),
				Content: content,
				WordCount: uint64(len(tokens)),
			}

			s.bm25Index.AddDocument(docID, tokens, info)
			docID++
		}
	}

	s.bm25Built = true
}

// SearchResultSet holds a paginated set of search results.
type SearchResultSet struct {
	results   []bm25.SearchResult
	estimated int
}

// Results returns the search results for the current page.
func (rs *SearchResultSet) Results() []SearchResult {
	var out []SearchResult
	for _, r := range rs.results {
		out = append(out, SearchResult{
			Path:      r.Path,
			Title:     r.Title,
			Score:     r.Score,
			Snippet:   r.Snippet,
			WordCount: r.WordCount,
		})
	}
	return out
}

// EstimatedMatches returns the total number of matching documents.
func (rs *SearchResultSet) EstimatedMatches() int {
	return rs.estimated
}

// SearchResult represents a single search hit.
type SearchResult struct {
	Path      string
	Title     string
	Score     float64
	Snippet   string
	WordCount uint64
}

// SuggestionSearcher provides title autocomplete functionality.
type SuggestionSearcher struct {
	archive *Archive
}

// NewSuggestionSearcher creates a suggestion searcher for an archive.
func NewSuggestionSearcher(archive *Archive) *SuggestionSearcher {
	return &SuggestionSearcher{archive: archive}
}

// Suggest returns title suggestions matching the given prefix.
func (s *SuggestionSearcher) Suggest(prefix string, limit int) ([]SuggestionItem, error) {
	if limit <= 0 {
		limit = 10
	}

	var matches []SuggestionItem
	prefixLower := strings.ToLower(prefix)

	for entry := range s.archive.IterateByTitle() {
		titleLower := strings.ToLower(entry.Title())
		if strings.HasPrefix(titleLower, prefixLower) {
			matches = append(matches, SuggestionItem{
				Title: entry.Title(),
				Path:  entry.Path(),
			})
			if len(matches) >= limit {
				break
			}
		}
	}

	return matches, nil
}

// SuggestionItem is a title suggestion result.
type SuggestionItem struct {
	Title   string
	Path    string
	Snippet string
}
