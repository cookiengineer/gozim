# Xapian Glass Backend Specification

The Glass backend is the default Xapian database backend used by libzim for full-text indexing within ZIM files. This document describes the binary format needed to read and write Glass databases in pure Go.

## Database File Structure

libzim stores the database as a **single-file compacted blob** at ZIM path `X/fulltext/xapian` (or `X/title/xapian` for title suggestions).

### Single-File Format

```
[Version header (variable)]
[Postlist table data (B-tree blocks)]
[Docdata table data (B-tree blocks)]
```

Tables **NOT present** in libzim's output:
- **Termlist** — disabled via `DB_NO_TERMLIST`
- **Position** — no position data stored (`index_text_without_positions`)
- **Spelling** — minimal use
- **Synonym** — not used by libzim

## Implementation Status

The `archive/zim/glass/` package implements the full Glass backend with read and write support. Actual files:

| File | Purpose |
|------|---------|
| `doc.go` | Package documentation |
| `glass.go` | Constants: block size (8192), magic, table types, item flags, metadata keys |
| `pack.go` | Encoding primitives: VLQ varint, sort-preserving uint/string, boolean |
| `block.go` | B-tree block: header (revision, level, free space), directory, leaf/branch items. zlib compress/decompress |
| `btree.go` | B-tree: insert with splitting, get, cursor for sequential access |
| `table.go` | Table: wraps B-tree with table-type-specific compression settings. Serialize/Deserialize |
| `postlist.go` | Posting list: first chunk (termfreq, collfreq, delta-encoded postings), continuation chunks, PostingIterator, BuildPostingList |
| `docdata.go` | Document data table (docid → path), ValuesMap parsing, Metadata struct (valuesmap, kind, data, language, stopwords), Document + TermEntry types |
| `version.go` | Version header: magic (0x0f 0x0d "Xapian Glass"), format version, UUID, 6 RootInfo entries, database statistics |
| `database.go` | High-level API: OpenDatabase, CreateDatabase, AddDocument, Commit, Compact |

## Version Header (iamglass)

### Byte layout

```
[14 bytes: magic]         ← \x0f\x0d Xapian Glass
[2 bytes: format_version]  ← DATE_TO_VERSION(2016,03,14) = 0x046E
[16 bytes: uuid]           ← Binary UUID (matches ZIM UUID)
[pack_uint: revision]
[6× RootInfo entries]      ← One per table type (postlist, docdata, termlist, position, spelling, synonym)
[pack_uint: doccount]
[pack_uint: last_docid - doccount]
[pack_uint: doclen_lbound]
[pack_uint: wdf_ubound]
[pack_uint: doclen_ubound - wdf_ubound]
[pack_uint: oldest_changeset]
[pack_uint: total_doclen]
[pack_uint: spelling_wordfreq_ubound]
```

### RootInfo per table

```
pack_uint(root_block_number)         // 0 if table is empty
pack_uint((level << 2) | flags)      // level + sequential (0x02) + root_is_fake (0x01)
pack_uint(num_entries)                // Total entries in table
pack_uint(blocksize >> 11)           // Block size / 2048
pack_uint(compress_min)              // 0 = no compression, >= 4 = min bytes for compression
pack_uint(freelist_length)            // Length of serialized freelist
[freelist_length bytes: freelist data]
```

### Table-specific compress_min

| Table | compress_min | Rationale |
|-------|-------------|-----------|
| Postlist | 0 | Posting list chunks should not be compressed (already delta-encoded) |
| Docdata | 18 | Compress document data strings if >= 18 bytes |
| Termlist | 18 | (Not used by libzim) |
| Position | 0 | (Not used by libzim) |
| Spelling | 18 | (Minimal use) |
| Synonym | 18 | (Not used by libzim) |

## Encoding Primitives (pack.go)

### pack_uint(value uint64) → []byte

Variable-length integer using 7 bits per byte with MSB as continuation bit:

```
while (value >= 128):
    output byte: (value & 0x7F) | 0x80
    value >>= 7
output byte: value
```

### pack_uint_preserving_sort(value uint64) → []byte

Encodes so that lexicographic byte comparison equals numeric comparison:

| Range | Bytes | First Byte Pattern |
|-------|-------|---------------------|
| 0x00000000 - 0x00007FFF | 2 | 0AAAAAAA BBBBBBBB |
| 0x00008000 - 0x003FFFFF | 3 | 10AAAAAA BBBBBBBB CCCCCCCC |
| 0x00400000 - 0x1FFFFFFF | 4 | 110AAAAA BBBBBBBB CCCCCCCC DDDDDDDD |
| up to max | 5-8 | Additional leading 1-bits |
| max range | 9 | 0xFE + 8-byte big-endian |

**Critical**: First byte is never 0xFF, making it safe to follow `pack_string_preserving_sort` output.

### pack_string_preserving_sort(s string) → []byte

Null bytes are escaped as `\x00\xff`. Terminated by bare `\x00`. Last string in sequence can omit terminator.

## B-tree Block Format

### Block Header (8192 bytes default)

```
Offset  Size  Field       Description
0       4     REVISION    Revision number (uint32, big-endian)
4       1     LEVEL       Tree level: 0=leaf, 1+=internal, 254=freelist
5       2     MAX_FREE    Max gap between directory end and first item
7       2     TOTAL_FREE  Total free space remaining
9       2     DIR_END     Offset to end of directory (always >= 11)
11      var   DIRECTORY   uint16 offsets for each item, grows forward
```

Items grow from the end toward the front. Directory grows from the front. Free space in between.

### Leaf Item Format

```
[2B: I2 = item_size - 3 with flags] [1B: key_len] [key bytes] [(2B: component_count if not first)] [tag bytes...]
```

I2 flags:
- `0x8000` (ItemCompressedBit): Tag is zlib-compressed
- `0x4000` (ItemLastBit): Last component chunk
- `0x2000` (ItemFirstBit): First component chunk

### Branch Item Format

```
[4B: block_number (big-endian)] [1B: key_len] [key bytes] [2B: component_count]
```

## Table-Specific Key/Value Formats

### Postlist Table

**Key formats**:
- Term posting list chunk: `pack_string_preserving_sort(term)` (terminated)
- Term continuation chunk: above + `pack_uint_preserving_sort(first_docid)`
- Document length: `\x00\xe0` + `pack_uint_preserving_sort(docid)`
- Metadata: `\x00\x00` + key name
- Value statistics: `\x00\xd8` + `pack_uint(slot)` + `pack_uint_preserving_sort(docid)`

**First chunk tag** (per term):
```
pack_uint(termfreq)            // Documents containing term
pack_uint(collfreq)            // Total term occurrences
pack_uint(first_docid - 1)     // First docid minus 1
pack_bool(is_last_chunk)
pack_uint(last_docid - first_docid)
[for each posting]:
  pack_uint(docid_delta - 1)   // delta-1 encoded
  pack_uint(wdf)               // within-document frequency
```

Chunk target size: ~2000 bytes.

### Docdata Table

**Key**: `pack_uint_preserving_sort(docid)`
**Value**: Document data string (e.g., `"C/path/to/article"`)
**Compression**: zlib when > 18 bytes

### Metadata (stored in Postlist table under `\x00\x00` prefix)

```
valuesmap → "title:0;wordcount:1;geo.position:2"
kind      → "fulltext" or "title"
data      → "fullPath"
language  → ISO-639 code (e.g., "eng")
stopwords → Stopword list
```

## B-tree Operations (Simplified)

### Insert
1. Find target leaf via binary search at each level
2. If leaf has space, insert item (sorted by key)
3. If full, split leaf into two halves
4. Insert separator key into parent
5. If parent full, recursively split up to root
6. If root splits, create new root

### Split
- Sort items by key
- Midpoint split: first half stays, second half goes to new block
- New block gets next available block number (MaxBlock++)

### Block Allocation
- New blocks appended at MaxBlock
- Freed blocks tracked in freelist (Level 254 block)
- Freelist checked before appending new blocks

## libzim's Compact Output

When `compact(DBCOMPACT_SINGLE_FILE | FULL)`:
1. New B-trees created for postlist and docdata only
2. Posting list chunks merged (target ~2000 bytes)
3. Docdata entries compacted
4. All tables serialized sequentially
5. Version header prepended with merged database statistics
