package zim

// IsUserContent returns true if this namespace holds user-visible content entries.
// In new namespace scheme: only C.
// In old namespace scheme: A, I, J, -.
func (ns Namespace) IsUserContent() bool {
	switch ns {
	case NamespaceContent, NamespaceArticle, NamespaceImage, NamespaceScript, NamespaceLayout:
		return true
	default:
		return false
	}
}

// IsMetadata returns true if this namespace holds metadata entries.
func (ns Namespace) IsMetadata() bool {
	return ns == NamespaceMetadata
}

// IsIndex returns true if this namespace holds index entries (fulltext, title listing).
func (ns Namespace) IsIndex() bool {
	return ns == NamespaceIndex
}

// String returns the single-character namespace identifier.
func (ns Namespace) String() string {
	return string(byte(ns))
}

// urlEntryIndex is a sentinel value for the URL pointer index field.
const urlEntryIndexNone uint32 = 0xFFFFFFFF

// clusterIndexNone is a sentinel value for cluster indexes that are not set.
const clusterIndexNone uint32 = 0xFFFFFFFF

// blobIndexNone is a sentinel value for blob indexes that are not set.
const blobIndexNone uint32 = 0xFFFFFFFF

// mimeRedirect is the MIME type index that marks a redirect entry.
const mimeRedirect uint16 = 0xFFFF

// mimeLinkTarget is the MIME type index that marks a link target entry.
const mimeLinkTarget uint16 = 0xFFFE

// mimeDeleted is the MIME type index that marks a deleted entry.
const mimeDeleted uint16 = 0xFFFD
