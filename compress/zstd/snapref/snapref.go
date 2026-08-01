package snapref

import "errors"

var ErrCorrupt = errors.New("snappy: corrupt input")

func DecodedLen(src []byte) (int, error) {
	return 0, errors.New("snappy: not supported in pure Go zstd port")
}

func Decode(dst, src []byte) ([]byte, error) {
	return nil, errors.New("snappy: snappy-encoded zstd blocks are not supported in this pure Go port")
}
