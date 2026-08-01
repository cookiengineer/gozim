package glass

import (
	"fmt"
	"strconv"
	"strings"
)

// DocDataTable stores document data (path strings) keyed by document ID.
type DocDataTable struct {
	table *Table
}

// OpenDocDataTable opens the document data table.
func OpenDocDataTable(table *Table) *DocDataTable {
	return &DocDataTable{table: table}
}

// Get retrieves the path string for a document ID.
func (dt *DocDataTable) Get(docID uint32) (string, error) {
	key := packUintPreservingSort(uint64(docID))

	tag, ok, err := dt.table.Get(key)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}

	return string(tag), nil
}

// Set stores a path string for a document ID.
func (dt *DocDataTable) Set(docID uint32, path string) error {
	key := packUintPreservingSort(uint64(docID))
	return dt.table.Insert(key, []byte(path))
}

// ValuesMap stores the mapping of slot names to slot numbers.
// This is written as the "valuesmap" metadata string.
type ValuesMap struct {
	Entries map[string]uint32 // name → slot number
}

// ParseValuesMap parses a valuesmap string (e.g., "title:0;wordcount:1;geo.position:2").
func ParseValuesMap(s string) *ValuesMap {
	vm := &ValuesMap{Entries: make(map[string]uint32)}

	if s == "" {
		return vm
	}

	for _, part := range strings.Split(s, ";") {
		parts := strings.SplitN(part, ":", 2)
		if len(parts) == 2 {
			name := parts[0]
			slot, err := strconv.ParseUint(parts[1], 10, 32)
			if err == nil {
				vm.Entries[name] = uint32(slot)
			}
		}
	}

	return vm
}

// Encode serializes the values map to its string representation.
func (vm *ValuesMap) Encode() string {
	var parts []string
	for name, slot := range vm.Entries {
		parts = append(parts, fmt.Sprintf("%s:%d", name, slot))
	}
	return strings.Join(parts, ";")
}

// Document is a searchable document in the Glass database.
type Document struct {
	DocID  uint32
	Data   string             // e.g., "C/path/to/article"
	Values map[uint32]string  // slot → value
	Terms  []TermEntry        // term → within-document frequency
}

// TermEntry records a term occurrence within a document.
type TermEntry struct {
	Term string
	WDF  uint32
}

// Metadata represents the user metadata stored in the database.
type Metadata struct {
	ValuesMap string
	Kind      string
	Data      string
	Language  string
	Stopwords string
}

// ReadMetadata reads metadata from the postlist table.
func ReadMetadata(table *Table) (*Metadata, error) {
	meta := &Metadata{}

	if val, ok, _ := table.Get([]byte{0x00, 0x00, 'v', 'a', 'l', 'u', 'e', 's', 'm', 'a', 'p'}); ok {
		meta.ValuesMap = string(val)
	}
	if val, ok, _ := table.Get([]byte{0x00, 0x00, 'k', 'i', 'n', 'd'}); ok {
		meta.Kind = string(val)
	}
	if val, ok, _ := table.Get([]byte{0x00, 0x00, 'd', 'a', 't', 'a'}); ok {
		meta.Data = string(val)
	}
	if val, ok, _ := table.Get([]byte{0x00, 0x00, 'l', 'a', 'n', 'g', 'u', 'a', 'g', 'e'}); ok {
		meta.Language = string(val)
	}
	if val, ok, _ := table.Get([]byte{0x00, 0x00, 's', 't', 'o', 'p', 'w', 'o', 'r', 'd', 's'}); ok {
		meta.Stopwords = string(val)
	}

	return meta, nil
}

// WriteMetadata writes metadata to the postlist table.
func WriteMetadata(table *Table, meta *Metadata) error {
	if meta.ValuesMap != "" {
		if err := table.Insert([]byte{0x00, 0x00, 'v', 'a', 'l', 'u', 'e', 's', 'm', 'a', 'p'}, []byte(meta.ValuesMap)); err != nil {
			return err
		}
	}
	if meta.Kind != "" {
		if err := table.Insert([]byte{0x00, 0x00, 'k', 'i', 'n', 'd'}, []byte(meta.Kind)); err != nil {
			return err
		}
	}
	if meta.Data != "" {
		if err := table.Insert([]byte{0x00, 0x00, 'd', 'a', 't', 'a'}, []byte(meta.Data)); err != nil {
			return err
		}
	}
	if meta.Language != "" {
		if err := table.Insert([]byte{0x00, 0x00, 'l', 'a', 'n', 'g', 'u', 'a', 'g', 'e'}, []byte(meta.Language)); err != nil {
			return err
		}
	}
	if meta.Stopwords != "" {
		if err := table.Insert([]byte{0x00, 0x00, 's', 't', 'o', 'p', 'w', 'o', 'r', 'd', 's'}, []byte(meta.Stopwords)); err != nil {
			return err
		}
	}

	return nil
}
