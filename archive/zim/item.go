package zim

import "fmt"

// Item represents the content of a ZIM entry.
// It provides access to the MIME type, data size, and the actual content bytes.
type Item struct {
	archive    *Archive
	clusterNum uint32
	blobNum    uint32
	mimeType   string
	dataSize   uint64
}

// MimeType returns the MIME type of this item's content.
func (i *Item) MimeType() string {
	return i.mimeType
}

// Size returns the total size of the item's data in bytes.
func (i *Item) Size() uint64 {
	return i.dataSize
}

// Data returns a byte range from the item's content.
// If offset+size exceeds the item size, it returns up to the end of the data.
func (i *Item) Data(offset, size uint64) ([]byte, error) {
	if offset > i.dataSize {
		return nil, fmt.Errorf("%w: offset %d exceeds item size %d", ErrFormat, offset, i.dataSize)
	}

	cluster, err := i.archive.getCluster(i.clusterNum)
	if err != nil {
		return nil, fmt.Errorf("reading item data: %w", err)
	}

	if size == 0 || offset+size > i.dataSize {
		size = i.dataSize - offset
	}

	return cluster.BlobRange(i.blobNum, offset, size)
}

// DataAll returns the entire content of the item as a byte slice.
func (i *Item) DataAll() ([]byte, error) {
	return i.Data(0, i.dataSize)
}

// readItem reads an Item from the archive using the directory entry.
func (a *Archive) readItem(dirent *Dirent) (*Item, error) {
	mimeType := a.mimeList.MimeType(dirent.MimeTypeIndex)

	cluster, err := a.getCluster(dirent.ClusterNumber)
	if err != nil {
		return nil, fmt.Errorf("reading item: %w", err)
	}

	blobSize := cluster.blobSize(dirent.BlobNumber)

	return &Item{
		archive:    a,
		clusterNum: dirent.ClusterNumber,
		blobNum:    dirent.BlobNumber,
		mimeType:   mimeType,
		dataSize:   blobSize,
	}, nil
}

// blobSize returns the uncompressed size of a blob in this cluster.
func (c *Cluster) blobSize(index uint32) uint64 {
	if int(index) >= c.BlobCount() {
		return 0
	}
	return c.blobOffsets[index+1] - c.blobOffsets[index]
}

// Blob is a read-only view of binary data within a ZIM file.
type Blob struct {
	data []byte
}

// Data returns the raw bytes of the blob.
func (b *Blob) Data() []byte {
	return b.data
}

// Size returns the size of the blob in bytes.
func (b *Blob) Size() uint64 {
	return uint64(len(b.data))
}

// NewBlob creates a Blob from a byte slice.
func NewBlob(data []byte) *Blob {
	return &Blob{data: data}
}
