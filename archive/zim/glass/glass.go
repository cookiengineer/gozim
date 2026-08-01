package glass

const (
	// BlockSize is the default B-tree block size in bytes.
	BlockSize = 8192

	// DirStart is the offset where the block directory begins.
	DirStart = 11

	// MaxKeyLen is the maximum key length in a B-tree block.
	MaxKeyLen = 255

	// MaxLevels is the maximum B-tree depth.
	MaxLevels = 10

	// LevelFreelist is the block level byte that identifies a freelist block.
	LevelFreelist = 254

	// BlockCapacity is the minimum number of maximum-size items per block.
	BlockCapacity = 4

	// ChunkSize is the target chunk size for posting list chunks in bytes.
	ChunkSize = 2000

	// FormatVersion is the database format version (DATE_TO_VERSION(2016,03,14)).
	FormatVersion = 0x046E
)

// VersionMagic is the magic byte sequence at the start of a Glass version file.
var VersionMagic = []byte{
	0x0f, 0x0d, 'X', 'a', 'p', 'i', 'a', 'n', ' ', 'G', 'l', 'a', 's', 's',
}

// TableType identifies which kind of table in the Glass database.
type TableType int

const (
	TablePostlist  TableType = 0
	TableDocdata   TableType = 1
	TableTermlist  TableType = 2 // Not used by libzim (DB_NO_TERMLIST)
	TablePosition  TableType = 3 // Not used by libzim
	TableSpelling  TableType = 4
	TableSynonym   TableType = 5 // Not used by libzim
)

const numTables = 6

// Item flags for leaf items.
const (
	ItemCompressedBit = 0x8000
	ItemLastBit       = 0x4000
	ItemFirstBit      = 0x2000
)

// RootInfo flags.
const (
	RootFakeBit        = 0x01
	RootSequentialBit  = 0x02
)

// Metadata keys used by libzim.
const (
	MetaValuesmap = "valuesmap"
	MetaKind      = "kind"
	MetaData      = "data"
	MetaLanguage  = "language"
	MetaStopwords = "stopwords"
)

// Database flags.
const (
	DBNoTermlist = 1 << iota
	DBCreateOrOverwrite
)

// Block number sentinel for non-existent blocks.
const blockNone = 0
