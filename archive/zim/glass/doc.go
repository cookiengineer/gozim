// Package glass implements a pure Go reader and writer for the Xapian Glass
// database backend used by libzim for full-text indexing within ZIM files.
//
// The Glass backend is a B-tree based key-value store using fixed-size blocks
// (default 8192 bytes). It stores posting lists, document data, term metadata,
// and user-defined values for BM25 search and ranking.
//
// This package supports both reading existing Glass databases (for interop
// with ZIM files created by libzim) and writing new databases (for ZIM files
// created by gozim).
package glass
