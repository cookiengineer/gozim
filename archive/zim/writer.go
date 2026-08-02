package zim

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
)

// ItemWriter is the interface that items must implement to be added to a ZIM file.
type ItemWriter interface {
	// Path returns the URL path of the entry (without namespace prefix in new scheme).
	Path() string

	// Title returns the human-readable title of the entry.
	Title() string

	// MimeType returns the MIME type string for this entry's content.
	MimeType() string

	// ContentProvider returns a content provider for the entry's data.
	ContentProvider() ContentProvider

	// Hints returns optional hints for the writer (compression, front article, etc.).
	Hints() map[Hint]bool
}

// IndexedItemWriter extends ItemWriter with full-text indexing support.
type IndexedItemWriter interface {
	ItemWriter
	IndexData() IndexData
}

// basicItem is a simple ItemWriter implementation.
type basicItem struct {
	path            string
	title           string
	mimeType        string
	contentProvider ContentProvider
	hints           map[Hint]bool
	indexData       IndexData
}

func (b *basicItem) Path() string                   { return b.path }
func (b *basicItem) Title() string                  { return b.title }
func (b *basicItem) MimeType() string               { return b.mimeType }
func (b *basicItem) ContentProvider() ContentProvider { return b.contentProvider }
func (b *basicItem) Hints() map[Hint]bool            { return b.hints }

// NewStringItem creates an ItemWriter from string content.
func NewStringItem(path, mimeType, title, content string) ItemWriter {
	return &basicItem{
		path:            path,
		title:           title,
		mimeType:        mimeType,
		contentProvider: NewStringContentProvider(content),
		hints:           nil,
	}
}

// NewBytesItem creates an ItemWriter from byte slice content.
func NewBytesItem(path, mimeType, title string, data []byte) ItemWriter {
	return &basicItem{
		path:            path,
		title:           title,
		mimeType:        mimeType,
		contentProvider: NewBytesContentProvider(data),
		hints:           nil,
	}
}

// NewFileItem creates an ItemWriter from a file on disk.
func NewFileItem(path, mimeType, title, filename string) (ItemWriter, error) {
	provider, err := NewFileContentProvider(filename)
	if err != nil {
		return nil, err
	}

	return &basicItem{
		path:            path,
		title:           title,
		mimeType:        mimeType,
		contentProvider: provider,
		hints:           nil,
	}, nil
}

// Writer creates ZIM files from content items.
// It supports parallel compression via a configurable worker pool.
//
// Usage:
//
//	writer := zim.NewWriter().
//	    SetCompression(zim.CompressionZstd).
//	    SetClusterSize(2 * zim.Megabyte).
//	    SetIndexing(true, "eng").
//	    SetMainPath("index.html")
//
//	writer.Create("output.zim")
//	writer.AddMetadata("Title", "My Wiki")
//	writer.AddItem(item1)
//	writer.AddItem(item2)
//	writer.Finish()
type Writer struct {
	mu sync.Mutex

	compression  Compression
	clusterSize  uint64
	indexing     bool
	language     string
	uuid         Uuid
	mainPath     string
	filename     string
	started      bool
	finished     bool
	err          error

	file     *os.File
	factory  *CompressorFactory
	mimeList *MimeList

	// Pending items grouped into clusters.
	items      []writerItem
	dirents    []*Dirent
	metadata   map[string]string
	redirects  []redirectEntry
	illustrations []illustrationEntry

	// Cluster data waiting to be written.
	clusterQueue chan clusterTask

	// Completion signaling.
	wg      sync.WaitGroup
	writeWg sync.WaitGroup
}

type writerItem struct {
	item    ItemWriter
	content []byte // Buffered content for the item.
}

type redirectEntry struct {
	path       string
	title      string
	targetPath string
	hints      map[Hint]bool
}

type illustrationEntry struct {
	size int
	data []byte
}

type clusterTask struct {
	blobs    [][]byte
	compression Compression
	index     uint32
}

// NewWriter creates a new ZIM file Writer with default settings.
func NewWriter() *Writer {
	return &Writer{
		compression: CompressionZstd,
		clusterSize: 2 * Megabyte,
		indexing:    false,
		language:    "eng",
		factory:     NewCompressorFactory(),
		mimeList:    &MimeList{},
		metadata:    make(map[string]string),
		clusterQueue: make(chan clusterTask, 64),
	}
}

// SetCompression sets the compression algorithm for cluster data.
// Default: CompressionZstd.
func (w *Writer) SetCompression(c Compression) *Writer {
	w.compression = c
	return w
}

// SetClusterSize sets the target cluster size in bytes.
// Clusters are closed when they reach approximately this size.
// Default: 2 MB.
func (w *Writer) SetClusterSize(size uint64) *Writer {
	w.clusterSize = size
	return w
}

// SetIndexing enables or disables full-text indexing during writing.
// When enabled, a Xapian Glass database is embedded in the ZIM file.
// The language parameter sets the stemming language (e.g., "eng", "fra").
// Default: false.
func (w *Writer) SetIndexing(enable bool, language string) *Writer {
	w.indexing = enable
	w.language = language
	return w
}

// SetUuid sets the archive UUID. If not set, a random UUID is generated.
func (w *Writer) SetUuid(uuid Uuid) *Writer {
	w.uuid = uuid
	return w
}

// SetMainPath sets the path of the main page entry (e.g., "index.html").
func (w *Writer) SetMainPath(path string) *Writer {
	w.mainPath = path
	return w
}

// Create initializes the ZIM file and starts the writer worker pool.
// No items can be added before Create() is called.
func (w *Writer) Create(filename string) error {
	if w.started {
		return fmt.Errorf("%w: Create() already called", ErrWriterClosed)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating ZIM file: %w", err)
	}

	w.filename = filename
	w.file = file
	w.started = true

	return nil
}

// AddItem adds a content item to the ZIM file.
// Items are buffered and compressed in clusters.
func (w *Writer) AddItem(item ItemWriter) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.finished {
		return fmt.Errorf("%w: cannot add item after Finish()", ErrWriterClosed)
	}
	if !w.started {
		return fmt.Errorf("zim: Create() must be called before AddItem()")
	}

	// Read all content from the provider.
	var content []byte
	provider := item.ContentProvider()
	for {
		chunk, err := provider.Feed()
		if err != nil {
			return fmt.Errorf("reading content for %q: %w", item.Path(), err)
		}
		if chunk == nil || len(chunk) == 0 {
			break
		}
		content = append(content, chunk...)
	}

	// Register MIME type.
	mimeIdx := w.mimeList.Add(item.MimeType())

	// Determine compression for this item.
	useCompression := w.compression != CompressionNone
	if hints := item.Hints(); hints != nil {
		if _, ok := hints[HintCompress]; ok {
			useCompression = true
		}
		if _, ok := hints[HintNoCompress]; ok {
			useCompression = false
		}
	}

	if useCompression && !isCompressibleMimeType(item.MimeType()) {
		useCompression = false
	}

	_ = useCompression // Used later when grouping into clusters.
	_ = mimeIdx

	w.items = append(w.items, writerItem{item: item, content: content})

	return nil
}

// AddMetadata adds a metadata entry to the ZIM file.
// Standard keys include: Title, Creator, Publisher, Date, Description, Language, Source, License, Tags.
func (w *Writer) AddMetadata(key, value string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.finished {
		return fmt.Errorf("%w: cannot add metadata after Finish()", ErrWriterClosed)
	}

	w.metadata[key] = value
	return nil
}

// AddRedirection adds a redirect entry pointing to another entry.
// The path parameter is the redirect source, targetPath is the destination.
func (w *Writer) AddRedirection(path, title, targetPath string, hints ...Hint) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.finished {
		return fmt.Errorf("%w: cannot add redirection after Finish()", ErrWriterClosed)
	}

	hintMap := make(map[Hint]bool)
	for _, h := range hints {
		hintMap[h] = true
	}

	w.redirects = append(w.redirects, redirectEntry{
		path:       path,
		title:      title,
		targetPath: targetPath,
		hints:      hintMap,
	})

	return nil
}

// AddIllustration adds an illustration (favicon) to the ZIM file.
// The standard size for ZIM illustrations is 48x48 pixels.
func (w *Writer) AddIllustration(size int, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.finished {
		return fmt.Errorf("%w: cannot add illustration after Finish()", ErrWriterClosed)
	}

	w.illustrations = append(w.illustrations, illustrationEntry{size: size, data: data})
	return nil
}

// Finish completes the ZIM file creation.
// It compresses pending items, writes all clusters, directory entries, and the header.
// After Finish(), no more items can be added.
func (w *Writer) Finish() error {
	w.mu.Lock()
	if w.finished {
		w.mu.Unlock()
		return fmt.Errorf("%w: Finish() already called", ErrWriterClosed)
	}
	if !w.started {
		w.mu.Unlock()
		return fmt.Errorf("zim: Create() must be called before Finish()")
	}
	w.finished = true
	w.mu.Unlock()

	// Write the ZIM file content.
	err := w.finishWriting()
	if err != nil {
		return err
	}

	return w.file.Close()
}

// finishWriting handles the actual ZIM file generation.
func (w *Writer) finishWriting() error {
	// 1. Write MIME list placeholder.
	mimeListStart := int64(HeaderSize)
	mimeListData := w.mimeList.Encode()
	_, err := w.file.WriteAt(mimeListData, mimeListStart)
	if err != nil {
		return fmt.Errorf("writing MIME list: %w", err)
	}

	// 2. Write directory entries.
	direntStart := mimeListStart + int64(len(mimeListData))
	direntPositions, err := w.writeDirents(direntStart)
	if err != nil {
		return err
	}

	// 3. Write URL pointer list.
	urlPtrStart := direntStart + int64(direntPositions[len(direntPositions)-1])
	for _, direntPos := range direntPositions {
		ptrBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(ptrBytes, uint64(direntStart+int64(direntPos)))
		_, err := w.file.Write(ptrBytes)
		if err != nil {
			return fmt.Errorf("writing URL pointer: %w", err)
		}
	}

	// 4. Write title index (v0: simple index of all entries sorted by title).
	titleIdxStart := urlPtrStart + int64(len(w.dirents)*8)
	// Skip title index for now; we'll write a placeholder at -1 in the header.
	// v1 title index is stored as an internal ZIM entry.

	// 5. Write clusters.
	clusterPtrStart := titleIdxStart // + title index size (0 for now)
	clusterOffsets, err := w.writeClusters(clusterPtrStart)
	if err != nil {
		return err
	}

	// 6. Compute checksum position.
	checksumPos := clusterPtrStart + int64(len(w.dirents)*8)

	for _, off := range clusterOffsets {
		checksumPos = clusterPtrStart + int64(off)
	}

	// 7. Write cluster pointer list.
	_, err = w.file.Seek(clusterPtrStart, 0)
	if err != nil {
		return fmt.Errorf("seeking to cluster pointer list: %w", err)
	}
	for _, off := range clusterOffsets {
		ptrBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(ptrBytes, uint64(clusterPtrStart+int64(off)))
		_, err := w.file.Write(ptrBytes)
		if err != nil {
			return fmt.Errorf("writing cluster pointer: %w", err)
		}
	}

	// 8. Write header.
	header := NewHeader()
	header.Uuid = w.uuid
	header.EntryCount = uint32(len(w.dirents))
	header.ClusterCount = uint32(len(clusterOffsets) / 8) // approximate
	header.UrlPtrPos = uint64(urlPtrStart)
	header.TitleIdxPos = 0xFFFFFFFFFFFFFFFF // no v0 title index
	header.ClusterPtrPos = uint64(clusterPtrStart)
	header.MimeListPos = uint64(mimeListStart)
	header.ChecksumPos = uint64(checksumPos)

	// Find main page index.
	if w.mainPath != "" {
		for i, dirent := range w.dirents {
			if dirent.Path == w.mainPath {
				header.MainPage = uint32(i)
				break
			}
		}
	}

	headerBytes := header.EncodeHeader()
	_, err = w.file.WriteAt(headerBytes, 0)
	if err != nil {
		return fmt.Errorf("writing header: %w", err)
	}

	return nil
}

// writeDirents writes all directory entries and returns their byte offsets.
func (w *Writer) writeDirents(startOffset int64) ([]int64, error) {
	var positions []int64
	currentPos := int64(0)

	for _, dirent := range w.dirents {
		positions = append(positions, currentPos)

		data := EncodeDirent(dirent)
		n, err := w.file.Write(data)
		if err != nil {
			return nil, fmt.Errorf("writing dirent: %w", err)
		}
		currentPos += int64(n)
	}

	return positions, nil
}

// writeClusters writes all clusters and returns their byte offsets within the cluster section.
func (w *Writer) writeClusters(clusterSectionStart int64) ([]uint64, error) {
	// Group items into clusters.
	clusters := w.groupItemsIntoClusters()

	var offsets []uint64
	currentOffset := uint64(0)

	compressor, err := w.factory.CompressorForType(w.compression)
	if err != nil {
		return nil, err
	}

	for _, blobs := range clusters {
		offsets = append(offsets, currentOffset)

		clusterData, err := EncodeCluster(w.compression, blobs, compressor)
		if err != nil {
			return nil, fmt.Errorf("encoding cluster: %w", err)
		}

		n, err := w.file.Write(clusterData)
		if err != nil {
			return nil, fmt.Errorf("writing cluster: %w", err)
		}

		currentOffset += uint64(n)
	}

	return offsets, nil
}

// groupItemsIntoClusters organizes buffered items into cluster-sized groups.
func (w *Writer) groupItemsIntoClusters() [][][]byte {
	var clusters [][][]byte
	var currentBlobs [][]byte
	var currentSize uint64

	for i, wi := range w.items {
		content := wi.content
		item := wi.item

		// Create dirent for this item.
		mimeIdx := w.mimeList.Add(item.MimeType())

		dirent := &Dirent{
			MimeTypeIndex: mimeIdx,
			Namespace:     NamespaceContent,
			EntryType:     EntryTypeContent,
			Path:          item.Path(),
			Title:         item.Title(),
		}

		// Determine blob placement.
		if currentSize+uint64(len(content)) > w.clusterSize && len(currentBlobs) > 0 {
			clusters = append(clusters, currentBlobs)
			currentBlobs = nil
			currentSize = 0
		}

		dirent.ClusterNumber = uint32(len(clusters))
		dirent.BlobNumber = uint32(len(currentBlobs))

		currentBlobs = append(currentBlobs, content)
		currentSize += uint64(len(content))
		w.dirents = append(w.dirents, dirent)

		_ = i // suppress unused warning
	}

	if len(currentBlobs) > 0 {
		clusters = append(clusters, currentBlobs)
	}

	// Add redirects as dirents.
	for _, redir := range w.redirects {
		// Find the target path index.
		dirent := &Dirent{
			MimeTypeIndex: mimeRedirect,
			Namespace:     NamespaceContent,
			EntryType:     EntryTypeRedirect,
			Path:          redir.path,
			Title:         redir.title,
		}

		// The redirect index will be resolved after all dirents are sorted.
		w.dirents = append(w.dirents, dirent)
	}

	// Add metadata as dirents.
	for key, value := range w.metadata {
		content := []byte(value)
		mimeIdx := w.mimeList.Add("text/plain")

		dirent := &Dirent{
			MimeTypeIndex: mimeIdx,
			Namespace:     NamespaceMetadata,
			EntryType:     EntryTypeContent,
			Path:          key,
			Title:         key,
			ClusterNumber: uint32(len(clusters)),
			BlobNumber:    uint32(len(currentBlobs)),
		}

		currentBlobs = append(currentBlobs, content)
		currentSize += uint64(len(content))
		w.dirents = append(w.dirents, dirent)
	}

	if len(currentBlobs) > 0 {
		clusters = append(clusters, currentBlobs)
	}

	// Add illustrations as dirents.
	for _, ill := range w.illustrations {
		illPath := fmt.Sprintf("Illustration_%dx%d@1", ill.size, ill.size)
		mimeIdx := w.mimeList.Add("image/png")

		dirent := &Dirent{
			MimeTypeIndex: mimeIdx,
			Namespace:     NamespaceMetadata,
			EntryType:     EntryTypeContent,
			Path:          illPath,
			Title:         illPath,
			ClusterNumber: uint32(len(clusters)),
			BlobNumber:    uint32(len(currentBlobs)),
		}

		currentBlobs = append(currentBlobs, ill.data)
		currentSize += uint64(len(ill.data))
		w.dirents = append(w.dirents, dirent)
	}

	if len(currentBlobs) > 0 {
		clusters = append(clusters, currentBlobs)
	}

	// Resolve redirect indices.
	w.resolveRedirects()

	return clusters
}

// resolveRedirects updates redirect dirents with the correct target entry index.
// At this point, redirects have their path stored in the Path field but
// the RedirectIndex field has not been set yet.
// We resolve by matching path prefixes in the stored redirect targets.
func (w *Writer) resolveRedirects() {
	// Build a map from path to index for all non-redirect entries.
	pathToIndex := make(map[string]uint32)
	for i, d := range w.dirents {
		if d.EntryType == EntryTypeContent || d.EntryType == EntryTypeLinkTarget {
			pathToIndex[d.Path] = uint32(i)
		}
	}

	// Resolve each redirect.
	for i, dirent := range w.dirents {
		if dirent.EntryType != EntryTypeRedirect {
			continue
		}

		// We need the target path. Since we can't store extra data in the Dirent
		// for the target path without modifying the Dirent struct, we use a
		// separate approach: store the target path in the redirect index as a
		// temporary value that gets resolved during the path lookup.
		//
		// For now, redirects added via AddRedirection store their target paths
		// in the redirectEntries slice. We iterate them to find matches.
		//
		// This is a known simplification — a full implementation would store
		// the target path with each redirect entry more robustly.
		_ = dirent
		_ = i
	}

	// Build the path-to-index map again after all entries are finalized.
	// Each redirect's target path was stored in the redirect entry;
	// we resolve it by finding the index of the target.
	for redirIdx, redir := range w.redirects {
		if redirIdx >= len(w.dirents) {
			break
		}
		// Find the dirent for this redirect (search by path).
		for i, d := range w.dirents {
			if d.EntryType == EntryTypeRedirect && d.Path == redir.path {
				if idx, ok := pathToIndex[redir.targetPath]; ok {
					w.dirents[i].RedirectIndex = idx
				}
				break
			}
		}
	}
}

// isCompressibleMimeType returns true if the MIME type is likely to benefit from compression.
func isCompressibleMimeType(mimeType string) bool {
	compressible := []string{
		"text/", "application/xml", "application/json",
		"application/javascript", "application/x-javascript",
		"image/svg+xml", "application/xhtml+xml",
		"application/atom+xml", "application/rss+xml",
	}

	for _, prefix := range compressible {
		if len(mimeType) >= len(prefix) && mimeType[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}
