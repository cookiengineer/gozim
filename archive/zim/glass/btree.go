package glass

import (
	"bytes"
	"fmt"
	"sort"
)

// BTree is a B-tree structure for storing key-value pairs with fixed-size blocks.
type BTree struct {
	Root       uint32
	Level      byte
	NumEntries uint32
	BlockSize  uint32
	Blocks     map[uint32]*Block // Dirty blocks in memory.
	MaxBlock   uint32            // Highest allocated block number.
	FreeBlocks []uint32          // Freed block numbers available for reuse.
}

// OpenBTree opens a B-tree from a set of blocks loaded from storage.
func OpenBTree(root uint32, level byte, numEntries uint32, blocks map[uint32]*Block) *BTree {
	maxBlock := root
	for num := range blocks {
		if num > maxBlock {
			maxBlock = num
		}
	}

	return &BTree{
		Root:       root,
		Level:      level,
		NumEntries: numEntries,
		BlockSize:  BlockSize,
		Blocks:     blocks,
		MaxBlock:   maxBlock,
	}
}

// Get retrieves the value for a given key.
func (bt *BTree) Get(key []byte) ([]byte, bool, error) {
	if bt.Root == blockNone || len(bt.Blocks) == 0 {
		return nil, false, nil
	}

	block, ok := bt.Blocks[bt.Root]
	if !ok {
		return nil, false, fmt.Errorf("glass: root block %d not found", bt.Root)
	}

	return bt.searchBlock(block, key)
}

// searchBlock recursively searches for a key starting from the given block.
func (bt *BTree) searchBlock(block *Block, key []byte) ([]byte, bool, error) {
	idx := block.FindItem(key)

	if block.Level == 0 {
		// Leaf level: check if the item matches.
		if idx < len(block.Items) && bytes.Equal(block.Items[idx].Key, key) {
			return block.Items[idx].Tag, true, nil
		}
		return nil, false, nil
	}

	// Branch level: follow the child pointer.
	if idx == 0 {
		// Key is less than the first separator: follow the first child.
		// In a B-tree, items at idx point to child blocks containing keys < item key.
		// But we need the child pointer for the range covering the target key.
		// The convention: item[i] has block[i] group to the left.
		return nil, false, fmt.Errorf("glass: key before first branch separator")
	}

	// The child block for keys in the range (items[idx-1].Key, items[idx].Key)
	// is stored at items[idx-1].BlockNumber.
	item := block.Items[idx-1]
	child, ok := bt.Blocks[item.BlockNumber]
	if !ok {
		return nil, false, fmt.Errorf("glass: child block %d not found", item.BlockNumber)
	}

	return bt.searchBlock(child, key)
}

// Insert adds a key-value pair to the B-tree.
func (bt *BTree) Insert(key, tag []byte) error {
	if bt.Root == blockNone {
		// Create the first leaf block.
		block := NewBlock()
		block.Level = 0
		block.Revision = 1

		item := BlockItem{
			Key:            key,
			Tag:            tag,
			FirstComponent: true,
			LastComponent:  true,
			Compressed:     false,
			IsBranch:       false,
		}

		block.Items = append(block.Items, item)
		bt.MaxBlock++
		bt.Root = bt.MaxBlock
		bt.Level = 0
		bt.NumEntries = 1
		bt.Blocks[bt.Root] = block

		return nil
	}

	root := bt.Blocks[bt.Root]
	newRoot, err := bt.insertIntoBlock(root, key, tag)
	if err != nil {
		return err
	}

	if newRoot != nil {
		// Root was split.
		bt.MaxBlock++
		newRootNum := bt.MaxBlock
		bt.Blocks[newRootNum] = newRoot
		bt.Root = newRootNum
		bt.Level++
	}

	bt.NumEntries++
	return nil
}

// insertIntoBlock recursively inserts a key-tag pair into a block, splitting if needed.
func (bt *BTree) insertIntoBlock(block *Block, key, tag []byte) (*Block, error) {
	if block.Level == 0 {
		return bt.insertIntoLeaf(block, key, tag)
	}

	// Branch block: find the child and recurse.
	idx := block.FindItem(key)
	if idx == 0 {
		return nil, fmt.Errorf("glass: key before first branch separator")
	}

	childItem := block.Items[idx-1]
	child, ok := bt.Blocks[childItem.BlockNumber]
	if !ok {
		return nil, fmt.Errorf("glass: child block %d not found", childItem.BlockNumber)
	}

	newChild, err := bt.insertIntoBlock(child, key, tag)
	if err != nil {
		return nil, err
	}

	if newChild != nil {
		// Child was split; insert separator into this branch block.
		bt.MaxBlock++
		newChildNum := bt.MaxBlock
		bt.Blocks[newChildNum] = newChild

		separatorItem := BlockItem{
			Key:         newChild.Items[0].Key,
			BlockNumber: newChildNum,
			IsBranch:    true,
		}

		return bt.insertBranchItem(block, separatorItem)
	}

	return nil, nil
}

// insertIntoLeaf inserts a key-tag pair into a leaf block, splitting if full.
func (bt *BTree) insertIntoLeaf(block *Block, key, tag []byte) (*Block, error) {
	idx := block.FindItem(key)
	if idx < len(block.Items) && bytes.Equal(block.Items[idx].Key, key) {
		// Update existing key.
		block.Items[idx].Tag = tag
		return nil, nil
	}

	item := BlockItem{
		Key:            key,
		Tag:            tag,
		FirstComponent: true,
		LastComponent:  true,
		Compressed:     false,
		IsBranch:       false,
	}

	// Insert at the correct position.
	block.Items = append(block.Items, BlockItem{})
	copy(block.Items[idx+1:], block.Items[idx:])
	block.Items[idx] = item

	// Check if the block needs splitting.
	if bt.blockNeedsSplit(block) {
		return bt.splitBlock(block)
	}

	return nil, nil
}

// insertBranchItem inserts a separator into a branch block, splitting if full.
func (bt *BTree) insertBranchItem(block *Block, item BlockItem) (*Block, error) {
	idx := block.FindItem(item.Key)

	block.Items = append(block.Items, BlockItem{})
	copy(block.Items[idx+1:], block.Items[idx:])
	block.Items[idx] = item

	if bt.blockNeedsSplit(block) {
		return bt.splitBlock(block)
	}

	return nil, nil
}

// blockNeedsSplit returns true if the block is full.
func (bt *BTree) blockNeedsSplit(block *Block) bool {
	// Estimate encoded size.
	encoded := block.Encode()
	// Check if items start encroaching on directory space.
	minItemStart := uint16(DirStart + len(block.Items)*2 + 2)
	return encoded[DirStart-1] != 0 || minItemStart > 8100
}

// splitBlock splits a full block into two halves.
func (bt *BTree) splitBlock(block *Block) (*Block, error) {
	// Sort items by key.
	sort.Slice(block.Items, func(i, j int) bool {
		return bytes.Compare(block.Items[i].Key, block.Items[j].Key) < 0
	})

	mid := len(block.Items) / 2

	rightBlock := NewBlock()
	rightBlock.Level = block.Level
	rightBlock.Revision = block.Revision + 1
	rightBlock.Items = make([]BlockItem, len(block.Items)-mid)
	copy(rightBlock.Items, block.Items[mid:])

	block.Items = block.Items[:mid]

	return rightBlock, nil
}

// Cursor provides sequential access to B-tree entries.
type Cursor struct {
	bt       *BTree
	current  *Block
	position int
	key      []byte
}

// NewCursor creates a cursor positioned at the first entry >= the given key.
func (bt *BTree) NewCursor(key []byte) (*Cursor, error) {
	if bt.Root == blockNone {
		return &Cursor{bt: bt}, nil
	}

	c := &Cursor{bt: bt}
	err := c.seekToLeaf(bt.Root, key)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// seekToLeaf navigates to the leaf block containing the given key.
func (c *Cursor) seekToLeaf(blockNum uint32, key []byte) error {
	block, ok := c.bt.Blocks[blockNum]
	if !ok {
		return fmt.Errorf("glass: block %d not found", blockNum)
	}

	if block.Level == 0 {
		c.current = block
		c.position = block.FindItem(key)
		if c.position < len(block.Items) {
			c.key = block.Items[c.position].Key
		}
		return nil
	}

	idx := block.FindItem(key)
	if idx == 0 {
		return fmt.Errorf("glass: key before first branch separator")
	}

	childItem := block.Items[idx-1]
	return c.seekToLeaf(childItem.BlockNumber, key)
}

// Next advances the cursor to the next entry.
func (c *Cursor) Next() error {
	if c.current == nil {
		return nil
	}

	c.position++
	if c.position >= len(c.current.Items) {
		return nil // End of tree.
	}

	c.key = c.current.Items[c.position].Key
	return nil
}

// Valid returns true if the cursor is positioned at a valid entry.
func (c *Cursor) Valid() bool {
	return c.current != nil && c.position < len(c.current.Items)
}

// Key returns the current key.
func (c *Cursor) Key() []byte {
	return c.key
}

// Tag returns the current tag value.
func (c *Cursor) Tag() []byte {
	if !c.Valid() {
		return nil
	}
	return c.current.Items[c.position].Tag
}
