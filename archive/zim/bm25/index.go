package bm25

import (
	"math"
	"sort"
	"sync"
)

// PostingEntry records a single term occurrence in a document.
type PostingEntry struct {
	DocumentID uint32
	Frequency  uint32
	Positions  []uint32
}

// PostingList is a sorted list of postings for a term.
type PostingList struct {
	Term      string
	DocFreq   uint32 // Number of documents containing this term.
	Postings  []PostingEntry
	mu        sync.RWMutex
}

// Index is an in-memory inverted index for BM25 scoring.
type Index struct {
	mu         sync.RWMutex
	terms      map[string]*PostingList
	docCount   uint32
	totalLen   uint64
	docLengths map[uint32]uint32
	documents  map[uint32]*DocumentInfo
}

// DocumentInfo holds metadata about an indexed document.
type DocumentInfo struct {
	ID      uint32
	Path    string
	Title   string
	Content string
	WordCount uint64
}

// NewIndex creates an empty inverted index.
func NewIndex() *Index {
	return &Index{
		terms:      make(map[string]*PostingList),
		docLengths: make(map[uint32]uint32),
		documents:  make(map[uint32]*DocumentInfo),
	}
}

// AddDocument indexes a document and its tokens.
func (idx *Index) AddDocument(docID uint32, tokens []Token, info *DocumentInfo) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Count term frequencies within the document.
	termFreqs := make(map[string]uint32)
	termPositions := make(map[string][]uint32)

	for _, tok := range tokens {
		termFreqs[tok.Text]++
		termPositions[tok.Text] = append(termPositions[tok.Text], tok.Position)
	}

	docLen := uint32(len(tokens))
	idx.docLengths[docID] = docLen
	idx.docCount++
	idx.totalLen += uint64(docLen)

	if info != nil {
		info.ID = docID
		idx.documents[docID] = info
	}

	// Add postings for each term.
	for term, freq := range termFreqs {
		pl, ok := idx.terms[term]
		if !ok {
			pl = &PostingList{Term: term}
			idx.terms[term] = pl
		}

		pl.Postings = append(pl.Postings, PostingEntry{
			DocumentID: docID,
			Frequency:  freq,
			Positions:  termPositions[term],
		})
		pl.DocFreq++
	}
}

// Search performs a BM25-ranked search for the given terms.
func (idx *Index) Search(queryTerms []string, limit int) []SearchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.docCount == 0 || len(queryTerms) == 0 {
		return nil
	}

	avgDocLen := float64(idx.totalLen) / float64(idx.docCount)

	// Score each document.
	docScores := make(map[uint32]float64)

	for _, term := range queryTerms {
		pl, ok := idx.terms[term]
		if !ok || len(pl.Postings) == 0 {
			continue
		}

		idf := math.Log((float64(idx.docCount)-float64(pl.DocFreq)+0.5)/(float64(pl.DocFreq)+0.5) + 1.0)

		for _, posting := range pl.Postings {
			docLen := float64(idx.docLengths[posting.DocumentID])
			tf := float64(posting.Frequency)
			numerator := tf * (k1 + 1)
			denominator := tf + k1*(1-b+b*docLen/avgDocLen)
			docScores[posting.DocumentID] += idf * numerator / denominator
		}
	}

	// Sort results by score.
	var results []SearchResult
	for docID, score := range docScores {
		results = append(results, SearchResult{
			DocumentID: docID,
			Score:      score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	// Populate document info.
	for i := range results {
		if doc, ok := idx.documents[results[i].DocumentID]; ok {
			results[i].Path = doc.Path
			results[i].Title = doc.Title
			results[i].WordCount = doc.WordCount
		}
	}

	return results
}

// GetDocument returns document info by ID.
func (idx *Index) GetDocument(docID uint32) (*DocumentInfo, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	doc, ok := idx.documents[docID]
	return doc, ok
}

// DocCount returns the number of indexed documents.
func (idx *Index) DocCount() uint32 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.docCount
}

// SearchResult is a single search result.
type SearchResult struct {
	DocumentID uint32
	Path       string
	Title      string
	Score      float64
	Snippet    string
	WordCount  uint64
}

// BM25 parameters.
const (
	k1 = 1.2
	b  = 0.75
)
