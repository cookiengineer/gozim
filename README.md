# GoZIM

Minimal OpenZIM compliant implementation in Pure Go with zero dependencies
and with Go stdlib's `archive/*` and `compress/*` interface compatibility.

## Usage

```bash
# Inspect a ZIM file with structured debug output
gozim trace wikipedia.zim;

# Print file metadata: version, entries, checksum, namespace scheme
gozim info wikipedia.zim;

# List all entries by namespace+path order
gozim dump wikipedia.zim --list;

# Show entry details: path, title, MIME type, size
gozim dump wikipedia.zim;

# Filter by namespace (C=content, A=articles, I=images, M=metadata)
gozim dump wikipedia.zim --ns=C;

# Full-text search with BM25 ranking
gozim search wikipedia.zim "quantum physics" --limit 20;

# Extract archive contents to a directory
gozim unpack wikipedia.zim --output ./wiki/;

# Pack a directory into a ZIM archive
gozim pack ./html-content --output archive.zim --title "My Wiki" --language eng;

# Verify file integrity: checksum, broken links, redirect loops, redundancy
gozim check wikipedia.zim --full;

# Print version
gozim version;
```

## GoZIM API

```go
package main

import (
    "fmt"
    "github.com/cookiengineer/gozim/archive/zim"
)

func main() {

    // Read ZIM file
    archive, _ := zim.Open("wikipedia.zim")
    defer archive.Close()

    fmt.Println("entries:", archive.EntryCount())
    fmt.Println("uuid:", archive.Uuid())

    // Look up entry by path
    entry, _ := archive.EntryByPath("index.html")
    item, _ := entry.Item(false)
    data, _ := item.DataAll()
    fmt.Printf("content: %d bytes, MIME: %s\n", len(data), item.MimeType())

    // Resolve a redirect
    if entry.IsRedirect() {
        target, _ := entry.RedirectEntry()
        fmt.Println("redirects to:", target.Path())
    }

    // Iterate all entries
    for entry := range archive.IterateByPath() {
        fmt.Println(entry.Path(), entry.Title())
    }

    // Read metadata
    if title, ok := archive.Metadata("Title"); ok {
        fmt.Println("title:", title)
    }

    // Write ZIM file
    writer := zim.NewWriter().
        SetCompression(zim.CompressionZstd).
        SetClusterSize(2 * zim.Megabyte).
        SetIndexing(true, "eng").
        SetMainPath("index.html")

    writer.Create("output.zim")

    // Add content items
    item1 := zim.NewStringItem("page.html", "text/html", "My Page", "<html>...</html>")
    writer.AddItem(item1)

    // Add metadata
    writer.AddMetadata("Title", "My Wiki")
    writer.AddMetadata("Language", "eng")

    // Add redirect
    writer.AddRedirection("old.html", "Old Page", "new.html")

    // Add a favicon
    pngData, _ := os.ReadFile("favicon.png")
    writer.AddIllustration(48, pngData)

    writer.Finish()

    // Search entries
    searcher := zim.NewSearcher(archive)
    results, _ := searcher.Search("quantum physics", 0, 20)
    for _, r := range results.Results() {
        fmt.Printf("%s (score: %.2f)\n", r.Title, r.Score)
        fmt.Println(r.Snippet)
    }

    // Integrity checks
    report, _ := zim.CheckFile("wikipedia.zim", zim.CheckAll)
    fmt.Println("checksum OK:", report.ChecksumOK)
    fmt.Println("broken links:", len(report.BrokenLinks))
    fmt.Println("redirect loops:", len(report.RedirectLoops))

}
```

## GoZIM's Compression Packages

```go
import "github.com/cookiengineer/gozim/compress/lzma"
import "github.com/cookiengineer/gozim/compress/xz"
import "github.com/cookiengineer/gozim/compress/zstd"

// zstd
var buf bytes.Buffer
encoder, _ := zstd.NewWriter(&buf)
encoder.Write(data)
encoder.Close()
decoder, _ := zstd.NewReader(&buf)
decompressed, _ := io.ReadAll(decoder)
decoder.Close()

// xz
xzReader, _ := xz.NewReader(compressedReader)
data, _ := io.ReadAll(xzReader)

// lzma
lzmaWriter, _ := lzma.NewWriter(&buf)
lzmaWriter.Write(data)
lzmaWriter.Close()
```

## Unit Tests

The library uses Go's standard test format for unit tests and integration tests.

```bash
cd /path/to/gozim;

# Run the test suite
go test -v ./...;
```

The current test coverage is:

| Package                   | Coverage  | Tests   |
|:--------------------------|----------:|--------:|
| `archive/zim`             |     52.3% |      99 |
| `archive/zim/glass`       |     67.6% |      40 |
| `compress/lzma`           |     67.9% |      65 |
| `compress/lzma/hash`      |    100.0% |      25 |
| `compress/xz`             |     70.8% |      22 |
| `compress/zstd`           |     43.6% |      64 |
| `compress/zstd/huff0`     |     51.5% |      22 |
| `compress/zstd/huff0/fse` |      0.0% |       0 |
| `compress/zstd/le`        |      0.0% |       0 |
| `compress/zstd/snapref`   |    100.0% |       5 |
| `compress/zstd/xxhash`    |    100.0% |      21 |
| **Total**                 | **47.0%** | **363** |

## Specification Compliance Tests

Each package has a specification compliance test suite, and the specifications are
included in the repository in the [specifications](./specifications) folder for
future reference.

- [archive/zim compliance tests](./archive/zim/zim_spec_test.go)
- [compress/lzma compliance tests](./compress/lzma/lzma_spec_test.go)
- [compress/xz compliance tests](./compress/xz/xz_spec_test.go)
- [compress/zstd compliance tests](./compress/zstd/zstd_spec_test.go)

## License

This library is licensed under the [X11/MIT License](./LICENSE.txt)

