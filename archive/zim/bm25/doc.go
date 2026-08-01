// Package bm25 implements the BM25 ranking function for full-text search
// within ZIM archives.
//
// BM25 (Best Match 25) is a probabilistic relevance ranking function
// used by search engines to rank matching documents according to their
// relevance to a given search query. It improves upon the classic TF-IDF
// weighting by incorporating document length normalization.
//
// # Index
//
// The Index type provides an in-memory inverted index that maps tokens
// (words) to the documents that contain them:
//
//	idx := bm25.NewIndex()
//	idx.AddDocument(docID, tokens, docInfo)
//
//	results := idx.Search(queryTokens, limit)
//
// # Tokenizer
//
// The Tokenizer splits text into tokens. It handles:
//   - ASCII alphanumeric word boundaries
//   - Unicode letter boundaries
//   - CJK bigram segmentation (Chinese, Japanese, Korean)
//   - Lowercasing (both ASCII and Unicode)
//   - Position tracking for phrase search
//
// # Stemmer
//
// The Porter stemmer reduces words to their root form for better
// matching. The NoopStemmer passes tokens through unchanged for
// languages without stemming support.
//
// # Snippet Generation
//
// The Snippet function generates search result context snippets with
// highlighted matching terms.
package bm25
