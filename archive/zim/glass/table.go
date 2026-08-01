package glass

import (
	"fmt"
)

// Table wraps a B-tree with table-type-specific behavior.
// Each Glass database has up to 6 tables (postlist, docdata, termlist, position, spelling, synonym).
type Table struct {
	btree      *BTree
	tableType  TableType
	compressMin int
}

// OpenTable opens a table from a set of blocks.
func OpenTable(tableType TableType, root uint32, level byte, numEntries uint32, blocks map[uint32]*Block, compressMin int) *Table {
	btree := OpenBTree(root, level, numEntries, blocks)

	return &Table{
		btree:       btree,
		tableType:   tableType,
		compressMin: compressMin,
	}
}

// NewTable creates a new empty table.
func NewTable(tableType TableType) *Table {
	compressMin := 4 // Default: compress if >= 4 bytes.

	switch tableType {
	case TablePostlist, TablePosition:
		compressMin = 0 // No compression for these tables.
	case TableDocdata, TableTermlist, TableSpelling, TableSynonym:
		compressMin = 18
	}

	return &Table{
		btree: &BTree{
			BlockSize: BlockSize,
			Blocks:    make(map[uint32]*Block),
		},
		tableType: tableType,
		compressMin: compressMin,
	}
}

// Get retrieves a value by key.
func (t *Table) Get(key []byte) ([]byte, bool, error) {
	return t.btree.Get(key)
}

// Insert adds a key-value pair to the table.
func (t *Table) Insert(key, tag []byte) error {
	// Compress the tag if it's large enough.
	if t.compressMin > 0 && len(tag) >= t.compressMin {
		compressed, err := CompressTag(tag)
		if err == nil && len(compressed) < len(tag) {
			tag = compressed
		}
	}

	return t.btree.Insert(key, tag)
}

// Cursor returns a cursor positioned at the first entry >= key.
func (t *Table) Cursor(key []byte) (*Cursor, error) {
	return t.btree.NewCursor(key)
}

// NumEntries returns the number of entries in the table.
func (t *Table) NumEntries() uint32 {
	return t.btree.NumEntries
}

// RootInfo returns the B-tree root information.
func (t *Table) RootInfo() (root uint32, level byte, numEntries uint32) {
	return t.btree.Root, t.btree.Level, t.btree.NumEntries
}

// Blocks returns all blocks in the table (used during serialization).
func (t *Table) Blocks() map[uint32]*Block {
	return t.btree.Blocks
}

// MaxBlock returns the highest block number.
func (t *Table) MaxBlock() uint32 {
	return t.btree.MaxBlock
}

// FreeList returns free block numbers.
func (t *Table) FreeList() []uint32 {
	return t.btree.FreeBlocks
}

// TableType returns the type of this table.
func (t *Table) TableType() TableType {
	return t.tableType
}

// Serialize returns all blocks as a byte slice for storage.
func (t *Table) Serialize() ([]byte, error) {
	if len(t.btree.Blocks) == 0 {
		return nil, nil
	}

	maxBlock := t.btree.MaxBlock
	totalSize := int(maxBlock+1) * BlockSize
	data := make([]byte, totalSize)

	for num, block := range t.btree.Blocks {
		offset := int(num) * BlockSize
		if offset+BlockSize > totalSize {
			return nil, fmt.Errorf("glass: block %d out of range", num)
		}
		encoded := block.Encode()
		copy(data[offset:], encoded)
	}

	return data, nil
}

// DeserializeTable reconstructs a table from serialized block data.
func DeserializeTable(tableType TableType, root uint32, level byte, numEntries uint32, data []byte, compressMin int) (*Table, error) {
	numBlocks := len(data) / BlockSize
	blocks := make(map[uint32]*Block, numBlocks)

	for i := 0; i < numBlocks; i++ {
		offset := i * BlockSize
		blockData := data[offset : offset+BlockSize]

		block, err := ReadBlock(blockData)
		if err != nil {
			// Skip empty blocks.
			continue
		}
		blocks[uint32(i)] = block
	}

	btree := OpenBTree(root, level, numEntries, blocks)

	return &Table{
		btree:        btree,
		tableType:    tableType,
		compressMin:  compressMin,
	}, nil
}
