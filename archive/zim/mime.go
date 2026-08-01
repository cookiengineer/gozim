package zim

import (
	"bytes"
	"fmt"
)

// MimeList represents the MIME type table embedded in a ZIM file header.
// It maps a 16-bit index to a MIME type string. The index 0xFFFF is
// reserved for redirect entries, 0xFFFE for link targets, and 0xFFFD
// for deleted entries.
type MimeList struct {
	entries []string
}

// ParseMimeList parses a null-terminated list of MIME type strings
// from the given byte slice. Each string is separated by a null byte.
// Parsing stops at the first empty string (consecutive null bytes)
// or end of data.
func ParseMimeList(data []byte) (*MimeList, error) {
	if len(data) == 0 {
		return &MimeList{}, nil
	}

	var entries []string
	start := 0

	for i, b := range data {
		if b == 0 {
			if i > start {
				entry := string(data[start:i])
				if entry != "" {
					entries = append(entries, entry)
				}
			}
			start = i + 1

			// Stop at empty entries (consecutive null bytes or end of MIME list).
			if i+1 < len(data) && data[i+1] == 0 {
				return &MimeList{entries: entries}, nil
			}
		}
	}

	// Last entry if data is not null-terminated.
	if start < len(data) {
		entry := string(data[start:])
		if entry != "" {
			entries = append(entries, entry)
		}
	}

	return &MimeList{entries: entries}, nil
}

// MimeType returns the MIME type string for the given index.
// Returns an empty string if the index is out of bounds.
// Special indexes:
//
//	0xFFFF (65535) = redirect entry
//	0xFFFE (65534) = linktarget entry
//	0xFFFD (65533) = deleted entry
func (ml *MimeList) MimeType(index uint16) string {
	switch index {
	case mimeRedirect:
		return "redirect"
	case mimeLinkTarget:
		return "linktarget"
	case mimeDeleted:
		return "deleted"
	}

	idx := int(index)
	if idx < 0 || idx >= len(ml.entries) {
		return ""
	}

	return ml.entries[idx]
}

// Index returns the MIME type index for the given MIME type string.
// Returns false if the MIME type is not found.
func (ml *MimeList) Index(mimeType string) (uint16, bool) {
	for i, entry := range ml.entries {
		if entry == mimeType {
			return uint16(i), true
		}
	}
	return 0, false
}

// Encode serializes the MIME type list to a null-terminated byte sequence.
func (ml *MimeList) Encode() []byte {
	if len(ml.entries) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, entry := range ml.entries {
		buf.WriteString(entry)
		buf.WriteByte(0)
	}

	return buf.Bytes()
}

// Len returns the number of MIME type entries.
func (ml *MimeList) Len() int {
	return len(ml.entries)
}

// Add appends a MIME type to the list if it is not already present.
// Returns the index of the MIME type.
func (ml *MimeList) Add(mimeType string) uint16 {
	for i, entry := range ml.entries {
		if entry == mimeType {
			return uint16(i)
		}
	}

	ml.entries = append(ml.entries, mimeType)
	return uint16(len(ml.entries) - 1)
}

// Validate checks that a MIME type index refers to a valid entry in the list.
// It returns nil for valid redirect/linktarget/deleted sentinel indices,
// and an error for out-of-range indices.
func (ml *MimeList) Validate(index uint16) error {
	if index == mimeRedirect || index == mimeLinkTarget || index == mimeDeleted {
		return nil
	}

	idx := int(index)
	if idx < 0 || idx >= len(ml.entries) {
		return fmt.Errorf("zim: MIME index %d out of range (0-%d)", idx, len(ml.entries)-1)
	}

	return nil
}
