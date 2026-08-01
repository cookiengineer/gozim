package zim

import "encoding/binary"

// putUint16 writes a uint16 in little-endian byte order to buf at the given offset.
func putUint16(buf []byte, offset int, value uint16) {
	binary.LittleEndian.PutUint16(buf[offset:offset+2], value)
}

// putUint32 writes a uint32 in little-endian byte order to buf at the given offset.
func putUint32(buf []byte, offset int, value uint32) {
	binary.LittleEndian.PutUint32(buf[offset:offset+4], value)
}

// putUint64 writes a uint64 in little-endian byte order to buf at the given offset.
func putUint64(buf []byte, offset int, value uint64) {
	binary.LittleEndian.PutUint64(buf[offset:offset+8], value)
}

// readUint16 reads a uint16 in little-endian byte order from buf.
func readUint16(buf []byte) uint16 {
	return binary.LittleEndian.Uint16(buf[:2])
}

// readUint32 reads a uint32 in little-endian byte order from buf.
func readUint32(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf[:4])
}

// readUint64 reads a uint64 in little-endian byte order from buf.
func readUint64(buf []byte) uint64 {
	return binary.LittleEndian.Uint64(buf[:8])
}
