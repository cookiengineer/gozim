// Package glass implements a pure Go reader and writer for the Xapian
// Glass database backend used by libzim for full-text indexing within
// ZIM files.
//
// The Glass backend is a B-tree based key-value store using fixed-size
// blocks (default 8192 bytes). It stores posting lists, document data,
// term metadata, and user-defined values for BM25 search and ranking.
//
// This package supports both reading existing Glass databases (for
// interoperability with ZIM files created by libzim) and writing new
// databases (for ZIM files created by gozim).
//
// # Database lifecycle
//
// Create, populate, compact, and reopen:
//
//	var uuid [16]byte // typically obtained from the ZIM header
//	db := glass.CreateDatabase(uuid)
//	db.SetMetadata(&glass.Metadata{Kind: "fulltext", Language: "eng"})
//	db.AddDocument(&glass.Document{Data: "C/article.html", Terms: ...})
//	blob, _ := db.Compact()
//
//	reopened, _ := glass.OpenDatabase(blob)
//	doc, _ := reopened.GetDocument(1)
//
// # Encoding primitives
//
// The pack subroutines encode integers in variable-length and
// sort-preserving formats. These match the on-disk encoding used
// in the Glass version file and B-tree blocks.
//
// # B-tree tables
//
// The B-tree implementation supports Insert, Get, cursors, and
// block splitting. Tables wrap B-trees with type-specific
// compression behavior.
//
// # Posting lists
//
// Posting lists map terms to the documents that contain them, using
// delta encoding and chunk continuation for efficient storage.
// BuildPostingList appends postings to a term's list; NewPostingIterator
// provides sequential access.
package glass

