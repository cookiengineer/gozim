package glass

import (
	"bytes"
	"fmt"
)

// Posting represents a document entry in a posting list.
type Posting struct {
	DocumentID uint32
	WDF        uint32 // Within-document frequency.
}

// PostingListChunk is a decoded chunk from a term's posting list.
type PostingListChunk struct {
	TermFreq   uint64
	CollFreq   uint64
	FirstDocID uint32
	LastDocID  uint32
	IsLastChunk bool
	Postings    []Posting
}

// EncodeFirstChunk encodes the first chunk of a term's posting list.
func EncodeFirstChunk(termFreq, collFreq uint64, postings []Posting, isLast bool) ([]byte, error) {
	if len(postings) == 0 {
		return nil, fmt.Errorf("glass: empty posting list")
	}

	var buf bytes.Buffer

	// Term frequency and collection frequency.
	buf.Write(packUint(termFreq))
	buf.Write(packUint(collFreq))

	// First document ID (minus 1).
	firstDocID := postings[0].DocumentID
	buf.Write(packUint(uint64(firstDocID - 1)))

	// Is last chunk flag.
	buf.WriteByte(packBool(isLast))

	// Last document ID delta.
	lastDocID := postings[len(postings)-1].DocumentID
	buf.Write(packUint(uint64(lastDocID - firstDocID)))

	// Encode postings with delta compression.
	prevDocID := firstDocID
	for _, p := range postings[1:] {
		delta := p.DocumentID - prevDocID
		buf.Write(packUint(uint64(delta - 1))) // delta-1 encoding
		buf.Write(packUint(uint64(p.WDF)))
		prevDocID = p.DocumentID
	}

	return buf.Bytes(), nil
}

// EncodeSubsequentChunk encodes a continuation chunk.
func EncodeSubsequentChunk(postings []Posting, isLast bool) ([]byte, error) {
	if len(postings) == 0 {
		return nil, fmt.Errorf("glass: empty posting list chunk")
	}

	var buf bytes.Buffer

	buf.WriteByte(packBool(isLast))

	firstDocID := postings[0].DocumentID
	lastDocID := postings[len(postings)-1].DocumentID
	buf.Write(packUint(uint64(lastDocID - firstDocID)))

	prevDocID := firstDocID
	for _, p := range postings[1:] {
		delta := p.DocumentID - prevDocID
		buf.Write(packUint(uint64(delta - 1)))
		buf.Write(packUint(uint64(p.WDF)))
		prevDocID = p.DocumentID
	}

	return buf.Bytes(), nil
}

// DecodeFirstChunk decodes a first chunk from a posting list tag.
func DecodeFirstChunk(data []byte) (*PostingListChunk, int, error) {
	chunk := &PostingListChunk{}

	termFreq, n, err := unpackUint(data)
	if err != nil {
		return nil, 0, fmt.Errorf("glass: termfreq: %w", err)
	}
	chunk.TermFreq = termFreq
	offset := n

	collFreq, n, err := unpackUint(data[offset:])
	if err != nil {
		return nil, 0, fmt.Errorf("glass: collfreq: %w", err)
	}
	chunk.CollFreq = collFreq
	offset += n

	firstDocIDminus1, n, err := unpackUint(data[offset:])
	if err != nil {
		return nil, 0, fmt.Errorf("glass: first_docid: %w", err)
	}
	chunk.FirstDocID = uint32(firstDocIDminus1 + 1)
	offset += n

	if offset >= len(data) {
		return nil, 0, fmt.Errorf("glass: truncated first chunk")
	}
	chunk.IsLastChunk = data[offset] != 0
	offset++

	lastDocIDDelta, n, err := unpackUint(data[offset:])
	if err != nil {
		return nil, 0, fmt.Errorf("glass: last_docid_delta: %w", err)
	}
	chunk.LastDocID = chunk.FirstDocID + uint32(lastDocIDDelta)
	offset += n

	// Decode individual postings.
	// The first posting's WDF is not stored directly.
	// It is collFreq minus the sum of all subsequent postings' WDFs.
	firstWDF := collFreq
	chunk.Postings = append(chunk.Postings, Posting{
		DocumentID: chunk.FirstDocID,
		WDF:        0, // Will be recomputed below.
	})

	prevDocID := chunk.FirstDocID
	for offset < len(data) {
		delta, n, err := unpackUint(data[offset:])
		if err != nil {
			break
		}
		offset += n

		wdf, n, err := unpackUint(data[offset:])
		if err != nil {
			break
		}
		offset += n

		docID := prevDocID + uint32(delta) + 1
		chunk.Postings = append(chunk.Postings, Posting{
			DocumentID: docID,
			WDF:        uint32(wdf),
		})
		firstWDF -= wdf
		prevDocID = docID
	}

	// Set the first posting's WDF.
	if firstWDF > 0 && len(chunk.Postings) > 0 {
		chunk.Postings[0].WDF = uint32(firstWDF)
	}

	return chunk, offset, nil
}

// PostingIterator provides sequential access to a term's posting list.
type PostingIterator struct {
	table     *Table
	termKey   []byte
	current   *PostingListChunk
	chunkIdx  int
	postingIdx int
	done      bool
}

// NewPostingIterator creates a posting iterator for a term.
func NewPostingIterator(table *Table, term string) (*PostingIterator, error) {
	termKey := packStringPreservingSortTerminated(term)

	chunk, err := loadPostingChunk(table, termKey)
	if err != nil {
		return nil, err
	}

	return &PostingIterator{
		table:     table,
		termKey:   termKey,
		current:   chunk,
		chunkIdx:  0,
		postingIdx: 0,
	}, nil
}

// Next advances to the next posting.
func (it *PostingIterator) Next() bool {
	if it.current == nil || it.done {
		return false
	}

	if it.postingIdx < len(it.current.Postings)-1 {
		it.postingIdx++
		return true
	}

	if it.current.IsLastChunk {
		it.done = true
		return false
	}

	// Load next chunk.
	it.chunkIdx++
	nextKey := append(it.termKey, packUintPreservingSort(uint64(it.current.Postings[0].DocumentID))...)
	chunk, err := loadPostingChunk(it.table, nextKey)
	if err != nil || chunk == nil {
		it.done = true
		return false
	}

	it.current = chunk
	it.postingIdx = 0
	return true
}

// Posting returns the current posting.
func (it *PostingIterator) Posting() Posting {
	if it.current == nil || it.postingIdx >= len(it.current.Postings) {
		return Posting{}
	}
	return it.current.Postings[it.postingIdx]
}

// loadPostingChunk loads and decodes a posting list chunk from the table.
func loadPostingChunk(table *Table, key []byte) (*PostingListChunk, error) {
	tag, ok, err := table.Get(key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	chunk, _, err := DecodeFirstChunk(tag)
	return chunk, err
}

// BuildPostingList adds a posting to a term's posting list and writes it to the table.
func BuildPostingList(table *Table, term string, posting Posting) error {
	termKey := packStringPreservingSortTerminated(term)

	// Try to load existing chunk.
	existing, _ := loadPostingChunk(table, termKey)

	if existing == nil {
		// First posting for this term.
		encoded, err := EncodeFirstChunk(1, uint64(posting.WDF), []Posting{posting}, true)
		if err != nil {
			return err
		}

		return table.Insert(termKey, encoded)
	}

	// Append to existing chunk.
	existing.Postings = append(existing.Postings, posting)
	existing.TermFreq++
	existing.CollFreq += uint64(posting.WDF)
	existing.LastDocID = posting.DocumentID

	estimatedSize := len(existing.Postings) * 4
	if estimatedSize > ChunkSize {
		// Need to start a new chunk.
		existing.IsLastChunk = false
		encoded, err := EncodeFirstChunk(existing.TermFreq, existing.CollFreq, existing.Postings, false)
		if err != nil {
			return err
		}

		if err := table.Insert(termKey, encoded); err != nil {
			return err
		}

		// Create continuation chunk.
		nextKey := append(termKey, packUintPreservingSort(uint64(posting.DocumentID))...)
		encoded, err = EncodeSubsequentChunk([]Posting{posting}, true)
		if err != nil {
			return err
		}

		return table.Insert(nextKey, encoded)
	}

	encoded, err := EncodeFirstChunk(existing.TermFreq, existing.CollFreq, existing.Postings, true)
	if err != nil {
		return err
	}

	return table.Insert(termKey, encoded)
}
