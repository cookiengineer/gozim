# ZIM File Format Specification

Based on the libzim reference implementation (v9.x) and the openZIM format specification. Verified against real-world ZIM files produced by libzim, mwoffliner, and zim-tools.

## Magic Number

```
0x044d495a  ("ZIM" in little-endian with leading 0x04)
```

Byte representation in file: `5A 49 4D 04`

## File Header (80 bytes)

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| 0 | 4 | magicNumber | `0x044d495a` |
| 4 | 2 | majorVersion | 5 (old) or 6 (current) |
| 6 | 2 | minorVersion | 0-3. Minor >= 1 = new namespace scheme |
| 8 | 16 | uuid | 16-byte UUID |
| 24 | 4 | entryCount | Total number of directory entries |
| 28 | 4 | clusterCount | Total number of clusters |
| 32 | 8 | urlPtrPos | Offset of URL pointer list |
| 40 | 8 | titleIdxPos | Offset of title index (-1 if not present) |
| 48 | 8 | clusterPtrPos | Offset of cluster pointer list |
| 56 | 8 | mimeListPos | Offset of MIME type list (normally 80) |
| 64 | 4 | mainPage | Entry index of main page (0xFFFFFFFF = none) |
| 68 | 4 | layoutPage | Entry index of layout page (0xFFFFFFFF = none) |
| 72 | 8 | checksumPos | Offset of MD5 checksum (16 bytes) |

- `hasChecksum()` = `mimeListPos >= 80 && checksumPos > 0`
- `hasNewNamespaceScheme()` = `majorVersion >= 6 && minorVersion >= 1`
- `hasTitleIndex()` = `titleIdxPos != 0xFFFFFFFFFFFFFFFF`
- `hasMainPage()` = `mainPage != 0xFFFFFFFF`

## Namespace Schemes

### Old Scheme (minor version 0, or major version 5)
| Namespace | Purpose |
|-----------|---------|
| `A` | Articles (user content, text/html) |
| `I` | Images (binaries) |
| `J` | Scripts |
| `-` | Layout (CSS, templates) |
| `M` | Metadata |
| `X` | Indexes (fulltext, title listing, etc.) |
| `W` | Redirect/category listings (used in both schemes) |
| `Z` | (deprecated) |

Entry paths include the namespace prefix: `A/article.html`, `I/logo.png`
But in the dirent struct, the path field is `article.html` and the namespace field is `A`.

### New Scheme (minor version >= 1, major version 6)
| Namespace | Purpose |
|-----------|---------|
| `C` | Content (all user entries) |
| `M` | Metadata |
| `X` | Indexes |
| `W` | Redirects / category listings |

Entry paths do NOT include the namespace: `article.html`, `assets/logo.png`
The path stored in the dirent is just the bare path without namespace prefix.

## MIME Type List

Located at `mimeListPos` (offset 80 in modern files). A null-terminated list of MIME type strings. Each directory entry stores a 16-bit index into this list.

Special indices (sentinel values, not real MIME list entries):
- `0xFFFF` (65535) = redirect entry
- `0xFFFE` (65534) = linktarget entry
- `0xFFFD` (65533) = deleted entry

## URL Pointer List

Located at `urlPtrPos`. For each entry, an 8-byte little-endian offset pointing to the directory entry data.

**Critical**: Entries are sorted by **namespace + path**, not by path alone. This means the sort key is effectively `namespace_byte + "/" + path`. Content entries (C/A) come first, then metadata (M), then indexes (X), then others (W). When performing binary search, you must compare using the combined namespace+path key.

## Title Index

### v0 (at titleIdxPos)
If `titleIdxPos != -1`: For each entry, a 4-byte little-endian entry index into the URL pointer list. Sorted by namespace + title.

### v1 (entry X/listing/titleOrdered/v1)
For new namespace scheme. An uncompressed blob containing a list of 4-byte entry indices, sorted by title. Only entries marked as FRONT_ARTICLE are included. May be empty (0 bytes).

Also present in some old-namespace files as `X/listing/titleOrdered/v0`.

## Cluster Pointer List

Located at `clusterPtrPos`. For each cluster, an 8-byte little-endian offset pointing to the cluster data.

## Cluster Format (Corrected)

This section was revised after integration testing against real ZIM files.

### Cluster Layout — Two Distinct Formats

The layout depends on the compression type. This was a critical debugging finding.

#### Uncompressed Clusters (Compression = None, type 1)

```
[1 byte: clusterInfo]
[offset table: (N+1) entries of 4 or 8 bytes each]
[blob data: raw bytes of N concatenated blobs]
```

- `offset[0] = (N+1) * offsetSize` — does NOT include the info byte
- `numOffsets = offset[0] / offsetSize` (which equals N+1 for valid clusters)
- `numBlobs = numOffsets - 1`
- Blob data starts at `1 + offset[0]` bytes from cluster start
- For each blob i: blob-relative `start = offset[i] - offset[0]`, `end = offset[i+1] - offset[0]`

**Example**: Single PNG image (1 blob, 4-byte offsets):
- `offset[0] = 2 * 4 = 8`
- `offset[1] = 8 + image_size`
- `numOffsets = 8 / 4 = 2 → 1 blob`

#### Compressed Clusters (Compression = Zstd/Lzma, types 4/5)

```
[1 byte: clusterInfo]
[compressed stream — decompressed it yields:]
    [offset table: (N+1) entries of 4 or 8 bytes each]
    [blob data: raw bytes of N concatenated blobs]
```

- The raw bytes after the info byte are compressed data (starts with the compression format's magic: `0xFD2FB528` for zstd, or `0x............` for LZMA)
- Decompress the entire stream first, THEN parse the offset table
- `offset[0] = (N+1) * offsetSize` within the decompressed data
- `numOffsets = offset[0] / offsetSize` within the decompressed data
- Blob data starts at `offset[0]` bytes into the decompressed data
- For each blob i: blob-relative `start = offset[i] - offset[0]`, `end = offset[i+1] - offset[0]`

**Common mistake**: Trying to parse the offset table from the raw compressed bytes at `clusterBytes[1:]`. The first 4 bytes of a zstd-compressed cluster are the zstd magic `0xFD2FB528`, not a valid offset table size.

### Cluster Info Byte

| Bits | Description |
|------|-------------|
| 3-0 (low nibble) | Compression type: 1=None, 2=Zip (deprecated), 3=Bzip2 (deprecated), 4=Lzma, 5=Zstd |
| 4 (0x10) | Extended flag: if set, offsets are 8 bytes; otherwise 4 bytes |

### Minimum Cluster Size

A cluster with 1 blob and 4-byte offsets is `1 + 8 + blob_size` bytes uncompressed (info byte + 2 offsets × 4 + blob data).

## Directory Entry (Dirent)

Variable-length format:

```
[2 bytes: mime type index]
[1 byte: extra parameter length]
[1 byte: namespace character]
[4 bytes: version]

If redirect (mime index == 0xFFFF):
    [4 bytes: redirect entry index]

If article/linktarget/deleted:
    [4 bytes: cluster number]
    [4 bytes: blob number (or 0xFFFFFFFF for linktarget/deleted)]

[null-terminated string: path (URL)]
[null-terminated string: title]
[extra parameter bytes (length from byte 3)]
```

- Path and title are UTF-8 strings.
- Extra parameters are binary data with semantics defined by the ZIM creator.
- Total dirent size is determined by reading until both null terminators and extra data are consumed.
- In old namespace scheme: the path in the dirent includes the namespace prefix (e.g., `A/index.html`).
- In new namespace scheme: the path does NOT include the namespace (e.g., `index.html`).

## Checksum

At `checksumPos`: 16 bytes of MD5 hash. The hash covers all file content up to (but not including) the checksum itself.

**Note**: `HasChecksum()` returning true means the file has a checksum position in the header. A file may have an all-zero stored checksum (like `bad_checksum.zim`) which means the checksum was not computed — `CheckFile()` will correctly report the mismatch.

## Internal ZIM Entries

Entries at specific paths within namespaces `M` and `X`:

| Namespace | Path | Purpose |
|-----------|------|---------|
| M | Counter | MIME type counter metadata |
| M | Title | Archive title |
| M | Language | Language code |
| M | Description | Archive description |
| M | Date | Creation date |
| M | Creator | Creator/author |
| M | Publisher | Publisher |
| M | Name | Archive name |
| M | Tags | Comma-separated tags |
| M | Scraper | Scraper tool identifier |
| M | Illustration_48x48@1 | Favicon (48×48 PNG) |
| X | listing/titleOrdered/v0 | Title index (v0 style) |
| X | listing/titleOrdered/v1 | Title index (v1) |
| X | fulltext/xapian | Xapian full-text search database |
| X | title/xapian | Xapian title suggestion database |
| W | mainPage | Redirect to main page (both schemes) |

## Multipart Files

ZIM files exceeding a size threshold (typically 2GB) can be split into multiple parts:
- `archive.zimaa` (or `archive.zim` + auto-detected `.zimaa`)
- `archive.zimab`, `archive.zimac`, etc.

Parts are concatenated logically. The header in the first part describes the total logical file. Cluster boundaries are preserved (a cluster is never split across parts).

## Compression Decision

Items are compressed or stored uncompressed based on:
1. `COMPRESS`/`NO_COMPRESS` hints from the item
2. Default heuristic: text-based MIME types are compressed, binary MIME types are not

Compressible MIME types include: `text/*`, `application/xml`, `application/json`, `application/javascript`, `image/svg+xml`, `application/xhtml+xml`.

## FRONT_ARTICLE

Entries marked as `FRONT_ARTICLE` are included in:
- Title index (v1)
- Article count
- Random article selection

In old namespace scheme: all entries in namespace `A`.
In new namespace scheme: entries with the `FRONT_ARTICLE` hint set during creation.

## Version Compatibility Matrix

| Feature | v5 | v6 min0 | v6 min1+ |
|---------|-----|---------|-----------|
| Namespace scheme | Old (A/I/J/-/M/X/W) | Old | New (C/M/X/W) |
| Title index | v0 (titleIdxPos) | v0 | v1 (entry-based) |
| Checksum | No | No | Yes (mimeListPos >= 80) |
| Read compression | None, Lzma | None, Lzma, Zstd | None, Lzma, Zstd |
| Write compression | N/A | N/A | None, Zstd |
| Cluster offsets | 32-bit | 32/64-bit | 32/64-bit |
| Path in dirent | Includes namespace prefix | Includes namespace prefix | No prefix |
