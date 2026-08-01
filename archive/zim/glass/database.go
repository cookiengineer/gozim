package glass

import (
	"bytes"
	"crypto/rand"
	"fmt"
)

// Database represents an open Xapian Glass database.
// It provides read and write access to the underlying B-tree tables
// for full-text search indexing.
type Database struct {
	version  *VersionInfo
	postlist *Table
	docdata  *Table
	metadata *Metadata

	docCount    uint32
	lastDocID   uint32
	totalDocLen uint64
}

// OpenDatabase opens an existing Glass database from a byte slice.
// This is used to read Xapian databases embedded in ZIM files.
func OpenDatabase(data []byte) (*Database, error) {
	version, versionLen, err := ReadVersion(data)
	if err != nil {
		return nil, fmt.Errorf("glass: reading version: %w", err)
	}

	// Tables follow the version header.
	tableData := data[versionLen:]

	db := &Database{
		version:  version,
		docCount: version.DocCount,
		lastDocID: version.LastDocID + version.DocCount,
		totalDocLen: version.TotalDocLen,
	}

	// Read postlist table.
	plInfo := version.RootInfos[TablePostlist]
	if plInfo.Root != blockNone {
		plData, err := readTableData(tableData, 0, plInfo.BlockSize)
		if err != nil {
			return nil, fmt.Errorf("glass: reading postlist: %w", err)
		}

		db.postlist, err = DeserializeTable(TablePostlist, plInfo.Root, plInfo.Level, plInfo.NumEntries, plData, plInfo.CompressMin)
		if err != nil {
			return nil, err
		}

		// Read metadata from postlist.
		db.metadata, _ = ReadMetadata(db.postlist)
	}

	// Read docdata table.
	ddInfo := version.RootInfos[TableDocdata]
	if ddInfo.Root != blockNone {
		// Skip past postlist table data.
		var ddOffset int64
		if plInfo.BlockSize > 0 {
			ddOffset = estimateTableSize(&plInfo)
		}

		ddData, err := readTableData(tableData, ddOffset, ddInfo.BlockSize)
		if err != nil {
			return nil, fmt.Errorf("glass: reading docdata: %w", err)
		}

		db.docdata, err = DeserializeTable(TableDocdata, ddInfo.Root, ddInfo.Level, ddInfo.NumEntries, ddData, ddInfo.CompressMin)
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

// CreateDatabase creates a new empty Glass database for indexing.
func CreateDatabase(uuid [16]byte) *Database {
	v := &VersionInfo{
		Revision: 1,
	}
	copy(v.UUID[:], uuid[:])

	if uuid == [16]byte{} {
		rand.Read(v.UUID[:])
	}

	for i := 0; i < numTables; i++ {
		v.RootInfos[i].BlockSize = BlockSize
		v.RootInfos[i].CompressMin = 4
	}

	// Postlist and position don't compress.
	v.RootInfos[TablePostlist].CompressMin = 0
	v.RootInfos[TablePosition].CompressMin = 0

	db := &Database{
		version:  v,
		postlist: NewTable(TablePostlist),
		docdata:  NewTable(TableDocdata),
		docCount: 0,
		lastDocID: 0,
	}

	return db
}

// AddDocument adds a document to the database and returns its assigned ID.
func (db *Database) AddDocument(doc *Document) (uint32, error) {
	if db.docdata == nil {
		return 0, fmt.Errorf("glass: docdata table not initialized")
	}

	db.lastDocID++
	doc.DocID = db.lastDocID

	// Store document data.
	if err := DocDataTable_set(db.docdata, doc.DocID, doc.Data); err != nil {
		return 0, fmt.Errorf("glass: storing docdata: %w", err)
	}

	// Build posting lists for each term.
	for _, termEntry := range doc.Terms {
		posting := Posting{
			DocumentID: doc.DocID,
			WDF:        termEntry.WDF,
		}

		if err := BuildPostingList(db.postlist, termEntry.Term, posting); err != nil {
			return 0, fmt.Errorf("glass: building posting list for %q: %w", termEntry.Term, err)
		}

		db.totalDocLen += uint64(termEntry.WDF)
	}

	// Store values.
	for slot, value := range doc.Values {
		valKey := append([]byte{0x00, 0xd8}, packUint(uint64(slot))...)
		valKey = append(valKey, packUintPreservingSort(uint64(doc.DocID))...)

		valData := append(packUint(0), packUint(1)...)
		valData = append(valData, []byte(value)...)

		if err := db.postlist.Insert(valKey, valData); err != nil {
			return 0, fmt.Errorf("glass: storing value: %w", err)
		}
	}

	db.docCount++

	return doc.DocID, nil
}

// GetDocument retrieves a document by ID.
func (db *Database) GetDocument(docID uint32) (*Document, error) {
	path, err := DocDataTable_get(db.docdata, docID)
	if err != nil {
		return nil, err
	}

	return &Document{
		DocID: docID,
		Data:  path,
	}, nil
}

// DocCount returns the number of indexed documents.
func (db *Database) DocCount() uint32 {
	return db.docCount
}

// SetMetadata stores metadata in the postlist table.
func (db *Database) SetMetadata(meta *Metadata) error {
	if db.postlist == nil {
		db.postlist = NewTable(TablePostlist)
	}

	if err := WriteMetadata(db.postlist, meta); err != nil {
		return err
	}

	db.metadata = meta
	return nil
}

// GetMetadata retrieves stored metadata.
func (db *Database) GetMetadata() *Metadata {
	return db.metadata
}

// Commit finalizes pending writes and updates version info.
func (db *Database) Commit() error {
	db.version.DocCount = db.docCount
	db.version.LastDocID = db.lastDocID - db.docCount
	db.version.TotalDocLen = db.totalDocLen
	db.version.Revision++

	// Update root infos for used tables.
	if db.postlist != nil {
		root, level, numEntries := db.postlist.RootInfo()
		db.version.RootInfos[TablePostlist].Root = root
		db.version.RootInfos[TablePostlist].Level = level
		db.version.RootInfos[TablePostlist].NumEntries = numEntries
	}

	if db.docdata != nil {
		root, level, numEntries := db.docdata.RootInfo()
		db.version.RootInfos[TableDocdata].Root = root
		db.version.RootInfos[TableDocdata].Level = level
		db.version.RootInfos[TableDocdata].NumEntries = numEntries
	}

	return nil
}

// Compact produces a single-file byte representation of the database,
// matching libzim's DBCOMPACT_SINGLE_FILE output.
func (db *Database) Compact() ([]byte, error) {
	db.Commit()

	var buf bytes.Buffer

	// Write version header.
	versionData := EncodeVersion(db.version)
	buf.Write(versionData)

	// Write tables sequentially.
	tables := []*Table{db.postlist, db.docdata}
	for _, table := range tables {
		if table == nil {
			continue
		}

		tableData, err := table.Serialize()
		if err != nil {
			return nil, fmt.Errorf("glass: serializing table: %w", err)
		}

		buf.Write(tableData)
	}

	return buf.Bytes(), nil
}

// Helper functions.

func readTableData(data []byte, offset int64, blockSize uint32) ([]byte, error) {
	if offset >= int64(len(data)) {
		return nil, nil
	}

	available := int64(len(data)) - offset
	// Estimate: read up to 1000 blocks.
	maxRead := int64(1000) * int64(blockSize)
	if available > maxRead {
		available = maxRead
	}

	return data[offset : offset+available], nil
}

func estimateTableSize(ri *RootInfo) int64 {
	return int64(ri.BlockSize) * int64(ri.NumEntries)
}

func DocDataTable_set(table *Table, docID uint32, path string) error {
	key := packUintPreservingSort(uint64(docID))
	return table.Insert(key, []byte(path))
}

func DocDataTable_get(table *Table, docID uint32) (string, error) {
	key := packUintPreservingSort(uint64(docID))
	tag, ok, err := table.Get(key)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	return string(tag), nil
}
