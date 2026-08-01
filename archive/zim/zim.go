package zim

const (
	// Magic is the ZIM file magic number.
	// In little-endian byte order, it spells "ZIM" followed by 0x04.
	Magic = 0x044d495a

	// MajorVersion is the current ZIM major version number.
	MajorVersion = 6

	// MinorVersion is the current ZIM minor version number.
	// Minor >= 1 indicates the new namespace scheme (C/M/W/X).
	MinorVersion = 3

	// OldMajorVersion is the previous ZIM major version number.
	// Version 5 files use the old namespace scheme (A/I/J/-/M/X).
	OldMajorVersion = 5

	// HeaderSize is the fixed size of the ZIM file header in bytes.
	HeaderSize = 80

	// ChecksumSize is the size of the MD5 checksum appended to the
	// end of the file, in bytes.
	ChecksumSize = 16
)

// Compression specifies the algorithm used for cluster data.
//
// Values match the on-disk ZIM format byte:
//
//	1 = no compression
//	4 = LZMA2/XZ (legacy)
//	5 = Zstandard
type Compression uint8

const (
	// CompressionNone means no compression is applied to cluster data.
	CompressionNone Compression = 1

	// CompressionZstd means cluster data is compressed using Zstandard.
	CompressionZstd Compression = 5

	// CompressionLzma means cluster data is compressed using LZMA/XZ.
	// This is only supported for reading legacy ZIM files.
	CompressionLzma Compression = 4
)

// Namespace identifies the category of a ZIM entry.
// In old namespace scheme: A=Article, I=Image, J=Script, -=Layout, M=Metadata, X=Index.
// In new namespace scheme: C=Content, M=Metadata, X=Index.
type Namespace byte

const (
	NamespaceContent  Namespace = 'C' // New scheme: all user content
	NamespaceMetadata Namespace = 'M' // Metadata entries
	NamespaceIndex    Namespace = 'X' // Index entries (fulltext, title listing)
	NamespaceArticle  Namespace = 'A' // Old scheme: articles
	NamespaceImage    Namespace = 'I' // Old scheme: images
	NamespaceScript   Namespace = 'J' // Old scheme: scripts
	NamespaceLayout   Namespace = '-' // Old scheme: layout (CSS, templates)
)

// Hint controls item behavior during ZIM file creation.
type Hint int

const (
	// HintCompress forces compression for this item regardless of MIME type heuristic.
	HintCompress Hint = iota

	// HintNoCompress disables compression for this item.
	HintNoCompress

	// HintFrontArticle marks this entry as a front article for title index and random selection.
	HintFrontArticle
)

// Megabyte is 2^20 bytes.
const Megabyte = 1 << 20
