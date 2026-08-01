package glass

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

// Block represents a B-tree block in the Glass backend.
type Block struct {
	Revision uint32
	Level    byte
	MaxFree  uint16
	TotalFree uint16
	DirEnd   uint16
	Items    []BlockItem
	Data     []byte // Raw block data (8192 bytes).
}

// BlockItem represents a single item within a block.
type BlockItem struct {
	Key            []byte
	Tag            []byte
	Compressed     bool
	LastComponent  bool
	FirstComponent bool
	ComponentCount uint16 // Only for non-first components.
	BlockNumber    uint32 // Only for branch items.
	IsBranch       bool
}

// NewBlock creates an empty block with default capacity.
func NewBlock() *Block {
	return &Block{
		Data: make([]byte, BlockSize),
	}
}

// ReadBlock parses a block from raw byte data.
func ReadBlock(data []byte) (*Block, error) {
	if len(data) < DirStart {
		return nil, fmt.Errorf("glass: block too short (%d bytes)", len(data))
	}

	b := &Block{
		Revision:  binary.BigEndian.Uint32(data[0:4]),
		Level:     data[4],
		MaxFree:   binary.BigEndian.Uint16(data[5:7]),
		TotalFree: binary.BigEndian.Uint16(data[7:9]),
		DirEnd:    binary.BigEndian.Uint16(data[9:11]),
		Data:      data,
	}

	if int(b.DirEnd) > len(data) {
		return nil, fmt.Errorf("glass: directory extends past block end")
	}

	// Read directory entries.
	numItems := (b.DirEnd - DirStart) / 2
	b.Items = make([]BlockItem, numItems)

	for i := 0; i < int(numItems); i++ {
		offset := binary.BigEndian.Uint16(data[DirStart+i*2:])

		item, err := b.readItem(int(offset))
		if err != nil {
			return nil, fmt.Errorf("glass: reading item %d: %w", i, err)
		}

		b.Items[i] = *item
	}

	return b, nil
}

// readItem decodes a single item from the block at the given offset.
func (b *Block) readItem(offset int) (*BlockItem, error) {
	if offset < DirStart || offset >= len(b.Data) {
		return nil, fmt.Errorf("invalid item offset %d", offset)
	}

	isBranch := b.Level != 0

	if isBranch {
		return b.readBranchItem(offset)
	}

	return b.readLeafItem(offset)
}

// readLeafItem decodes a leaf-level item.
func (b *Block) readLeafItem(offset int) (*BlockItem, error) {
	data := b.Data[offset:]

	if len(data) < 3 {
		return nil, fmt.Errorf("leaf item too short")
	}

	i2 := binary.BigEndian.Uint16(data[0:2])
	itemSize := int(i2&0x1FFF) + 3

	if itemSize > len(data) {
		return nil, fmt.Errorf("leaf item size %d exceeds available data", itemSize)
	}

	keyLen := int(data[2])
	if keyLen > itemSize-3 {
		return nil, fmt.Errorf("key length %d exceeds item", keyLen)
	}

	key := make([]byte, keyLen)
	copy(key, data[3:3+keyLen])

	tagOffset := 3 + keyLen
	if i2&ItemFirstBit == 0 {
		// Non-first component: has component count.
		if tagOffset+2 > itemSize {
			return nil, fmt.Errorf("component count missing")
		}
		componentCount := binary.BigEndian.Uint16(data[tagOffset:])
		tagOffset += 2

		tag := make([]byte, itemSize-tagOffset)
		copy(tag, data[tagOffset:itemSize])

		return &BlockItem{
			Key:            key,
			Tag:            tag,
			Compressed:     (i2 & ItemCompressedBit) != 0,
			LastComponent:  (i2 & ItemLastBit) != 0,
			FirstComponent: (i2 & ItemFirstBit) != 0,
			ComponentCount: componentCount,
			IsBranch:       false,
		}, nil
	}

	tag := make([]byte, itemSize-tagOffset)
	copy(tag, data[tagOffset:itemSize])

	return &BlockItem{
		Key:            key,
		Tag:            tag,
		Compressed:     (i2 & ItemCompressedBit) != 0,
		LastComponent:  (i2 & ItemLastBit) != 0,
		FirstComponent: (i2 & ItemFirstBit) != 0,
		ComponentCount: 0,
		IsBranch:       false,
	}, nil
}

// readBranchItem decodes a branch (internal node) item.
func (b *Block) readBranchItem(offset int) (*BlockItem, error) {
	data := b.Data[offset:]

	if len(data) < 7 {
		return nil, fmt.Errorf("branch item too short")
	}

	blockNumber := binary.BigEndian.Uint32(data[0:4])
	keyLen := int(data[4])

	if keyLen > len(data)-7 {
		return nil, fmt.Errorf("key length %d exceeds item", keyLen)
	}

	key := make([]byte, keyLen)
	copy(key, data[5:5+keyLen])

	componentCount := binary.BigEndian.Uint16(data[5+keyLen:])

	return &BlockItem{
		Key:            key,
		BlockNumber:    blockNumber,
		ComponentCount: componentCount,
		IsBranch:       true,
	}, nil
}

// Encode serializes the block to raw bytes.
func (b *Block) Encode() []byte {
	data := make([]byte, BlockSize)

	binary.BigEndian.PutUint32(data[0:4], b.Revision)
	data[4] = b.Level
	binary.BigEndian.PutUint16(data[5:7], b.MaxFree)
	binary.BigEndian.PutUint16(data[7:9], b.TotalFree)
	binary.BigEndian.PutUint16(data[9:11], b.DirEnd)

	// Write items from the end of the block.
	itemEnd := BlockSize
	var dirEntries []uint16

	for _, item := range b.Items {
		if item.IsBranch {
			itemEnd = b.writeBranchItem(data, itemEnd, item)
		} else {
			itemEnd = b.writeLeafItem(data, itemEnd, item)
		}
		dirEntries = append(dirEntries, uint16(itemEnd))
	}

	b.DirEnd = uint16(DirStart + len(dirEntries)*2)

	// Write directory.
	for i, entry := range dirEntries {
		binary.BigEndian.PutUint16(data[DirStart+i*2:], entry)
	}

	binary.BigEndian.PutUint16(data[9:11], b.DirEnd)
	b.Data = data

	return data
}

// writeLeafItem encodes a leaf item into the block data, writing backwards from end.
// Returns the new end offset.
func (b *Block) writeLeafItem(data []byte, endOffset int, item BlockItem) int {
	i2 := uint16(0)
	if item.Compressed {
		i2 |= ItemCompressedBit
	}
	if item.LastComponent {
		i2 |= ItemLastBit
	}
	if item.FirstComponent {
		i2 |= ItemFirstBit
	}

	totalSize := 2 + 1 + len(item.Key) + len(item.Tag) // I2 + keyLen + key + tag
	if !item.FirstComponent {
		totalSize += 2 // component count
	}

	i2 |= uint16(totalSize - 3) // item size minus 3
	start := endOffset - totalSize

	binary.BigEndian.PutUint16(data[start:], i2)
	data[start+2] = byte(len(item.Key))
	copy(data[start+3:], item.Key)

	tagPos := start + 3 + len(item.Key)
	if !item.FirstComponent {
		binary.BigEndian.PutUint16(data[tagPos:], item.ComponentCount)
		tagPos += 2
	}
	copy(data[tagPos:], item.Tag)

	return start
}

// writeBranchItem encodes a branch item into the block data.
func (b *Block) writeBranchItem(data []byte, endOffset int, item BlockItem) int {
	totalSize := 4 + 1 + len(item.Key) + 2 // blockNumber + keyLen + key + componentCount
	start := endOffset - totalSize

	binary.BigEndian.PutUint32(data[start:], item.BlockNumber)
	data[start+4] = byte(len(item.Key))
	copy(data[start+5:], item.Key)
	binary.BigEndian.PutUint16(data[start+5+len(item.Key):], item.ComponentCount)

	return start
}

// FindItem returns the index of the first item with key >= the given key.
func (b *Block) FindItem(key []byte) int {
	for i, item := range b.Items {
		if bytes.Compare(item.Key, key) >= 0 {
			return i
		}
	}
	return len(b.Items)
}

// DecompressTag decompresses a compressed tag using zlib.
func DecompressTag(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("glass: decompressing tag: %w", err)
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// CompressTag compresses a tag using zlib.
func CompressTag(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("glass: compressing tag: %w", err)
	}
	writer.Close()
	return buf.Bytes(), nil
}

// Ensure encoding/binary and other imports are used.
var _ = binary.BigEndian
