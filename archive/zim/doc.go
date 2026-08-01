// Package zim implements reading and writing of ZIM files, the open
// file format used to store wiki content for offline viewing.
//
// ZIM files are used primarily by the Kiwix project and other offline
// content distribution systems. The format supports compressed storage
// of HTML pages, images, and other web content with full-text search
// capabilities.
//
// # Reading ZIM files
//
// Open a ZIM file and read an entry by its path:
//
//	archive, err := zim.Open("wikipedia.zim")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer archive.Close()
//
//	entry, err := archive.EntryByPath("C/Wikipedia.html")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	item, err := entry.Item(false)
//	data, err := item.DataAll()
//
// Iterate over all entries:
//
//	for entry := range archive.IterateByPath() {
//	    fmt.Println(entry.Path(), entry.Title())
//	}
//
// # Writing ZIM files
//
// Create a new ZIM file with content entries and metadata:
//
//	writer := zim.NewWriter().
//	    SetCompression(zim.CompressionZstd).
//	    SetIndexing(true, "eng").
//	    SetClusterSize(2 * zim.Megabyte).
//	    SetMainPath("index.html")
//
//	if err := writer.Create("output.zim"); err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, item := range items {
//	    writer.AddItem(item)
//	}
//	writer.AddMetadata("Title", "My Wiki")
//	writer.AddMetadata("Language", "eng")
//	writer.Finish()
//
// # Searching
//
// Search across one or more archives with BM25 ranking:
//
//	searcher := zim.NewSearcher(archive)
//	result, err := searcher.Search("quantum physics", 0, 20)
//	for _, r := range result.Results() {
//	    fmt.Println(r.Title, r.Score)
//	}
//
// # Namespaces
//
// ZIM entries are organized into namespaces. The new scheme (v6.1+)
// uses:
//
//   - NamespaceContent (C): user-facing content
//   - NamespaceMetadata (M): archive metadata
//   - NamespaceIndex (X): full-text indexes and title listings
//
// The old scheme (v5 and v6.0) also includes:
//
//   - NamespaceArticle (A): articles (HTML)
//   - NamespaceImage (I): images
//   - NamespaceScript (J): scripts
//   - NamespaceLayout (-): CSS and templates
//
// # Compression
//
// Clusters of entries are compressed together. Supported compression:
//
//   - CompressionNone (1): no compression
//   - CompressionZstd (5): Zstandard compression (default for new files)
//   - CompressionLzma (4): LZMA2/XZ compression (legacy read support)
package zim
