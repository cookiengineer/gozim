package zim

import (
	"encoding/binary"
	"fmt"
)

// EntryType classifies what kind of ZIM entry a directory entry represents.
type EntryType uint8

const (
	// EntryTypeContent is a normal content entry with data in a cluster.
	EntryTypeContent EntryType = iota

	// EntryTypeRedirect points to another entry.
	EntryTypeRedirect

	// EntryTypeLinkTarget is marked as the target of a redirect (no cluster data).
	EntryTypeLinkTarget

	// EntryTypeDeleted marks an entry that has been removed.
	EntryTypeDeleted
)

// Dirent is a parsed ZIM directory entry (internal representation).
// Entry uses this data to provide the public API.
type Dirent struct {
	MimeTypeIndex uint16
	Namespace     Namespace
	EntryType     EntryType
	Version       uint32

	// For content/linktarget entries:
	ClusterNumber uint32
	BlobNumber    uint32

	// For redirect entries:
	RedirectIndex uint32

	// String fields:
	Path  string
	Title string

	// Extra binary parameters (rarely used).
	Extra []byte
}

// ParseDirent parses a single directory entry from the given byte slice.
// It returns the Dirent and the number of bytes consumed.
// The byte slice must contain the entire dirent (variable length).
func ParseDirent(data []byte) (*Dirent, int, error) {
	if len(data) < 12 {
		return nil, 0, fmt.Errorf("%w: directory entry too short (%d bytes)", ErrFormat, len(data))
	}

	d := &Dirent{
		MimeTypeIndex: binary.LittleEndian.Uint16(data[0:2]),
		Namespace:     Namespace(data[3]),
		Version:       binary.LittleEndian.Uint32(data[4:8]),
	}

	extraLen := int(data[2])

	offset := 8

	// Determine entry type and parse appropriate fields.
	switch d.MimeTypeIndex {
	case mimeRedirect:
		d.EntryType = EntryTypeRedirect
		if len(data) < offset+4 {
			return nil, 0, fmt.Errorf("%w: redirect dirent too short", ErrFormat)
		}
		d.RedirectIndex = binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

	case mimeLinkTarget:
		d.EntryType = EntryTypeLinkTarget
		if len(data) < offset+8 {
			return nil, 0, fmt.Errorf("%w: linktarget dirent too short", ErrFormat)
		}
		d.ClusterNumber = binary.LittleEndian.Uint32(data[offset : offset+4])
		d.BlobNumber = binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8

	case mimeDeleted:
		d.EntryType = EntryTypeDeleted
		if len(data) < offset+8 {
			return nil, 0, fmt.Errorf("%w: deleted dirent too short", ErrFormat)
		}
		d.ClusterNumber = binary.LittleEndian.Uint32(data[offset : offset+4])
		d.BlobNumber = binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8

	default:
		d.EntryType = EntryTypeContent
		if len(data) < offset+8 {
			return nil, 0, fmt.Errorf("%w: content dirent too short", ErrFormat)
		}
		d.ClusterNumber = binary.LittleEndian.Uint32(data[offset : offset+4])
		d.BlobNumber = binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8
	}

	// Parse path (null-terminated string).
	path, pathLen, err := readNullTerminated(data[offset:])
	if err != nil {
		return nil, 0, fmt.Errorf("%w: path in dirent: %v", ErrFormat, err)
	}
	d.Path = string(path)
	offset += pathLen

	// Parse title (null-terminated string).
	title, titleLen, err := readNullTerminated(data[offset:])
	if err != nil {
		return nil, 0, fmt.Errorf("%w: title in dirent: %v", ErrFormat, err)
	}
	d.Title = string(title)
	offset += titleLen

	// Read extra parameter bytes.
	if extraLen > 0 {
		if len(data) < offset+extraLen {
			return nil, 0, fmt.Errorf("%w: extra parameters truncated (%d bytes expected)", ErrFormat, extraLen)
		}
		d.Extra = make([]byte, extraLen)
		copy(d.Extra, data[offset:offset+extraLen])
		offset += extraLen
	}

	return d, offset, nil
}

// EncodeDirent serializes a Dirent to its binary representation.
func EncodeDirent(d *Dirent) []byte {
	// Calculate path + title + extra lengths.
	pathLen := len(d.Path) + 1 // +1 for null terminator
	titleLen := len(d.Title) + 1
	extraLen := len(d.Extra)

	fixedLen := 8 // base header before path
	switch d.EntryType {
	case EntryTypeRedirect:
		fixedLen += 4
	default:
		fixedLen += 8
	}

	totalLen := fixedLen + pathLen + titleLen + extraLen
	buf := make([]byte, totalLen)

	// MIME type index.
	binary.LittleEndian.PutUint16(buf[0:2], d.MimeTypeIndex)

	// Extra parameter length.
	buf[2] = byte(extraLen)

	// Namespace.
	buf[3] = byte(d.Namespace)

	// Version.
	binary.LittleEndian.PutUint32(buf[4:8], d.Version)

	offset := 8

	// Entry-specific fields.
	switch d.EntryType {
	case EntryTypeRedirect:
		binary.LittleEndian.PutUint32(buf[offset:offset+4], d.RedirectIndex)
		offset += 4
	default:
		binary.LittleEndian.PutUint32(buf[offset:offset+4], d.ClusterNumber)
		binary.LittleEndian.PutUint32(buf[offset+4:offset+8], d.BlobNumber)
		offset += 8
	}

	// Path.
	copy(buf[offset:], []byte(d.Path))
	offset += pathLen - 1
	buf[offset] = 0
	offset++

	// Title.
	copy(buf[offset:], []byte(d.Title))
	offset += titleLen - 1
	buf[offset] = 0
	offset++

	// Extra.
	if extraLen > 0 {
		copy(buf[offset:], d.Extra)
	}

	return buf
}

// IsRedirect returns true if this is a redirect entry.
func (d *Dirent) IsRedirect() bool {
	return d.EntryType == EntryTypeRedirect
}

// IsLinkTarget returns true if this is a link target entry.
func (d *Dirent) IsLinkTarget() bool {
	return d.EntryType == EntryTypeLinkTarget
}

// IsDeleted returns true if this is a deleted entry.
func (d *Dirent) IsDeleted() bool {
	return d.EntryType == EntryTypeDeleted
}

// IsContent returns true if this is a normal content entry.
func (d *Dirent) IsContent() bool {
	return d.EntryType == EntryTypeContent
}

// readNullTerminated reads a null-terminated byte string from data.
// Returns the string bytes (without null terminator), the number of bytes consumed
// (including the null terminator), and an error if no null is found within bounds.
func readNullTerminated(data []byte) ([]byte, int, error) {
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			return data[:i], i + 1, nil
		}
	}
	return nil, 0, fmt.Errorf("%w: unterminated string", ErrFormat)
}
