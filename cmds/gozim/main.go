package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/cookiengineer/gozim/archive/zim"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "unpack":
		cmdUnpack(args)
	case "pack":
		cmdPack(args)
	case "search":
		cmdSearch(args)
	case "check":
		cmdCheck(args)
	case "split":
		cmdSplit(args)
	case "dump":
		cmdDump(args)
	case "info":
		cmdInfo(args)
	case "trace":
		cmdTrace(args)
	case "version":
		fmt.Println("gozim v0.1.0")
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`gozim - ZIM file format toolkit

Usage:
  gozim <command> [arguments]

Commands:
  unpack    Extract ZIM contents to a directory
  pack      Create a ZIM file from a directory
  search    Full-text search a ZIM file
  check     Verify ZIM file integrity
  split     Split a ZIM file into multipart parts
  dump      Dump entries/blobs from a ZIM file
  info      Print ZIM file metadata and structure
  trace     Print detailed internal structure for debugging
  version   Print version information
  help      Show this help message

Global flags:
  --workers, -w N   Number of worker goroutines (default: GOMAXPROCS)

Examples:
  gozim unpack wikipedia.zim --output ./out/
  gozim pack ./content/ --output archive.zim --title "My Wiki"
  gozim search archive.zim "quantum physics" --limit 20
  gozim check archive.zim --full
  gozim info archive.zim
  gozim trace archive.zim`)
}

func parseWorkers(args []string) int {
	for i, arg := range args {
		if (arg == "--workers" || arg == "-w") && i+1 < len(args) {
			n := 0
			fmt.Sscanf(args[i+1], "%d", &n)
			if n > 0 {
				return n
			}
		}
	}
	return runtime.NumCPU()
}

func cmdUnpack(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gozim unpack <archive.zim> --output <dir> [--ns=C]")
		os.Exit(1)
	}

	filename := args[0]
	outputDir := "."
	nameSpace := "C"

	for i, arg := range args {
		if arg == "--output" || arg == "-o" {
			if i+1 < len(args) {
				outputDir = args[i+1]
			}
		}
		if arg == "--ns" {
			if i+1 < len(args) {
				nameSpace = args[i+1]
			}
		}
	}

	archive, err := zim.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening archive: %v\n", err)
		os.Exit(1)
	}
	defer archive.Close()

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output directory: %v\n", err)
		os.Exit(1)
	}

	count := 0
	for entry := range archive.IterateByPath() {
		if nameSpace != "" && string(entry.Namespace()) != nameSpace {
			continue
		}

		item, err := entry.Item(false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", entry.Path(), err)
			continue
		}

		outPath := outputDir + "/" + entry.Path()

		dir := outPath[:strings.LastIndex(outPath, "/")]
		if dir != "" {
			os.MkdirAll(dir, 0755)
		}

		data, err := item.DataAll()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: writing %s: %v\n", outPath, err)
			continue
		}
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "warning: writing %s: %v\n", outPath, err)
			continue
		}

		count++
	}

	fmt.Printf("unpacked %d entries to %s\n", count, outputDir)
}

func cmdPack(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: gozim pack <input-dir> --output <archive.zim> [--title <title>] [--language <code>]")
		os.Exit(1)
	}

	inputDir := args[0]
	outputFile := "output.zim"
	title := "Untitled"
	language := "eng"

	for i, arg := range args {
		if arg == "--output" || arg == "-o" {
			if i+1 < len(args) {
				outputFile = args[i+1]
			}
		}
		if arg == "--title" {
			if i+1 < len(args) {
				title = args[i+1]
			}
		}
		if arg == "--language" {
			if i+1 < len(args) {
				language = args[i+1]
			}
		}
	}

	writer := zim.NewWriter().
		SetCompression(zim.CompressionZstd).
		SetClusterSize(2 * zim.Megabyte).
		SetIndexing(true, language).
		SetMainPath("index.html")

	if err := writer.Create(outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "error creating archive: %v\n", err)
		os.Exit(1)
	}

	writer.AddMetadata("Title", title)
	writer.AddMetadata("Language", language)

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading directory: %v\n", err)
		os.Exit(1)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := inputDir + "/" + entry.Name()
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", entry.Name(), err)
			continue
		}

		mimeType := detectMimeType(entry.Name())
		item := zim.NewBytesItem(entry.Name(), mimeType, entry.Name(), data)
		if err := writer.AddItem(item); err != nil {
			fmt.Fprintf(os.Stderr, "error adding %s: %v\n", entry.Name(), err)
			continue
		}

		count++
	}

	if err := writer.Finish(); err != nil {
		fmt.Fprintf(os.Stderr, "error finishing archive: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("packed %d files into %s\n", count, outputFile)
}

func cmdSearch(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gozim search <archive.zim> <query> [--limit N]")
		os.Exit(1)
	}

	filename := args[0]
	query := args[1]
	limit := 20

	for i, arg := range args {
		if arg == "--limit" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &limit)
		}
	}

	archive, err := zim.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening archive: %v\n", err)
		os.Exit(1)
	}
	defer archive.Close()

	searcher := zim.NewSearcher(archive)
	results, err := searcher.Search(query, 0, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error searching: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("estimated matches: %d\n\n", results.EstimatedMatches())

	for i, result := range results.Results() {
		fmt.Printf("%d. %s (score: %.2f)\n", i+1, result.Title, result.Score)
		fmt.Printf("   path: %s\n", result.Path)
		if result.Snippet != "" {
			fmt.Printf("   snippet: %s\n", result.Snippet)
		}
		fmt.Println()
	}
}

func cmdCheck(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: gozim check <archive.zim> [--full]")
		os.Exit(1)
	}

	filename := args[0]
	checks := zim.CheckAll

	report, err := zim.CheckFile(filename, checks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error checking archive: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("checksum: %v\n", report.ChecksumOK)
	fmt.Printf("broken links: %d\n", len(report.BrokenLinks))
	fmt.Printf("redirect loops: %d\n", len(report.RedirectLoops))
	fmt.Printf("redundant entries: %d\n", len(report.Redundant))
	fmt.Printf("errors: %d\n", len(report.Errors))

	for _, errMsg := range report.Errors {
		fmt.Printf("  - %s\n", errMsg)
	}
	for _, link := range report.BrokenLinks {
		fmt.Printf("  broken link: %s -> %s (%s)\n", link.SourceEntry, link.TargetLink, link.Reason)
	}
}

func cmdSplit(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: gozim split <archive.zim> --max-size <bytes> --output-prefix <prefix>")
		os.Exit(1)
	}
	fmt.Println("split is not yet implemented")
}

func cmdDump(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: gozim dump <archive.zim> [--list] [--ns=C]")
		os.Exit(1)
	}

	filename := args[0]
	listMode := false
	nameSpace := ""

	for i, arg := range args {
		if arg == "--list" {
			listMode = true
		}
		if arg == "--ns" && i+1 < len(args) {
			nameSpace = args[i+1]
		}
	}

	archive, err := zim.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening archive: %v\n", err)
		os.Exit(1)
	}
	defer archive.Close()

	for entry := range archive.IterateByPath() {
		if nameSpace != "" && string(entry.Namespace()) != nameSpace {
			continue
		}

		if listMode {
			fmt.Printf("%s\n", entry.Path())
		} else {
			item, err := entry.Item(false)
			if err != nil {
				fmt.Printf("[%s] %s (error: %v)\n", entry.Namespace(), entry.Path(), err)
				continue
			}
			fmt.Printf("[%s] %s  title=%q  mime=%s  size=%d\n",
				entry.Namespace(), entry.Path(), entry.Title(),
				item.MimeType(), item.Size())
		}
	}
}

func cmdInfo(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: gozim info <archive.zim>")
		os.Exit(1)
	}

	filename := args[0]
	archive, err := zim.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening archive: %v\n", err)
		os.Exit(1)
	}
	defer archive.Close()

	fmt.Printf("Archive:        %s\n", filename)
	fmt.Printf("UUID:           %s\n", archive.Uuid())
	fmt.Printf("Version:        %d.%d\n", zim.MajorVersion, zim.MinorVersion)
	fmt.Printf("Entries:        %d\n", archive.EntryCount())
	fmt.Printf("Articles:       %d\n", archive.ArticleCount())
	fmt.Printf("Media:          %d\n", archive.MediaCount())
	fmt.Printf("Clusters:       %d\n", archive.ClusterCount())
	fmt.Printf("Checksum:       %s\n", archive.Checksum())
	fmt.Printf("Checksum OK:    %v\n", archive.HasChecksum())
	fmt.Printf("Fulltext Index: %v\n", archive.HasFulltextIndex())
	fmt.Printf("Title Index:    %v\n", archive.HasTitleIndex())
	fmt.Printf("New Namespace:  %v\n", archive.HasNewNamespaceScheme())
	fmt.Printf("Multipart:      %v\n", archive.IsMultiPart())

	if title, ok := archive.Metadata("Title"); ok {
		fmt.Printf("Title:          %s\n", title)
	}
	if lang, ok := archive.Metadata("Language"); ok {
		fmt.Printf("Language:       %s\n", lang)
	}
	if creator, ok := archive.Metadata("Creator"); ok {
		fmt.Printf("Creator:        %s\n", creator)
	}
}

func cmdTrace(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: gozim trace <archive.zim>")
		os.Exit(1)
	}

	filename := args[0]

	archive, err := zim.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening archive: %v\n", err)
		os.Exit(1)
	}
	defer archive.Close()

	reader := archive.RawReader()

	fmt.Printf("══════════════════════════════════════════════════════════════\n")
	fmt.Printf("  ZIM Trace: %s\n", filename)
	fmt.Printf("  File size: %d bytes\n", reader.Size())
	fmt.Printf("══════════════════════════════════════════════════════════════\n\n")

	traceHeader(archive, reader)
	traceMimeList(archive, reader)
	traceUrlPointers(archive, reader)
	traceDirents(archive, reader)
	traceTitleIndex(archive, reader)
	traceClusterPointers(archive, reader)
	traceClusters(archive, reader)
	traceChecksum(archive, reader)
	traceIntegrity(filename)
}

func traceHeader(a *zim.Archive, r zim.Reader) {
	fmt.Println("─── Section: Header (80 bytes at offset 0)")

	headerData := readRaw(r, 0, 80)
	fmt.Printf("  Raw bytes: %s\n", hexDisplay(headerData))
	fmt.Println()

	h := a.Header()
	fmt.Printf("  magicNumber:      0x%08X (%d)\n", h.MagicNumber, h.MagicNumber)
	fmt.Printf("  majorVersion:     %d\n", h.MajorVersion)
	fmt.Printf("  minorVersion:     %d\n", h.MinorVersion)
	fmt.Printf("  uuid:             %s\n", h.Uuid)
	fmt.Printf("  entryCount:       %d\n", h.EntryCount)
	fmt.Printf("  clusterCount:     %d\n", h.ClusterCount)
	fmt.Printf("  urlPtrPos:        0x%016X (%d)\n", h.UrlPtrPos, h.UrlPtrPos)
	fmt.Printf("  titleIdxPos:      0x%016X (%d)", h.TitleIdxPos, h.TitleIdxPos)
	if h.TitleIdxPos == 0xFFFFFFFFFFFFFFFF {
		fmt.Print(" = none")
	}
	fmt.Println()
	fmt.Printf("  clusterPtrPos:    0x%016X (%d)\n", h.ClusterPtrPos, h.ClusterPtrPos)
	fmt.Printf("  mimeListPos:      0x%016X (%d)\n", h.MimeListPos, h.MimeListPos)
	fmt.Printf("  mainPage:         %d", h.MainPage)
	if h.MainPage == 0xFFFFFFFF {
		fmt.Print(" = none")
	}
	fmt.Println()
	fmt.Printf("  layoutPage:       %d", h.LayoutPage)
	if h.LayoutPage == 0xFFFFFFFF {
		fmt.Print(" = none")
	}
	fmt.Println()
	fmt.Printf("  checksumPos:      0x%016X (%d)\n", h.ChecksumPos, h.ChecksumPos)
	fmt.Println()
	fmt.Printf("  hasChecksum:       %v\n", h.HasChecksum())
	fmt.Printf("  newNamespaceScheme: %v\n", h.HasNewNamespaceScheme())
	fmt.Printf("  hasTitleIndex:     %v\n", h.HasTitleIndex())
	fmt.Printf("  hasMainPage:       %v\n", h.HasMainPage())
	fmt.Println()
}

func traceMimeList(a *zim.Archive, r zim.Reader) {
	fmt.Println("─── Section: MIME List")

	ml := a.MimeTypeList()
	h := a.Header()
	mimeListSize := int(h.UrlPtrPos - h.MimeListPos)
	mimeData := readRaw(r, int64(h.MimeListPos), mimeListSize)

	fmt.Printf("  location: 0x%016X, length: %d bytes\n", h.MimeListPos, mimeListSize)
	fmt.Printf("  first 64 bytes: %s\n", hexDisplay(truncateBytes(mimeData, 64)))
	fmt.Println()

	count := ml.Len()
	if count == 0 {
		fmt.Println("  (empty)")
		return
	}

	for i := 0; i < count; i++ {
		mime := ml.MimeType(uint16(i))
		if mime == "" {
			fmt.Printf("  [%3d] (null string)\n", i)
		} else {
			fmt.Printf("  [%3d] %q\n", i, mime)
		}
	}
	fmt.Printf("  (%d entries)\n\n", count)
}

func traceUrlPointers(a *zim.Archive, r zim.Reader) {
	fmt.Println("─── Section: URL Pointer List")

	offsets := a.UrlPtrOffsets()
	h := a.Header()
	count := len(offsets)

	ptrData := readRaw(r, int64(h.UrlPtrPos), count*8)
	fmt.Printf("  location: 0x%016X, %d entries (%d bytes)\n", h.UrlPtrPos, count, count*8)
	fmt.Printf("  first 64 bytes: %s\n", hexDisplay(truncateBytes(ptrData, 64)))
	fmt.Println()

	maxShow := 30
	if count > maxShow {
		fmt.Printf("  (showing first %d of %d entries)\n", maxShow, count)
	}
	for i := 0; i < count && i < maxShow; i++ {
		off := offsets[i]
		fmt.Printf("  [%3d] → dirent at 0x%016X (%d)\n", i, off, off)
	}
	if count > maxShow {
		fmt.Printf("  ... (%d more entries)\n", count-maxShow)
	}
	fmt.Println()
}

func traceDirents(a *zim.Archive, r zim.Reader) {
	fmt.Println("─── Section: Directory Entries")

	dirents := a.Dirents()
	urlPtrs := a.UrlPtrOffsets()
	ml := a.MimeTypeList()

	fmt.Printf("  %d entries\n\n", len(dirents))

	maxShow := 40
	if len(dirents) > maxShow {
		fmt.Printf("  (showing first %d of %d entries)\n\n", maxShow, len(dirents))
	}

	for i := 0; i < len(dirents) && i < maxShow; i++ {
		d := dirents[i]
		direntOff := urlPtrs[i]

		rawBytes := readRaw(r, int64(direntOff), 512)
		_, direntSize, _ := zim.ParseDirent(rawBytes)
		direntData := rawBytes[:minInt(direntSize, len(rawBytes))]

		fmt.Printf("  [%3d] offset 0x%X (%d), %d raw bytes\n", i, direntOff, direntOff, direntSize)
		fmt.Printf("        namespace:    %c (%s)\n", d.Namespace, describeNamespace(d.Namespace))
		fmt.Printf("        path:         %q\n", d.Path)
		fmt.Printf("        title:        %q\n", truncateStr(d.Title, 60))
		fmt.Printf("        mime index:   %d", d.MimeTypeIndex)
		if mime := ml.MimeType(d.MimeTypeIndex); mime != "" {
			fmt.Printf(" (%s)", mime)
		}
		fmt.Println()

		switch {
		case d.IsRedirect():
			fmt.Printf("        type:         redirect → entry %d\n", d.RedirectIndex)
		case d.IsLinkTarget():
			fmt.Printf("        type:         linktarget (cluster %d, blob %d)\n", d.ClusterNumber, d.BlobNumber)
		case d.IsDeleted():
			fmt.Printf("        type:         deleted\n")
		default:
			fmt.Printf("        type:         content (cluster %d, blob %d)\n", d.ClusterNumber, d.BlobNumber)
		}

		if len(d.Extra) > 0 {
			fmt.Printf("        extra params: %d bytes: %s\n", len(d.Extra), hexDisplay(d.Extra))
		}

		hexStr := hexDisplay(truncateBytes(direntData, 48))
		fmt.Printf("        raw hex:      %s\n", hexStr)
		fmt.Println()
	}

	if len(dirents) > maxShow {
		fmt.Printf("  ... (%d more entries)\n\n", len(dirents)-maxShow)
	}
}

func traceTitleIndex(a *zim.Archive, r zim.Reader) {
	fmt.Println("─── Section: Title Index")

	h := a.Header()
	titleIdx := a.TitleIndices()

	if titleIdx == nil {
		fmt.Println("  (no title index loaded)")
		fmt.Println()
		return
	}

	if h.HasTitleIndex() && h.TitleIdxPos != 0xFFFFFFFFFFFFFFFF {
		size := len(titleIdx) * 4
		rawBytes := readRaw(r, int64(h.TitleIdxPos), size)
		fmt.Printf("  v0 location: 0x%016X, %d entries (%d bytes)\n", h.TitleIdxPos, len(titleIdx), size)
		fmt.Printf("  first 64 bytes: %s\n", hexDisplay(truncateBytes(rawBytes, 64)))
	}

	fmt.Printf("  title entries: %d\n", len(titleIdx))
	maxShow := 20
	if len(titleIdx) > maxShow {
		fmt.Printf("  (showing first %d of %d)\n", maxShow, len(titleIdx))
	}

	dirents := a.Dirents()
	for i := 0; i < len(titleIdx) && i < maxShow; i++ {
		entryIdx := titleIdx[i]
		if int(entryIdx) < len(dirents) {
			d := dirents[entryIdx]
			fmt.Printf("  [%3d] entry %d: %q (title: %q)\n", i, entryIdx, d.Path, truncateStr(d.Title, 50))
		} else {
			fmt.Printf("  [%3d] entry %d: (out of range)\n", i, entryIdx)
		}
	}
	if len(titleIdx) > maxShow {
		fmt.Printf("  ... (%d more)\n", len(titleIdx)-maxShow)
	}
	fmt.Println()
}

func traceClusterPointers(a *zim.Archive, r zim.Reader) {
	fmt.Println("─── Section: Cluster Pointer List")

	h := a.Header()
	count := int(h.ClusterCount)
	ptrData := readRaw(r, int64(h.ClusterPtrPos), count*8)

	fmt.Printf("  location: 0x%016X, %d entries (%d bytes)\n", h.ClusterPtrPos, count, count*8)
	fmt.Printf("  first 64 bytes: %s\n", hexDisplay(truncateBytes(ptrData, 64)))
	fmt.Println()

	var clusterOffsets []uint64
	for i := 0; i < count; i++ {
		off := binary.LittleEndian.Uint64(ptrData[i*8:])
		clusterOffsets = append(clusterOffsets, off)
	}

	for i, off := range clusterOffsets {
		endOff := uint64(r.Size())
		if i+1 < len(clusterOffsets) {
			endOff = clusterOffsets[i+1]
		}
		if h.HasChecksum() && i+1 >= len(clusterOffsets) {
			endOff = h.ChecksumPos
		}
		size := endOff - off
		fmt.Printf("  [%3d] offset 0x%X (%d), size %d bytes (ends at 0x%X)\n", i, off, off, size, endOff)
	}
	fmt.Println()
}

func traceClusters(a *zim.Archive, r zim.Reader) {
	fmt.Println("─── Section: Clusters")

	h := a.Header()
	count := int(h.ClusterCount)

	ptrData := readRaw(r, int64(h.ClusterPtrPos), count*8)
	var clusterOffsets []uint64
	for i := 0; i < count; i++ {
		off := binary.LittleEndian.Uint64(ptrData[i*8:])
		clusterOffsets = append(clusterOffsets, off)
	}

	maxShow := 10
	if count > maxShow {
		fmt.Printf("  (showing first %d of %d clusters)\n\n", maxShow, count)
	}

	for i := 0; i < count && i < maxShow; i++ {
		off := clusterOffsets[i]
		endOff := uint64(r.Size())
		if i+1 < len(clusterOffsets) {
			endOff = clusterOffsets[i+1]
		}
		if h.HasChecksum() && i+1 >= len(clusterOffsets) {
			endOff = h.ChecksumPos
		}

		clusterBytes := readRaw(r, int64(off), int(endOff-off))
		traceClusterDetail(i, off, clusterBytes)
	}

	if count > maxShow {
		fmt.Printf("  ... (%d more clusters)\n\n", count-maxShow)
	}
}

func traceClusterDetail(index int, offset uint64, data []byte) {
	divider := strings.Repeat("─", 50)
	fmt.Printf("  %s\n", divider)
	fmt.Printf("  Cluster %d at offset 0x%X, %d raw bytes\n", index, offset, len(data))
	fmt.Printf("  %s\n", divider)

	if len(data) < 1 {
		fmt.Println("  (empty)")
		fmt.Println()
		return
	}

	info := zim.ParseClusterInfo(data[0])
	fmt.Printf("  info byte:         0x%02X\n", data[0])
	fmt.Printf("  compression:        %d (%s)\n", info.Compression, compressionName(info.Compression))
	fmt.Printf("  extended offsets:   %v\n", info.IsExtended)

	offsetSize := 4
	if info.IsExtended {
		offsetSize = 8
	}

	if info.Compression == zim.CompressionNone {
		if len(data) < 1+offsetSize {
			fmt.Println("  (too short for offset table)")
		fmt.Println()
			return
		}

		var firstOff uint64
		if info.IsExtended {
			firstOff = binary.LittleEndian.Uint64(data[1:])
		} else {
			firstOff = uint64(binary.LittleEndian.Uint32(data[1:]))
		}

		numOffsets := int(firstOff) / offsetSize
		if numOffsets < 2 {
			fmt.Printf("  offset table:      invalid (first offset=%d, numOffsets=%d)\n\n", firstOff, numOffsets)
			return
		}

		fmt.Printf("  offset table:       %d entries (%d bytes each)\n", numOffsets, offsetSize)
		maxOffsets := 8
		if numOffsets > maxOffsets {
			fmt.Printf("  (showing first %d of %d offsets)\n", maxOffsets, numOffsets)
		}
		for j := 0; j < numOffsets && j < maxOffsets; j++ {
			var offVal uint64
			pos := 1 + j*offsetSize
			if info.IsExtended {
				offVal = binary.LittleEndian.Uint64(data[pos:])
			} else {
				offVal = uint64(binary.LittleEndian.Uint32(data[pos:]))
			}
			blobRelative := offVal - firstOff
			fmt.Printf("    offset[%d] = %d (blob-relative: %d)\n", j, offVal, blobRelative)
		}

		numBlobs := numOffsets - 1
		fmt.Printf("  blobs:              %d\n", numBlobs)
		maxBlobs := 5
		if numBlobs > maxBlobs {
			fmt.Printf("  (showing first %d of %d blobs)\n", maxBlobs, numBlobs)
		}
		for j := 0; j < numBlobs && j < maxBlobs; j++ {
			var start, end uint64
			pos := 1 + j*offsetSize
			if info.IsExtended {
				start = binary.LittleEndian.Uint64(data[pos:]) - firstOff
				end = binary.LittleEndian.Uint64(data[pos+offsetSize:]) - firstOff
			} else {
				start = uint64(binary.LittleEndian.Uint32(data[pos:])) - firstOff
				end = uint64(binary.LittleEndian.Uint32(data[pos+offsetSize:])) - firstOff
			}
			size := end - start
			blobStart := 1 + int(firstOff) + int(start)
			blobPreview := data[blobStart:minInt(blobStart+16, len(data))]
			contentType := detectContentType(blobPreview)
			fmt.Printf("    blob[%d]: %d bytes (offset %d-%d)   %s\n", j, size, start, end, contentType)
		}

		rawBlobStart := 1 + int(firstOff)
		rawPreview := data[rawBlobStart:minInt(rawBlobStart+32, len(data))]
		fmt.Printf("  raw blob data at:  0x%X (+%d)\n", offset+uint64(rawBlobStart), rawBlobStart)
		fmt.Printf("  first 32 bytes:    %s\n", hexDisplay(rawPreview))
	} else {
		compStart := 1
		compSize := len(data) - compStart
		fmt.Printf("  compressed data:    %d bytes at offset +%d\n", compSize, compStart)
		fmt.Printf("  compressed head:    %s\n", hexDisplay(truncateBytes(data[compStart:], 32)))
		fmt.Printf("  (compressed — decompress the full cluster\n")
		fmt.Printf("   with ParseCluster for blob-by-blob inspection)\n")
	}
	fmt.Println()
}

func traceChecksum(a *zim.Archive, r zim.Reader) {
	fmt.Println("─── Section: Checksum")

	h := a.Header()

	if !h.HasChecksum() {
		fmt.Println("  (no checksum in this archive)")
		fmt.Println()
		return
	}

	stored := readRaw(r, int64(h.ChecksumPos), 16)
	fmt.Printf("  location: 0x%016X\n", h.ChecksumPos)
	fmt.Printf("  raw bytes: %s\n", hexDisplay(stored))

	computed, err := zim.ComputeChecksum(r, h.ChecksumPos)
	if err != nil {
		fmt.Printf("  compute error: %v\n", err)
	} else if computed != nil {
		fmt.Printf("  stored:      %s\n", hex.EncodeToString(stored))
		fmt.Printf("  computed:    %s\n", hex.EncodeToString(computed))
		if hex.EncodeToString(stored) == hex.EncodeToString(computed) {
			fmt.Println("  match:       ✓ (OK)")
		} else {
			fmt.Println("  match:       ✗ (MISMATCH)")
		}
	}
	fmt.Println()
}

func traceIntegrity(filename string) {
	fmt.Println("─── Section: Integrity")

	report, err := zim.CheckFile(filename, zim.CheckAll)
	if err != nil {
		fmt.Printf("  error: %v\n\n", err)
		return
	}

	checkMark := func(ok bool) string {
		if ok {
			return "✓"
		}
		return "✗"
	}

	fmt.Printf("  checksum:          %s\n", checkMark(report.ChecksumOK))
	fmt.Printf("  broken links:       %d\n", len(report.BrokenLinks))
	fmt.Printf("  redirect loops:     %d\n", len(report.RedirectLoops))
	fmt.Printf("  redundant pairs:    %d\n", len(report.Redundant))
	fmt.Printf("  other errors:       %d\n", len(report.Errors))

	if len(report.BrokenLinks) > 0 {
		maxShow := 10
		fmt.Printf("  broken link detail (first %d):\n", minInt(maxShow, len(report.BrokenLinks)))
		for i, link := range report.BrokenLinks {
			if i >= maxShow {
				break
			}
			fmt.Printf("    %s → %s (%s)\n", link.SourceEntry, link.TargetLink, link.Reason)
		}
	}

	if len(report.RedirectLoops) > 0 {
		fmt.Printf("  redirect loops:\n")
		for i, loop := range report.RedirectLoops {
			fmt.Printf("    loop %d: entries %v\n", i, loop)
		}
	}

	if len(report.Errors) > 0 {
		maxShow := 10
		fmt.Printf("  error details (first %d):\n", minInt(maxShow, len(report.Errors)))
		for i, errMsg := range report.Errors {
			if i >= maxShow {
				break
			}
			fmt.Printf("    %s\n", errMsg)
		}
	}

	fmt.Println()
}

func readRaw(r zim.Reader, offset int64, size int) []byte {
	if size <= 0 || offset < 0 {
		return nil
	}
	buf := make([]byte, size)
	n, _ := r.ReadAt(buf, offset)
	return buf[:n]
}

func hexDisplay(data []byte) string {
	var parts []string
	for _, b := range data {
		parts = append(parts, fmt.Sprintf("%02X", b))
	}
	return strings.Join(parts, " ")
}

func truncateBytes(data []byte, maxLen int) []byte {
	if len(data) <= maxLen {
		return data
	}
	return data[:maxLen]
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func compressionName(c zim.Compression) string {
	switch c {
	case zim.CompressionNone:
		return "none"
	case zim.CompressionZstd:
		return "zstd"
	case zim.CompressionLzma:
		return "lzma/xz"
	default:
		return fmt.Sprintf("unknown(%d)", c)
	}
}

func describeNamespace(ns zim.Namespace) string {
	switch ns {
	case zim.NamespaceContent:
		return "content (new scheme)"
	case zim.NamespaceArticle:
		return "article (old scheme)"
	case zim.NamespaceImage:
		return "image (old scheme)"
	case zim.NamespaceScript:
		return "script (old scheme)"
	case zim.NamespaceLayout:
		return "layout (old scheme)"
	case zim.NamespaceMetadata:
		return "metadata"
	case zim.NamespaceIndex:
		return "index"
	case 'W':
		return "listing/redirect"
	default:
		if ns >= 'A' && ns <= 'Z' {
			return fmt.Sprintf("namespace %c", ns)
		}
		return fmt.Sprintf("unknown (0x%02X)", byte(ns))
	}
}

func detectContentType(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	switch {
	case data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G':
		return "[PNG]"
	case data[0] == 0xFF && data[1] == 0xD8:
		return "[JPEG]"
	case data[0] == 'G' && data[1] == 'I' && data[2] == 'F':
		return "[GIF]"
	case len(data) >= 9 && string(data[:9]) == "<!DOCTYPE":
		return "[HTML]"
	case data[0] == '<' && len(data) >= 5 && (data[1] == 'h' || data[1] == 'H'):
		return "[HTML]"
	case data[0] == 0x1F && data[1] == 0x8B:
		return "[gzip]"
	case data[0] == 0x28 && data[1] == 0xB5 && data[2] == 0x2F && data[3] == 0xFD:
		return "[zstd]"
	default:
		return fmt.Sprintf("[%02X %02X %02X %02X]", data[0], data[1], data[2], data[3])
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func detectMimeType(filename string) string {
	ext := strings.ToLower(filename[strings.LastIndex(filename, ".")+1:])
	switch ext {
	case "html", "htm":
		return "text/html"
	case "css":
		return "text/css"
	case "js":
		return "application/javascript"
	case "json":
		return "application/json"
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "svg":
		return "image/svg+xml"
	case "xml":
		return "application/xml"
	case "pdf":
		return "application/pdf"
	case "txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}
