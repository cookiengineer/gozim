package lzma

import "errors"

// MatchAlgorithm identifies an algorithm to find matches in the
// dictionary.
type MatchAlgorithm byte

// Supported matcher algorithms.
const (
	AlgorithmHashTable  MatchAlgorithm = iota
	AlgorithmBinaryTree
)

// maStrings are used by the String method.
var maStrings = map[MatchAlgorithm]string{
	AlgorithmHashTable:  "HashTable",
	AlgorithmBinaryTree: "BinaryTree",
}

// String returns a string representation of the Matcher.
func (a MatchAlgorithm) String() string {
	if s, ok := maStrings[a]; ok {
		return s
	}
	return "unknown"
}

var errUnsupportedMatchAlgorithm = errors.New(
	"lzma: unsupported match algorithm value")

// verify checks whether the matcher value is supported.
func (a MatchAlgorithm) verify() error {
	if _, ok := maStrings[a]; !ok {
		return errUnsupportedMatchAlgorithm
	}
	return nil
}

func (a MatchAlgorithm) new(dictCap int) (m matcher, err error) {
	switch a {
	case AlgorithmHashTable:
		return NewHashTable(dictCap, 4)
	case AlgorithmBinaryTree:
		return NewBinaryTree(dictCap)
	}
	return nil, errUnsupportedMatchAlgorithm
}
