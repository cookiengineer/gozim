package zim

import (
	"io"
	"os"
)

// ContentProvider supplies the content data for a ZIM item during writing.
// It is called repeatedly via Feed() to produce content chunks.
// When Feed() returns a nil or empty slice with nil error, the content is complete.
type ContentProvider interface {
	// Size returns the total content size in bytes, or 0 if unknown.
	Size() uint64

	// Feed returns the next chunk of content data.
	// When no more data is available, Feed returns nil, nil.
	Feed() ([]byte, error)
}

// StringContentProvider provides content from an in-memory string.
type StringContentProvider struct {
	data   []byte
	offset int
}

// NewStringContentProvider creates a ContentProvider from a string.
func NewStringContentProvider(data string) *StringContentProvider {
	return &StringContentProvider{data: []byte(data)}
}

func (p *StringContentProvider) Size() uint64 {
	return uint64(len(p.data))
}

func (p *StringContentProvider) Feed() ([]byte, error) {
	if p.offset >= len(p.data) {
		return nil, nil
	}

	chunkSize := 65536
	end := p.offset + chunkSize
	if end > len(p.data) {
		end = len(p.data)
	}

	chunk := p.data[p.offset:end]
	p.offset = end

	result := make([]byte, len(chunk))
	copy(result, chunk)
	return result, nil
}

// FileContentProvider provides content from a file on disk.
type FileContentProvider struct {
	file *os.File
	size uint64
}

// NewFileContentProvider creates a ContentProvider from a file path.
func NewFileContentProvider(path string) (*FileContentProvider, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	return &FileContentProvider{
		file: file,
		size: uint64(info.Size()),
	}, nil
}

func (p *FileContentProvider) Size() uint64 {
	return p.size
}

func (p *FileContentProvider) Feed() ([]byte, error) {
	buf := make([]byte, 65536)
	n, err := p.file.Read(buf)
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// Close closes the underlying file handle.
func (p *FileContentProvider) Close() error {
	return p.file.Close()
}

// BytesContentProvider provides content from a byte slice.
type BytesContentProvider struct {
	data   []byte
	offset int
}

// NewBytesContentProvider creates a ContentProvider from a byte slice.
func NewBytesContentProvider(data []byte) *BytesContentProvider {
	return &BytesContentProvider{data: data}
}

func (p *BytesContentProvider) Size() uint64 {
	return uint64(len(p.data))
}

func (p *BytesContentProvider) Feed() ([]byte, error) {
	if p.offset >= len(p.data) {
		return nil, nil
	}

	chunkSize := 65536
	end := p.offset + chunkSize
	if end > len(p.data) {
		end = len(p.data)
	}

	chunk := p.data[p.offset:end]
	p.offset = end

	result := make([]byte, len(chunk))
	copy(result, chunk)
	return result, nil
}
