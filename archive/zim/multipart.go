package zim

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MultipartReader implements the Reader interface for ZIM files split
// across multiple physical files (e.g., archive.zimaa, archive.zimab).
// It provides transparent access by mapping logical file offsets to
// the appropriate physical part file.
type MultipartReader struct {
	parts []partRange
	size  int64
}

type partRange struct {
	reader   Reader
	filename string
	start    int64
	end      int64
}

// OpenMultipartReader attempts to open a ZIM file that may be split into multiple parts.
// It first tries to open the file directly. If that fails or the file doesn't exist,
// it looks for split parts with the pattern: basename + "aa", "ab", ..., "zz".
func OpenMultipartReader(filename string) (Reader, io.Closer, error) {
	// Try single file first.
	if reader, closer, err := openSingleReader(filename); err == nil {
		return reader, closer, nil
	}

	// Try split ZIM files.
	return openSplitFiles(filename)
}

// openSingleReader opens a single ZIM file using the best available reader.
func openSingleReader(filename string) (Reader, io.Closer, error) {
	return openReader(filename)
}

// openSplitFiles opens multipart ZIM files matching the pattern
// "filenameaa", "filenameab", ..., "filenamezz".
func openSplitFiles(filename string) (Reader, io.Closer, error) {
	dir := filepath.Dir(filename)
	prefix := filepath.Base(filename)

	var parts []partRange
	var closers []io.Closer

	for i := 0; i < 26*26; i++ {
		suffix := string(rune('a'+i/26)) + string(rune('a'+i%26))
		partPath := filepath.Join(dir, prefix+suffix)

		stat, err := os.Stat(partPath)
		if os.IsNotExist(err) {
			if i == 0 {
				continue // Try alternate naming
			}
			break
		}
		if err != nil {
			break
		}

		reader, closer, err := openReader(partPath)
		if err != nil {
			break
		}
		closers = append(closers, closer)

		partSize := stat.Size()

		var startOffset int64
		if len(parts) > 0 {
			startOffset = parts[len(parts)-1].end
		}

		parts = append(parts, partRange{
			reader:   reader,
			filename: partPath,
			start:    startOffset,
			end:      startOffset + partSize,
		})
	}

	if len(parts) == 0 {
		return nil, nil, fmt.Errorf("%w: no ZIM parts found for %q", ErrFormat, filename)
	}

	mr := &MultipartReader{
		parts: parts,
		size:  parts[len(parts)-1].end,
	}

	return mr, &multiCloser{closers: closers}, nil
}

// multiCloser closes multiple Closers sequentially.
type multiCloser struct {
	closers []io.Closer
}

func (mc *multiCloser) Close() error {
	var lastErr error
	for _, closer := range mc.closers {
		if err := closer.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ReadAt reads len(p) bytes starting at the specified logical offset.
func (mr *MultipartReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= mr.size {
		return 0, fmt.Errorf("offset %d exceeds total size %d", off, mr.size)
	}

	totalRead := 0
	remaining := p

	for len(remaining) > 0 && off < mr.size {
		part := mr.findPart(off)
		if part == nil {
			break
		}

		partOffset := off - part.start
		partSize := part.end - part.start
		maxRead := int(partSize - partOffset)
		if maxRead > len(remaining) {
			maxRead = len(remaining)
		}

		n, err := part.reader.ReadAt(remaining[:maxRead], partOffset)
		totalRead += n
		if err != nil {
			return totalRead, err
		}

		off += int64(n)
		remaining = remaining[n:]
	}

	return totalRead, nil
}

// Size returns the total logical size of all parts combined.
func (mr *MultipartReader) Size() int64 {
	return mr.size
}

// findPart locates the part file containing the given logical offset.
func (mr *MultipartReader) findPart(offset int64) *partRange {
	for i := range mr.parts {
		if offset >= mr.parts[i].start && offset < mr.parts[i].end {
			return &mr.parts[i]
		}
	}
	return nil
}
