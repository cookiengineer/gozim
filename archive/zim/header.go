package zim

import (
	"encoding/binary"
	"fmt"
)

// Header represents the 80-byte ZIM file header.
//
// Field layout:
//
//	Offset  Size  Field          Description
//	0       4     magicNumber     Magic value, must equal Magic (0x044d495a)
//	4       2     majorVersion    Major version (5 or 6)
//	6       2     minorVersion    Minor version (0-3)
//	8       16    uuid            16-byte UUID
//	24      4     entryCount      Total number of directory entries
//	28      4     clusterCount    Total number of clusters
//	32      8     urlPtrPos       Offset of URL pointer list
//	40      8     titleIdxPos     Offset of title index (-1 if not present)
//	48      8     clusterPtrPos   Offset of cluster pointer list
//	56      8     mimeListPos     Offset of MIME type list
//	64      4     mainPage        Entry index of main page (0xFFFFFFFF if none)
//	68      4     layoutPage      Entry index of layout page (0xFFFFFFFF if none)
//	72      8     checksumPos     Offset of MD5 checksum (0 if no checksum)
type Header struct {
	MagicNumber  uint32
	MajorVersion uint16
	MinorVersion uint16
	Uuid         Uuid
	EntryCount   uint32
	ClusterCount uint32
	UrlPtrPos    uint64
	TitleIdxPos  uint64
	ClusterPtrPos uint64
	MimeListPos  uint64
	MainPage     uint32
	LayoutPage   uint32
	ChecksumPos  uint64
}

// ParseHeader reads and validates an 80-byte ZIM file header.
// It returns ErrFormat if the magic number does not match.
// It returns ErrUnsupportedVersion if the major version is not recognized.
func ParseHeader(data []byte) (*Header, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrFormat, HeaderSize, len(data))
	}

	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != Magic {
		return nil, fmt.Errorf("%w: invalid magic number 0x%08x, expected 0x%08x", ErrFormat, magic, Magic)
	}

	h := &Header{
		MagicNumber:   magic,
		MajorVersion:  binary.LittleEndian.Uint16(data[4:6]),
		MinorVersion:  binary.LittleEndian.Uint16(data[6:8]),
		EntryCount:    binary.LittleEndian.Uint32(data[24:28]),
		ClusterCount:  binary.LittleEndian.Uint32(data[28:32]),
		UrlPtrPos:     binary.LittleEndian.Uint64(data[32:40]),
		TitleIdxPos:   binary.LittleEndian.Uint64(data[40:48]),
		ClusterPtrPos: binary.LittleEndian.Uint64(data[48:56]),
		MimeListPos:   binary.LittleEndian.Uint64(data[56:64]),
		MainPage:      binary.LittleEndian.Uint32(data[64:68]),
		LayoutPage:    binary.LittleEndian.Uint32(data[68:72]),
		ChecksumPos:   binary.LittleEndian.Uint64(data[72:80]),
	}

	copy(h.Uuid[:], data[8:24])

	if h.MajorVersion != MajorVersion && h.MajorVersion != OldMajorVersion {
		return nil, fmt.Errorf("%w: major version %d (supported: %d, %d)", ErrUnsupportedVersion, h.MajorVersion, OldMajorVersion, MajorVersion)
	}

	return h, nil
}

// HasChecksum returns true if the ZIM file contains an integrity checksum.
// This is true when the MIME list position is at least 80 (header size),
// or the checksum position in the header is non-zero.
func (h *Header) HasChecksum() bool {
	return h.MimeListPos >= HeaderSize && h.ChecksumPos > 0
}

// HasNewNamespaceScheme returns true if the ZIM file uses the new namespace
// scheme (C/M/X) instead of the old scheme (A/I/J/-/M/X).
// This is true for minor version >= 1 with major version 6.
func (h *Header) HasNewNamespaceScheme() bool {
	return h.MajorVersion >= MajorVersion && h.MinorVersion >= 1
}

// HasTitleIndex returns true if the ZIM file contains a title index.
// Title index exists when TitleIdxPos is not -1 (0xFFFFFFFFFFFFFFFF).
func (h *Header) HasTitleIndex() bool {
	return h.TitleIdxPos != 0xFFFFFFFFFFFFFFFF
}

// HasMainPage returns true if the archive specifies a main page.
func (h *Header) HasMainPage() bool {
	return h.MainPage != 0xFFFFFFFF
}

// HasLayoutPage returns true if the archive specifies a layout page.
func (h *Header) HasLayoutPage() bool {
	return h.LayoutPage != 0xFFFFFFFF
}

// Validate performs structural validation of the header fields.
func (h *Header) Validate(fileSize int64) error {
	if h.EntryCount == 0 {
		return fmt.Errorf("%w: entry count is zero", ErrFormat)
	}

	if h.ClusterCount == 0 {
		return fmt.Errorf("%w: cluster count is zero", ErrFormat)
	}

	if h.UrlPtrPos >= uint64(fileSize) {
		return fmt.Errorf("%w: URL pointer list offset %d exceeds file size %d", ErrFormat, h.UrlPtrPos, fileSize)
	}

	if h.HasTitleIndex() && h.TitleIdxPos >= uint64(fileSize) {
		return fmt.Errorf("%w: title index offset %d exceeds file size %d", ErrFormat, h.TitleIdxPos, fileSize)
	}

	if h.ClusterPtrPos >= uint64(fileSize) {
		return fmt.Errorf("%w: cluster pointer list offset %d exceeds file size %d", ErrFormat, h.ClusterPtrPos, fileSize)
	}

	if h.MimeListPos >= uint64(fileSize) {
		return fmt.Errorf("%w: MIME list offset %d exceeds file size %d", ErrFormat, h.MimeListPos, fileSize)
	}

	if h.HasMainPage() && h.MainPage >= h.EntryCount {
		return fmt.Errorf("%w: main page index %d exceeds entry count %d", ErrFormat, h.MainPage, h.EntryCount)
	}

	if h.HasLayoutPage() && h.LayoutPage >= h.EntryCount {
		return fmt.Errorf("%w: layout page index %d exceeds entry count %d", ErrFormat, h.LayoutPage, h.EntryCount)
	}

	if h.HasChecksum() && h.ChecksumPos >= uint64(fileSize) {
		return fmt.Errorf("%w: checksum offset %d exceeds file size %d", ErrFormat, h.ChecksumPos, fileSize)
	}

	return nil
}

// EncodeHeader serializes the header into an 80-byte slice in ZIM file format.
func (h *Header) EncodeHeader() []byte {
	buf := make([]byte, HeaderSize)

	binary.LittleEndian.PutUint32(buf[0:4], h.MagicNumber)
	binary.LittleEndian.PutUint16(buf[4:6], h.MajorVersion)
	binary.LittleEndian.PutUint16(buf[6:8], h.MinorVersion)
	copy(buf[8:24], h.Uuid[:])
	binary.LittleEndian.PutUint32(buf[24:28], h.EntryCount)
	binary.LittleEndian.PutUint32(buf[28:32], h.ClusterCount)
	binary.LittleEndian.PutUint64(buf[32:40], h.UrlPtrPos)
	binary.LittleEndian.PutUint64(buf[40:48], h.TitleIdxPos)
	binary.LittleEndian.PutUint64(buf[48:56], h.ClusterPtrPos)
	binary.LittleEndian.PutUint64(buf[56:64], h.MimeListPos)
	binary.LittleEndian.PutUint32(buf[64:68], h.MainPage)
	binary.LittleEndian.PutUint32(buf[68:72], h.LayoutPage)
	binary.LittleEndian.PutUint64(buf[72:80], h.ChecksumPos)

	return buf
}

// NewHeader creates a Header with default values for a new ZIM file (v6, minor 3).
func NewHeader() *Header {
	return &Header{
		MagicNumber:  Magic,
		MajorVersion: MajorVersion,
		MinorVersion: MinorVersion,
		MainPage:     0xFFFFFFFF,
		LayoutPage:   0xFFFFFFFF,
		TitleIdxPos:  0xFFFFFFFFFFFFFFFF,
		MimeListPos:   HeaderSize, // Default: MIME list starts right after header
	}
}
