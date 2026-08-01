package glass

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// RootInfo describes the root block and structure of a B-tree table.
type RootInfo struct {
	Root       uint32
	Level      byte
	NumEntries uint32
	BlockSize  uint32
	CompressMin int
	FreeList   []byte
}

// VersionInfo holds the database header information.
type VersionInfo struct {
	UUID                 [16]byte
	Revision             uint32
	RootInfos            [numTables]RootInfo
	DocCount             uint32
	LastDocID            uint32
	DocLenLBound         uint32
	WDFUBound            uint32
	DocLenUBound         uint32
	OldestChangeset      uint32
	TotalDocLen          uint64
	SpellingWordFreqUBound uint32
}

// ReadVersion reads the version header from bytes.
func ReadVersion(data []byte) (*VersionInfo, int, error) {
	if len(data) < len(VersionMagic)+2 {
		return nil, 0, fmt.Errorf("glass: version data too short for magic")
	}

	if !bytes.Equal(data[:len(VersionMagic)], VersionMagic) {
		return nil, 0, fmt.Errorf("glass: invalid version magic")
	}

	offset := len(VersionMagic)

	// Format version (2 bytes, little-endian).
	fmtVersion := binary.LittleEndian.Uint16(data[offset:])
	offset += 2
	_ = fmtVersion

	v := &VersionInfo{}

	// UUID.
	if len(data) < offset+16 {
		return nil, 0, fmt.Errorf("glass: version data too short for UUID")
	}
	copy(v.UUID[:], data[offset:offset+16])
	offset += 16

	// Revision.
	rev, n, err := unpackUint(data[offset:])
	if err != nil {
		return nil, 0, err
	}
	v.Revision = uint32(rev)
	offset += n

	// Root info for each table.
	for i := 0; i < numTables; i++ {
		root, n1, err := unpackUint(data[offset:])
		if err != nil {
			return nil, 0, err
		}
		offset += n1

		levelFlags, n2, err := unpackUint(data[offset:])
		if err != nil {
			return nil, 0, err
		}
		offset += n2

		numEntries, n3, err := unpackUint(data[offset:])
		if err != nil {
			return nil, 0, err
		}
		offset += n3

		blockSizeShift, n4, err := unpackUint(data[offset:])
		if err != nil {
			return nil, 0, err
		}
		offset += n4

		compressMin, n5, err := unpackUint(data[offset:])
		if err != nil {
			return nil, 0, err
		}
		offset += n5

		// Read freelist.
		freeLen, n6, err := unpackUint(data[offset:])
		if err != nil {
			return nil, 0, err
		}
		offset += n6

		var freeData []byte
		if freeLen > 0 && offset+int(freeLen) <= len(data) {
			freeData = make([]byte, freeLen)
			copy(freeData, data[offset:offset+int(freeLen)])
			offset += int(freeLen)
		}

		v.RootInfos[i] = RootInfo{
			Root:        uint32(root),
			Level:       byte(levelFlags >> 2),
			NumEntries:  uint32(numEntries),
			BlockSize:   uint32(blockSizeShift << 11),
			CompressMin: int(compressMin),
			FreeList:    freeData,
		}
	}

	// Database statistics.
	dbStats := []struct {
		field *uint32
	}{
		{&v.DocCount},
		{&v.LastDocID},
		{&v.DocLenLBound},
		{&v.WDFUBound},
		{&v.DocLenUBound},
		{&v.OldestChangeset},
	}

	for _, s := range dbStats {
		val, n, err := unpackUint(data[offset:])
		if err != nil {
			return nil, 0, err
		}
		offset += n
		*s.field = uint32(val)
	}

	// TotalDocLen.
	totalDocLen, n, err := unpackUint(data[offset:])
	if err != nil {
		return nil, 0, err
	}
	v.TotalDocLen = totalDocLen
	offset += n

	// SpellingWordFreqUBound.
	spellBound, n, err := unpackUint(data[offset:])
	if err != nil {
		return nil, 0, err
	}
	v.SpellingWordFreqUBound = uint32(spellBound)
	offset += n

	// For single-file databases: tables start right after the version header.
	// The offset returned here points to the start of postlist table data.

	return v, offset, nil
}

// EncodeVersion serializes the version info to bytes.
func EncodeVersion(v *VersionInfo) []byte {
	var buf bytes.Buffer

	buf.Write(VersionMagic)
	binary.Write(&buf, binary.LittleEndian, uint16(FormatVersion))
	buf.Write(v.UUID[:])
	buf.Write(packUint(uint64(v.Revision)))

	for _, ri := range v.RootInfos {
		buf.Write(packUint(uint64(ri.Root)))
		buf.Write(packUint(uint64(int(ri.Level)<<2)))
		buf.Write(packUint(uint64(ri.NumEntries)))
		buf.Write(packUint(uint64(ri.BlockSize >> 11)))
		buf.Write(packUint(uint64(ri.CompressMin)))

		buf.Write(packUint(uint64(len(ri.FreeList))))
		buf.Write(ri.FreeList)
	}

	buf.Write(packUint(uint64(v.DocCount)))
	buf.Write(packUint(uint64(v.LastDocID)))
	buf.Write(packUint(uint64(v.DocLenLBound)))
	buf.Write(packUint(uint64(v.WDFUBound)))
	buf.Write(packUint(uint64(v.DocLenUBound)))
	buf.Write(packUint(uint64(v.OldestChangeset)))
	buf.Write(packUint(uint64(v.TotalDocLen)))
	buf.Write(packUint(uint64(v.SpellingWordFreqUBound)))

	return buf.Bytes()
}
