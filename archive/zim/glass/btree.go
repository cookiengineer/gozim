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
		if idx < len(block.Items) && bytes.Equal(block.Items[idx].Key, key) {
			return block.Items[idx].Tag, true, nil
		}
		return nil, false, nil
	}

	// Branch level: items[i] is [blockNumber, key, componentCount].
	// items[i].BlockNumber points to the child for the subtree.
	// items[i].Key is the key separator (first key in child after splits).
	// FindItem returns first item with Key >= search key.
	// For keys < items[0].Key, or when items[0].Key is null, use items[0].BlockNumber.
	// For keys >= items[0].Key, the child is items[idx-1] for idx>0,
	// or items[0] if idx==0.
	var childIdx int
	if idx == 0 {
		childIdx = 0
	} else {
		childIdx = idx - 1
	}

	childNum := block.Items[childIdx].BlockNumber
	child, ok := bt.Blocks[childNum]
	if !ok {
		return nil, false, fmt.Errorf("glass: child block %d not found", childNum)
	}
	return bt.searchBlock(child, key)
}

// Insert adds a key-value pair to the B-tree.
// Returns true if a new entry was created, false if an existing entry was updated.
func (bt *BTree) Insert(key, tag []byte) (bool, error) {
	if bt.Root == blockNone {
		block := NewBlock()
		block.Level = 0
		block.Revision = 1
		block.Items = append(block.Items, BlockItem{Key: key, Tag: tag, FirstComponent: true, LastComponent: true})
		bt.MaxBlock++
		bt.Root = bt.MaxBlock
		bt.Level = 0
		bt.NumEntries = 1
		bt.Blocks[bt.Root] = block
		return true, nil
	}

	wasNew, err := bt.insertAtRoot(key, tag)
	if err != nil {
		return false, err
	}
	if wasNew {
		bt.NumEntries++
	}
	return wasNew, nil
}

// insertAtRoot handles insertion starting from the current root.
func (bt *BTree) insertAtRoot(key, tag []byte) (bool, error) {
	root := bt.Blocks[bt.Root]
	newBlock, existed, err := bt.insertIntoBlock(root, key, tag)
	if err != nil {
		return false, err
	}

	if newBlock != nil {
		// Root split: create a new root block with two children.
		bt.MaxBlock++
		newBlockNum := bt.MaxBlock
		bt.Blocks[newBlockNum] = newBlock

		newRoot := NewBlock()
		newRoot.Level = bt.Level + 1
		newRoot.Revision = 1

		// First item: null key pointing to old root (left child)
		newRoot.Items = append(newRoot.Items, BlockItem{
			Key:         []byte{},
			BlockNumber: bt.Root,
			IsBranch:    true,
		})

		// Second item: separator key pointing to new child (right child)
		newRoot.Items = append(newRoot.Items, BlockItem{
			Key:         newBlock.Items[0].Key,
			BlockNumber: newBlockNum,
			IsBranch:    true,
		})

		bt.MaxBlock++
		bt.Root = bt.MaxBlock
		bt.Blocks[bt.Root] = newRoot
		bt.Level++
	}

	return !existed, nil
}

// insertIntoBlock recursively inserts a key-tag pair into a block, splitting if needed.
// Returns (new block if split, whether key already existed, error).
func (bt *BTree) insertIntoBlock(block *Block, key, tag []byte) (*Block, bool, error) {
	if block.Level == 0 {
		return bt.insertIntoLeaf(block, key, tag)
	}

	idx := block.FindItem(key)

	// For branch blocks, items[i].BlockNumber is the child.
	// The first item (items[0]) is always the leftmost child.
	// Use items[idx-1] for idx>0, or items[0] if idx==0.
	var childIdx int
	if idx == 0 {
		childIdx = 0
	} else {
		childIdx = idx - 1
	}

	childNum := block.Items[childIdx].BlockNumber
	child, ok := bt.Blocks[childNum]
	if !ok {
		return nil, false, fmt.Errorf("glass: child block %d not found", childNum)
	}

	newChild, existed, err := bt.insertIntoBlock(child, key, tag)
	if err != nil {
		return nil, false, err
	}

	if newChild != nil {
		bt.MaxBlock++
		newChildNum := bt.MaxBlock
		bt.Blocks[newChildNum] = newChild

		separatorItem := BlockItem{
			Key:         newChild.Items[0].Key,
			BlockNumber: newChildNum,
			IsBranch:    true,
		}

		newRoot, err := bt.insertBranchItem(block, separatorItem, childIdx)
		return newRoot, existed, err
	}

	return nil, existed, nil
}

// insertIntoLeaf inserts a key-tag pair into a leaf block, splitting if full.
// Returns (new block if split, whether key already existed, error).
func (bt *BTree) insertIntoLeaf(block *Block, key, tag []byte) (*Block, bool, error) {
	idx := block.FindItem(key)
	if idx < len(block.Items) && bytes.Equal(block.Items[idx].Key, key) {
		block.Items[idx].Tag = tag
		return nil, true, nil
	}

	item := BlockItem{
		Key: key, Tag: tag, FirstComponent: true, LastComponent: true,
		Compressed: false, IsBranch: false,
	}

	block.Items = append(block.Items, BlockItem{})
	copy(block.Items[idx+1:], block.Items[idx:])
	block.Items[idx] = item

	if bt.blockNeedsSplit(block) {
		rightBlock, err := bt.splitBlock(block)
		return rightBlock, false, err
	}

	return nil, false, nil
}

// insertBranchItem inserts a separator into a branch block, splitting if full.
// childIdx is the index of the child item that was split, so the new
// separator goes after it.
func (bt *BTree) insertBranchItem(block *Block, item BlockItem, childIdx int) (*Block, error) {
	// Insert after the child that was split.
	insertAt := childIdx + 1

	block.Items = append(block.Items, BlockItem{})
	copy(block.Items[insertAt+1:], block.Items[insertAt:])
	block.Items[insertAt] = item

	if bt.blockNeedsSplit(block) {
		return bt.splitBlock(block)
	}

	return nil, nil
}

// blockNeedsSplit returns true if the block is full.
func (bt *BTree) blockNeedsSplit(block *Block) bool {
	// Each item occupies: key length + tag length + ~5 bytes overhead.
	// Estimate average item size from the first few items.
	var totalSize int
	for _, item := range block.Items {
		totalSize += len(item.Key) + len(item.Tag) + 5
	}
	// Directory space: 2 bytes per item.
	estUsed := DirStart + len(block.Items)*2 + totalSize
	// Reserve ~64 bytes for the next item.
	return estUsed+64 > BlockSize
}

// splitBlock splits a full block into two halves.
func (bt *BTree) splitBlock(block *Block) (*Block, error) {
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
	var childIdx int
	if idx == 0 {
		childIdx = 0
	} else {
		childIdx = idx - 1
	}
	return c.seekToLeaf(block.Items[childIdx].BlockNumber, key)
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
