// Package zim implements reading and writing of ZIM files.
//
// ZIM is an open file format used to store wiki content offline.
// It is primarily used by the Kiwix project and other offline
// content distribution systems.
//
// # Reading ZIM files
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
//
//	item, err := entry.Item(false)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	data := item.DataAll()
//
// # Writing ZIM files
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
//
//	writer.AddMetadata("Title", "My Wiki")
//	writer.AddMetadata("Language", "eng")
//	writer.Finish()
//
// # Searching
//
//	searcher := zim.NewSearcher(archive)
//	results, err := searcher.Search("quantum physics", 0, 20)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, result := range results.Results() {
//	    fmt.Println(result.Title, result.Score)
//	}
package zim
