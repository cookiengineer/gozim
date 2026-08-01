package zim

import "errors"

var (
	// ErrFormat indicates that the file is not a valid ZIM file.
	ErrFormat = errors.New("zim: invalid ZIM file format")

	// ErrEntryNotFound indicates that a requested entry could not be found.
	ErrEntryNotFound = errors.New("zim: entry not found")

	// ErrInvalidType indicates that an operation was attempted on an incompatible entry type.
	// For example, calling Item() on a redirect entry, or RedirectEntry() on a content entry.
	ErrInvalidType = errors.New("zim: invalid entry type for operation")

	// ErrChecksum indicates that the file checksum does not match.
	ErrChecksum = errors.New("zim: checksum mismatch")

	// ErrWriterClosed indicates that the writer has already been closed with Finish().
	ErrWriterClosed = errors.New("zim: writer already finished")

	// ErrDuplicatePath indicates an attempt to add an entry with a path that already exists.
	ErrDuplicatePath = errors.New("zim: duplicate entry path")

	// ErrNotImplemented indicates that the requested feature is not yet implemented.
	ErrNotImplemented = errors.New("zim: not implemented")

	// ErrUnsupportedVersion indicates that the ZIM file version is not supported.
	ErrUnsupportedVersion = errors.New("zim: unsupported ZIM version")

	// ErrUnsupportedCompression indicates that the compression algorithm is not supported.
	ErrUnsupportedCompression = errors.New("zim: unsupported compression algorithm")
)
