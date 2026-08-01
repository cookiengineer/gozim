package zim

import (
	"fmt"
	"io"
	"math/rand"
	"sort"
	"sync"
)

// Archive represents an open ZIM file for reading.
// It is safe for concurrent use by multiple goroutines.
type Archive struct {
	mu       sync.RWMutex
	reader   Reader
	closer   io.Closer
	header   *Header
	mimeList *MimeList
	factory  *CompressorFactory

	// Parsed entry metadata.
	urlPtrs   []uint64     // Offsets to each dirent (sorted by namespace+path)
	dirents   []*Dirent    // Parsed directory entries
	titleIdx  []uint32     // Sorted indices into urlPtrs by title
	frontArticles []uint32 // Indices of front articles (for random selection)

	// Cluster cache.
	clusterCache *Cache[uint32, *Cluster]
}

// RawReader returns the underlying byte-level reader for debug tooling.
func (a *Archive) RawReader() Reader {
	return a.reader
}

// Header returns the parsed ZIM file header.
func (a *Archive) Header() *Header {
	return a.header
}

// MimeTypeList returns the parsed MIME type list.
func (a *Archive) MimeTypeList() *MimeList {
	return a.mimeList
}

// UrlPtrOffsets returns the raw URL pointer offsets for debug tooling.
func (a *Archive) UrlPtrOffsets() []uint64 {
	return a.urlPtrs
}

// Dirents returns all parsed directory entries for debug tooling.
func (a *Archive) Dirents() []*Dirent {
	return a.dirents
}

// TitleIndices returns the title index entries for debug tooling.
func (a *Archive) TitleIndices() []uint32 {
	return a.titleIdx
}

// CompressionFactory returns the compressor/decompressor factory.
func (a *Archive) CompressionFactory() *CompressorFactory {
	return a.factory
}

// Open opens a ZIM file for reading.
// It automatically detects and supports multipart ZIM files.
func Open(filename string) (*Archive, error) {
	reader, closer, err := OpenMultipartReader(filename)
	if err != nil {
		return nil, err
	}

	return openArchive(reader, closer)
}

// openArchive reads the header and index structures from an opened Reader.
func openArchive(reader Reader, closer io.Closer) (*Archive, error) {
	headerData := make([]byte, HeaderSize)
	if _, err := reader.ReadAt(headerData, 0); err != nil {
		return nil, fmt.Errorf("%w: reading header: %v", ErrFormat, err)
	}

	header, err := ParseHeader(headerData)
	if err != nil {
		return nil, err
	}

	if err := header.Validate(reader.Size()); err != nil {
		return nil, err
	}

	a := &Archive{
		reader:   reader,
		closer:   closer,
		header:   header,
		factory:  NewCompressorFactory(),
	}

	// Parse MIME type list.
	if err := a.loadMimeList(); err != nil {
		return nil, err
	}

	// Parse directory entries.
	if err := a.loadDirents(); err != nil {
		return nil, err
	}

	// Load title index if available.
	if header.HasTitleIndex() {
		a.loadTitleIndex()
	}

	// Initialize cluster cache (max 64MB of decompressed cluster data).
	a.clusterCache = NewCache[uint32, *Cluster](64 * Megabyte)

	return a, nil
}

// Close releases all resources associated with the archive.
func (a *Archive) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.clusterCache.Clear()
	if a.closer != nil {
		return a.closer.Close()
	}
	return nil
}

// loadMimeList reads and parses the MIME type list from the archive.
func (a *Archive) loadMimeList() error {
	// MIME list starts at mimeListPos (typically 80) up to urlPtrPos.
	endPos := a.header.UrlPtrPos
	if endPos <= a.header.MimeListPos {
		return fmt.Errorf("%w: invalid MIME list position", ErrFormat)
	}

	data := make([]byte, endPos-a.header.MimeListPos)
	if _, err := a.reader.ReadAt(data, int64(a.header.MimeListPos)); err != nil {
		return fmt.Errorf("%w: reading MIME list: %v", ErrFormat, err)
	}

	mimeList, err := ParseMimeList(data)
	if err != nil {
		return err
	}

	a.mimeList = mimeList
	return nil
}

// loadDirents reads all directory entries from the archive.
func (a *Archive) loadDirents() error {
	count := int(a.header.EntryCount)

	// Read URL pointer list.
	ptrSize := 8 * count
	ptrData := make([]byte, ptrSize)
	if _, err := a.reader.ReadAt(ptrData, int64(a.header.UrlPtrPos)); err != nil {
		return fmt.Errorf("%w: reading URL pointer list: %v", ErrFormat, err)
	}

	a.urlPtrs = make([]uint64, count)
	for i := 0; i < count; i++ {
		a.urlPtrs[i] = readUint64(ptrData[i*8:])
	}

	// Parse each directory entry.
	a.dirents = make([]*Dirent, count)
	for i := 0; i < count; i++ {
		// Read a generous chunk for the dirent (max 2KB per entry).
		maxDirentSize := 2048
		offset := int64(a.urlPtrs[i])
		remaining := a.reader.Size() - offset
		if remaining <= 0 {
			return fmt.Errorf("%w: dirent at offset %d is beyond file end", ErrFormat, offset)
		}

		readSize := maxDirentSize
		if int64(readSize) > remaining {
			readSize = int(remaining)
		}

		data := make([]byte, readSize)
		if _, err := a.reader.ReadAt(data, offset); err != nil {
			return fmt.Errorf("%w: reading dirent %d at offset %d: %v", ErrFormat, i, offset, err)
		}

		dirent, _, err := ParseDirent(data)
		if err != nil {
			return fmt.Errorf("%w: parsing dirent %d: %v", ErrFormat, i, err)
		}

		a.dirents[i] = dirent
	}

	return nil
}

// loadTitleIndex reads the title index (v0) from the archive.
func (a *Archive) loadTitleIndex() {
	count := int(a.header.EntryCount)
	a.titleIdx = make([]uint32, count)

	data := make([]byte, count*4)
	if _, err := a.reader.ReadAt(data, int64(a.header.TitleIdxPos)); err != nil {
		// If we can't read the title index, leave it empty.
		a.titleIdx = nil
		return
	}

	for i := 0; i < count; i++ {
		a.titleIdx[i] = readUint32(data[i*4:])
	}

	// Build front articles list.
	for _, idx := range a.titleIdx {
		if int(idx) < len(a.dirents) {
			ns := a.dirents[idx].Namespace
			if ns.IsUserContent() {
				a.frontArticles = append(a.frontArticles, idx)
			}
		}
	}
}

// loadTitleIndexV1 reads the v1 title index from X/listing/titleOrdered/v1.
func (a *Archive) loadTitleIndexV1() error {
	path := "X/listing/titleOrdered/v1"
	if !a.header.HasNewNamespaceScheme() {
		// Fall back to v0 index.
		return nil
	}

	// Find the v1 title index entry.
	entry, err := a.EntryByPath(path)
	if err != nil {
		return nil // Not an error if v1 index is missing; fall back to v0.
	}

	item, err := entry.Item(false)
	if err != nil {
		return err
	}

	data, err := item.DataAll()
	if err != nil {
		return err
	}
	if len(data)%4 != 0 {
		return fmt.Errorf("%w: invalid v1 title index size", ErrFormat)
	}

	count := len(data) / 4
	a.titleIdx = make([]uint32, count)
	a.frontArticles = make([]uint32, count)

	for i := 0; i < count; i++ {
		idx := readUint32(data[i*4:])
		a.titleIdx[i] = idx
		a.frontArticles[i] = idx
	}

	return nil
}

// EntryByPath looks up an entry by its URL path.
// For new namespace scheme, paths don't include the namespace prefix.
// For old namespace scheme, paths do include it (e.g., "A/index.html").
//
// The lookup accounts for the ZIM sort order: entries are sorted by
// namespace+path, not by path alone. Content entries (C/A) come first,
// then metadata (M), then indexes (X), etc.
func (a *Archive) EntryByPath(path string) (*Entry, error) {
	// For new namespace scheme without explicit prefix, try C namespace first.
	if a.header.HasNewNamespaceScheme() && len(path) > 0 && path[0] != 'A' && path[0] != 'I' && path[0] != 'J' && path[0] != '-' && path[0] != 'M' && path[0] != 'X' && path[0] != 'C' && path[0] != 'W' {
		return a.entryByNamespaceAndPath(NamespaceContent, path)
	}

	// For old namespace or paths with explicit prefix.
	if len(path) > 1 && path[1] == '/' {
		return a.entryByNamespaceAndPath(Namespace(path[0]), path[2:])
	}

	return nil, fmt.Errorf("%w: %q", ErrEntryNotFound, path)
}

// entryByNamespaceAndPath looks up an entry by namespace and path.
func (a *Archive) entryByNamespaceAndPath(ns Namespace, path string) (*Entry, error) {
	// Binary search using namespace+path as the sort key.
	// Dirents are sorted by namespace, then by path within each namespace.
	sortKey := string(byte(ns)) + "\x00" + path

	idx := sort.Search(len(a.dirents), func(i int) bool {
		d := a.dirents[i]
		dSortKey := string(byte(d.Namespace)) + "\x00" + d.Path
		return dSortKey >= sortKey
	})

	if idx < len(a.dirents) && a.dirents[idx].Namespace == ns && a.dirents[idx].Path == path {
		return &Entry{archive: a, dirent: a.dirents[idx], index: uint32(idx)}, nil
	}

	return nil, fmt.Errorf("%w: %q", ErrEntryNotFound, path)
}

// EntryByTitle looks up an entry by its title.
func (a *Archive) EntryByTitle(title string) (*Entry, error) {
	a.mu.RLock()
	titleIdx := a.titleIdx
	a.mu.RUnlock()

	if titleIdx == nil {
		// Try loading v1 title index.
		a.mu.Lock()
		err := a.loadTitleIndexV1()
		titleIdx = a.titleIdx
		a.mu.Unlock()
		if err != nil {
			return nil, fmt.Errorf("%w: %q (no title index)", ErrEntryNotFound, title)
		}
	}

	idx := sort.Search(len(titleIdx), func(i int) bool {
		entryIdx := titleIdx[i]
		if int(entryIdx) >= len(a.dirents) {
			return false
		}
		return a.dirents[entryIdx].Title >= title
	})

	if idx < len(titleIdx) {
		entryIdx := titleIdx[idx]
		if int(entryIdx) < len(a.dirents) && a.dirents[entryIdx].Title == title {
			return &Entry{archive: a, dirent: a.dirents[entryIdx], index: entryIdx}, nil
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrEntryNotFound, title)
}

// EntryByIndex returns the entry at the given URL pointer list index.
func (a *Archive) EntryByIndex(index uint32) (*Entry, error) {
	if int(index) >= len(a.dirents) {
		return nil, fmt.Errorf("%w: index %d out of range", ErrEntryNotFound, index)
	}

	return &Entry{archive: a, dirent: a.dirents[index], index: index}, nil
}

// HasEntry returns true if an entry with the given path exists in the archive.
func (a *Archive) HasEntry(path string) bool {
	_, err := a.EntryByPath(path)
	return err == nil
}

// MainEntry returns the main page entry of the archive.
func (a *Archive) MainEntry() (*Entry, error) {
	if !a.header.HasMainPage() {
		return nil, fmt.Errorf("%w: no main page defined", ErrEntryNotFound)
	}

	return a.EntryByIndex(a.header.MainPage)
}

// RandomEntry returns a randomly selected front article.
func (a *Archive) RandomEntry() (*Entry, error) {
	if len(a.frontArticles) == 0 {
		// Try loading v1 index which populates frontArticles.
		a.mu.Lock()
		a.loadTitleIndexV1()
		frontArticles := a.frontArticles
		a.mu.Unlock()

		if len(frontArticles) == 0 {
			return nil, fmt.Errorf("%w: no front articles", ErrEntryNotFound)
		}
	}

	idx := a.frontArticles[rand.Intn(len(a.frontArticles))]
	return a.EntryByIndex(idx)
}

// Uuid returns the 16-byte UUID of the archive.
func (a *Archive) Uuid() Uuid {
	return a.header.Uuid
}

// EntryCount returns the total number of entries in the archive.
func (a *Archive) EntryCount() uint32 {
	return a.header.EntryCount
}

// ArticleCount returns the number of front articles in the archive.
func (a *Archive) ArticleCount() uint32 {
	return uint32(len(a.frontArticles))
}

// MediaCount returns the number of non-article entries.
func (a *Archive) MediaCount() uint32 {
	return a.header.EntryCount - a.ArticleCount()
}

// ClusterCount returns the number of clusters in the archive.
func (a *Archive) ClusterCount() uint32 {
	return a.header.ClusterCount
}

// Checksum returns the MD5 checksum of the archive as a hex string.
func (a *Archive) Checksum() string {
	if !a.header.HasChecksum() {
		return ""
	}

	checksumData := make([]byte, ChecksumSize)
	if _, err := a.reader.ReadAt(checksumData, int64(a.header.ChecksumPos)); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", checksumData)
}

// Metadata returns the value of a metadata key, if present.
func (a *Archive) Metadata(key string) (string, bool) {
	entry, err := a.entryByNamespaceAndPath(NamespaceMetadata, key)
	if err != nil {
		return "", false
	}
	item, err := entry.Item(false)
	if err != nil {
		return "", false
	}
	data, err := item.DataAll()
	if err != nil {
		return "", false
	}
	return string(data), true
}

// MetadataKeys returns all available metadata key names.
func (a *Archive) MetadataKeys() []string {
	var keys []string
	for _, dirent := range a.dirents {
		if dirent.Namespace.IsMetadata() {
			keys = append(keys, dirent.Path)
		}
	}
	return keys
}

// Illustration returns the illustration (favicon) item for the given size, if present.
func (a *Archive) Illustration(size int) (*Item, bool) {
	var path string
	if a.header.HasNewNamespaceScheme() {
		path = fmt.Sprintf("M/Illustration_%dx%d@1", size, size)
	} else {
		path = fmt.Sprintf("M/Illustration_%dx%d@1", size, size)
	}

	entry, err := a.EntryByPath(path)
	if err != nil {
		return nil, false
	}
	item, err := entry.Item(false)
	if err != nil {
		return nil, false
	}
	return item, true
}

// HasFulltextIndex returns true if the archive contains an embedded
// Xapian full-text search database.
func (a *Archive) HasFulltextIndex() bool {
	return a.HasEntry("X/fulltext/xapian")
}

// HasTitleIndex returns true if a title index is available.
func (a *Archive) HasTitleIndex() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.titleIdx != nil
}

// HasChecksum returns true if the archive contains an integrity checksum.
func (a *Archive) HasChecksum() bool {
	return a.header.HasChecksum()
}

// HasNewNamespaceScheme returns true if the archive uses the new namespace scheme.
func (a *Archive) HasNewNamespaceScheme() bool {
	return a.header.HasNewNamespaceScheme()
}

// IsMultiPart returns true if the archive spans multiple physical files.
func (a *Archive) IsMultiPart() bool {
	_, ok := a.reader.(*MultipartReader)
	return ok
}

// getCluster retrieves a cluster from the cache or reads and decompresses it.
func (a *Archive) getCluster(clusterNum uint32) (*Cluster, error) {
	// Check cache first.
	if cluster, ok := a.clusterCache.Get(clusterNum); ok {
		return cluster, nil
	}

	// Read cluster from file.
	if int(clusterNum) >= int(a.header.ClusterCount) {
		return nil, fmt.Errorf("%w: cluster index %d out of range (0-%d)", ErrFormat, clusterNum, a.header.ClusterCount-1)
	}

	// Read cluster pointer.
	ptrOffset := int64(a.header.ClusterPtrPos) + int64(clusterNum)*8
	ptrData := make([]byte, 8)
	if _, err := a.reader.ReadAt(ptrData, ptrOffset); err != nil {
		return nil, fmt.Errorf("reading cluster pointer %d: %w", clusterNum, err)
	}
	clusterOffset := readUint64(ptrData)

	// Determine cluster size (next cluster offset or checksum position).
	var clusterSize uint64
	if int(clusterNum)+1 < int(a.header.ClusterCount) {
		nextPtrData := make([]byte, 8)
		if _, err := a.reader.ReadAt(nextPtrData, ptrOffset+8); err != nil {
			return nil, fmt.Errorf("reading next cluster pointer: %w", err)
		}
		nextOffset := readUint64(nextPtrData)
		clusterSize = nextOffset - clusterOffset
	} else {
		// Last cluster: size up to the checksum or end of file.
		if a.header.HasChecksum() {
			clusterSize = a.header.ChecksumPos - clusterOffset
		} else {
			clusterSize = uint64(a.reader.Size()) - clusterOffset
		}
	}

	// Read cluster data.
	if clusterSize > 256*Megabyte {
		return nil, fmt.Errorf("%w: cluster %d too large (%d bytes)", ErrFormat, clusterNum, clusterSize)
	}

	clusterData := make([]byte, clusterSize)
	if _, err := a.reader.ReadAt(clusterData, int64(clusterOffset)); err != nil {
		return nil, fmt.Errorf("reading cluster %d data: %w", clusterNum, err)
	}

	// Parse and decompress cluster.
	cluster, err := ParseCluster(clusterData, a.factory)
	if err != nil {
		return nil, fmt.Errorf("parsing cluster %d: %w", clusterNum, err)
	}

	// Cache the decompressed cluster.
	a.clusterCache.Set(clusterNum, cluster, int64(len(cluster.blobData)))

	return cluster, nil
}

// iterateDirents sends all non-deleted directory entries over a channel.
func (a *Archive) iterateDirents(filter func(*Dirent) bool) <-chan *Entry {
	ch := make(chan *Entry)

	go func() {
		defer close(ch)
		for i, dirent := range a.dirents {
			if dirent.IsDeleted() {
				continue
			}
			if filter != nil && !filter(dirent) {
				continue
			}
			ch <- &Entry{archive: a, dirent: dirent, index: uint32(i)}
		}
	}()

	return ch
}

// IterateByPath returns a channel that yields entries sorted by namespace+path.
func (a *Archive) IterateByPath() <-chan *Entry {
	return a.iterateDirents(nil)
}

// IterateByTitle returns a channel that yields entries sorted by title.
func (a *Archive) IterateByTitle() <-chan *Entry {
	titleIdx := a.titleIdx
	if titleIdx == nil {
		a.mu.Lock()
		a.loadTitleIndexV1()
		titleIdx = a.titleIdx
		a.mu.Unlock()
	}

	ch := make(chan *Entry)

	go func() {
		defer close(ch)
		for _, idx := range titleIdx {
			if int(idx) >= len(a.dirents) {
				continue
			}
			dirent := a.dirents[idx]
			if dirent.IsDeleted() {
				continue
			}
			ch <- &Entry{archive: a, dirent: dirent, index: idx}
		}
	}()

	return ch
}

// IterateByClusterOrder returns a channel that yields entries in cluster order
// (optimal for sequential cluster access).
func (a *Archive) IterateByClusterOrder() <-chan *Entry {
	// Build an index sorted by (cluster, blob) pairs.
	type indexedDirent struct {
		idx        uint32
		clusterNum uint32
		blobNum    uint32
	}

	items := make([]indexedDirent, 0, len(a.dirents))
	for i, dirent := range a.dirents {
		if dirent.IsDeleted() || dirent.IsRedirect() {
			continue
		}
		items = append(items, indexedDirent{
			idx:        uint32(i),
			clusterNum: dirent.ClusterNumber,
			blobNum:    dirent.BlobNumber,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].clusterNum != items[j].clusterNum {
			return items[i].clusterNum < items[j].clusterNum
		}
		return items[i].blobNum < items[j].blobNum
	})

	ch := make(chan *Entry)

	go func() {
		defer close(ch)
		for _, item := range items {
			dirent := a.dirents[item.idx]
			ch <- &Entry{archive: a, dirent: dirent, index: item.idx}
		}
	}()

	return ch
}

// FindByPath returns a channel that yields all entries whose path starts with the given prefix.
func (a *Archive) FindByPath(prefix string) <-chan *Entry {
	return a.iterateDirents(func(d *Dirent) bool {
		return len(d.Path) >= len(prefix) && d.Path[:len(prefix)] == prefix
	})
}
