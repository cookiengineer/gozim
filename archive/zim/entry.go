package zim

import "fmt"

// Entry represents a read-only ZIM directory entry.
// It provides access to the entry's metadata and, for content entries,
// the associated content Item.
type Entry struct {
	archive *Archive
	dirent  *Dirent
	index   uint32
}

// Path returns the entry's URL path.
// In old namespace scheme: "A/article.html" (includes namespace prefix).
// In new namespace scheme: "article.html" (no namespace prefix).
func (e *Entry) Path() string {
	return e.dirent.Path
}

// Title returns the human-readable title of the entry.
func (e *Entry) Title() string {
	return e.dirent.Title
}

// Namespace returns the namespace this entry belongs to.
func (e *Entry) Namespace() Namespace {
	return e.dirent.Namespace
}

// Index returns the position of this entry in the URL pointer list (0-based).
func (e *Entry) Index() uint32 {
	return e.index
}

// IsRedirect returns true if this entry is a redirect to another entry.
func (e *Entry) IsRedirect() bool {
	return e.dirent.IsRedirect()
}

// RedirectEntry returns the target entry for a redirect.
// Returns ErrInvalidType if this is not a redirect entry.
func (e *Entry) RedirectEntry() (*Entry, error) {
	if !e.dirent.IsRedirect() {
		return nil, fmt.Errorf("%w: entry is not a redirect", ErrInvalidType)
	}

	return e.archive.EntryByIndex(e.dirent.RedirectIndex)
}

// Item returns the content item associated with this entry.
// If followRedirect is true, redirect entries are resolved to their target item.
// Returns ErrInvalidType for deleted entries.
func (e *Entry) Item(followRedirect bool) (*Item, error) {
	if e.dirent.IsDeleted() {
		return nil, fmt.Errorf("%w: entry is deleted", ErrInvalidType)
	}

	if e.dirent.IsRedirect() {
		if !followRedirect {
			return nil, fmt.Errorf("%w: entry is a redirect (use followRedirect=true to resolve)", ErrInvalidType)
		}
		target, err := e.RedirectEntry()
		if err != nil {
			return nil, err
		}
		return target.Item(false)
	}

	if e.dirent.IsLinkTarget() {
		return nil, fmt.Errorf("%w: entry is a link target (no content)", ErrInvalidType)
	}

	return e.archive.readItem(e.dirent)
}

// ClusterIndex returns the cluster number containing this entry's data.
func (e *Entry) ClusterIndex() uint32 {
	return e.dirent.ClusterNumber
}

// BlobIndex returns the blob index within the cluster for this entry's data.
func (e *Entry) BlobIndex() uint32 {
	return e.dirent.BlobNumber
}

// MimeTypeIndex returns the raw MIME type index from the directory entry.
func (e *Entry) MimeTypeIndex() uint16 {
	return e.dirent.MimeTypeIndex
}
