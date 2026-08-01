package zim

import (
	"io"
	"os"
)

// Reader provides random-access read capability for ZIM file data.
// It is the abstraction layer that allows transparent use of mmap
// or standard file I/O depending on platform support.
type Reader interface {
	io.ReaderAt
	Size() int64
}

// OSReader wraps an *os.File and implements the Reader interface using
// standard file read operations.
type OSReader struct {
	file *os.File
	size int64
}

// OpenOSReader opens a file and returns an OSReader that uses ReadAt for access.
func OpenOSReader(filename string) (*OSReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	return &OSReader{file: file, size: info.Size()}, nil
}

// ReadAt reads len(p) bytes from the file starting at byte offset off.
func (r *OSReader) ReadAt(p []byte, off int64) (int, error) {
	return r.file.ReadAt(p, off)
}

// Size returns the total file size in bytes.
func (r *OSReader) Size() int64 {
	return r.size
}

// Close closes the underlying file.
func (r *OSReader) Close() error {
	return r.file.Close()
}

// openReader opens a ZIM file with the best available reader implementation.
// On platforms with mmap support (linux, macOS, etc.), it uses memory-mapped I/O.
// Otherwise, it falls back to standard file ReadAt operations.
func openReader(filename string) (Reader, io.Closer, error) {
	// Try mmap first; fall back to OSReader on failure.
	mmapReader, err := openMmapReader(filename)
	if err == nil {
		return mmapReader, mmapReader, nil
	}

	reader, err := OpenOSReader(filename)
	if err != nil {
		return nil, nil, err
	}

	return reader, reader, nil
}
