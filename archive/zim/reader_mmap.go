//go:build !windows

package zim

import (
	"os"
	"syscall"
)

// MmapReader implements Reader using a memory-mapped file.
// This is typically faster than ReadAt-based access because the kernel
// manages page faults and caching transparently.
type MmapReader struct {
	data []byte
	size int64
}

// openMmapReader opens a file and returns a memory-mapped reader.
// Returns an error if mmap is not available or fails.
func openMmapReader(filename string) (*MmapReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	size := info.Size()
	if size == 0 {
		return nil, os.ErrInvalid
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	return &MmapReader{data: data, size: size}, nil
}

// ReadAt reads len(p) bytes from the mapped memory starting at byte offset off.
func (r *MmapReader) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= r.size {
		return 0, syscall.EINVAL
	}

	n := copy(p, r.data[off:])
	if int64(n) < int64(len(p)) && off+int64(len(p)) > r.size {
		return n, syscall.EINVAL
	}

	return n, nil
}

// Size returns the total mapped size in bytes.
func (r *MmapReader) Size() int64 {
	return r.size
}

// Close unmaps the memory-mapped region.
func (r *MmapReader) Close() error {
	return syscall.Munmap(r.data)
}
