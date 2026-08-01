# Search Architecture

## Overview

Gozim supports two search backends:

1. **BM25 (default)** — Pure Go in-memory BM25 index. Built on-the-fly from Archive entries via the `Searcher`. This is the currently active backend used for all search queries.

2. **Glass (Xapian)** — Embedded Xapian database stored at `X/fulltext/xapian` in ZIM files. The `archive/zim/glass/` package implements the full Glass backend with read/write support. Integration with the `Searcher` is implemented but not yet wired — currently the Searcher always uses BM25.

## Architecture Diagram

```
                              ┌─────────────────────┐
                              │   zim.Searcher       │  (search.go)
                              │                      │
                              │ Search(query) → []   │
                              │ Suggest(prefix) → [] │
                              └──────────┬───────────┘
                                         │
                              (currently always BM25)
                                         │
                              ┌──────────▼──────────┐
                              │   bm25.Index        │  (bm25/index.go)
                              │                     │
                              │ In-memory inverted  │
                              │ index with BM25     │
                              │ scoring             │
                              │                     │
                              │ + Tokenizer         │  (bm25/tokenizer.go)
                              │ + Stemmer (Porter)  │  (bm25/stemmer.go)
                              │ + Query parser      │  (bm25/query.go)
                              │ + Snippet gen       │  (bm25/snippet.go)
                              └─────────────────────┘
```

## Search Flow

### BM25 Index Building

```
Searcher.Search("quantum physics")
  → buildBM25Index() if not already built:
      For each Archive:
        For each content entry:
          Tokenize content → tokens
          AddDocument(docID, tokens, DocumentInfo{path, title, content, wordCount})
  → ParseQuery("quantum physics") → ["quantum", "physics"]
  → bm25Index.Search(["quantum", "physics"], limit)
  → For each result:
      GenerateSnippet(content, queryTerms, 200)
  → Return SearchResultSet with ranked results
```

### BM25 Scoring Formula

```
Score(d, Q) = Σ IDF(q_i) * (f(q_i, d) * (k1 + 1)) / (f(q_i, d) + k1 * (1 - b + b * |d| / avgdl))

Where:
  IDF(q_i) = ln((N - n(q_i) + 0.5) / (n(q_i) + 0.5) + 1)
  f(q_i, d) = term frequency in document
  |d| = document length
  avgdl = average document length
  k1 = 1.2 (term frequency saturation)
  b = 0.75 (length normalization)
```

## Tokenizer

Handles:
- ASCII alphanumeric word boundaries
- Unicode letter boundaries
- CJK bigram segmentation (overlapping 2-character n-grams for Chinese, Japanese, Korean, Hangul)
  - Unicode ranges: CJK Unified (0x4E00-0x9FFF), Extension A (0x3400-0x4DBF), Hiragana (0x3040-0x309F), Katakana (0x30A0-0x30FF), Hangul (0xAC00-0xD7AF)
- ASCII fast-path lowercasing (`'A'..'Z'` → `'a'..'z'`)
- Unicode lowercasing for non-ASCII via `unicode.ToLower`
- Position tracking (enables future phrase search)

## Stemmer

Porter stemmer v1 for English (pure Go, ~140 lines). Extensible via `Stemmer` interface.

Algorithm steps:
1. **Step 1a**: Remove plural -s, -sses → -ss, -ies → -i
2. **Step 1b**: Remove -ed, -ing suffixes when preceded by a vowel
3. **Step 4**: Remove common suffixes (-ement, -ance, -ence, -able, -ible, -ment, -ness, -less, -ful, -ous, -ant, -ent, -ism, -ate, -iti, -ive, -ize, -ion) when measure > 1
4. **Step 5a**: Remove final -e when measure > 1

Also includes `NoopStemmer` for languages without stemmer support.

## Query Parser

Supports:
- Bare terms (OR semantics by default)
- Quoted phrases (currently tokenized as individual terms)
- NOT terms (`-term`) — separated from main terms
- Prefix matching (`term*`) — suffix stripped, treated as prefix match in index
- Punctuation stripping from term boundaries

## Snippet Generation

```
GenerateSnippet(content, queryTerms, maxLen=200):
  1. Find all occurrences of query terms in content (case-insensitive)
  2. Slide a window of maxLen chars across content
  3. Select window with the most term matches
  4. Extract window, highlight matching terms with ** markers
  5. Add "..." prefix/suffix if window is not at start/end
```

## Suggestion (Autocomplete)

```
SuggestionSearcher.Suggest(prefix, limit=10):
  → Iterate archive entries by title
  → Case-insensitive prefix match
  → Return matching SuggestionItem{Title, Path}
  → Stop at limit matches
```

Currently uses title-ordered iteration (binary search on title index). Future: integrate Glass/Xapian title database for substring/typo-tolerant matching.

## Performance Characteristics

- **BM25 index building**: O(total terms) — reads and tokenizes all archive content on first search
- **Search**: O(query_terms × avg_posting_list_length) — linear scan of posting lists, map-based document scoring
- **Snippet generation**: O(content_length × query_terms) — window scanning with string matching
- **Memory**: In-memory inverted index (posting lists) + full document content for snippet generation

## Future: Glass/Xapian Integration

When the Searcher detects `HasFulltextIndex() == true`, it should:
1. Read the `X/fulltext/xapian` blob from the ZIM entry
2. Open it via `glass.OpenDatabase(blob)`
3. Use `glass.PostingIterator` and `glass.DocDataTable` for search
4. Apply BM25 scoring using the Xapian term frequencies and document lengths

This is implemented in `glass/database.go` but not yet wired into `search.go`'s `Searcher`.
