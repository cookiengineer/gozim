package glass

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"
)

func TestPackUint_RoundTrip(t *testing.T) {
	values := []uint64{0, 1, 127, 128, 255, 256, 16383, 16384, 65535,
		1<<20 - 1, 1 << 20, 1<<30 - 1, 1 << 30, 1<<40 - 1, 1 << 40,
		0xFFFFFFFFFFFFFFFF}
	for _, v := range values {
		t.Run(fmt.Sprintf("val=%d", v), func(t *testing.T) {
			encoded := packUint(v)
			decoded, n, err := unpackUint(encoded)
			if err != nil {
				t.Fatalf("unpackUint error: %v", err)
			}
			if decoded != v {
				t.Errorf("round-trip failed: got %d, want %d", decoded, v)
			}
			if n != len(encoded) {
				t.Errorf("byte count mismatch: %d != %d", n, len(encoded))
			}
		})
	}
}

func TestPackUintPreservingSort_RoundTrip(t *testing.T) {
	values := []uint64{0, 1, 100, 0x7FFF, 0x8000, 0xFFFF, 0x10000, 0x3FFFFF, 0x400000,
		0xFFFFFF, 0x1FFFFFFF, 0x20000000, 0xFFFFFFFF, 0x100000000,
		1<<40 - 1, 1 << 40}
	for _, v := range values {
		t.Run(fmt.Sprintf("val=%d", v), func(t *testing.T) {
			encoded := packUintPreservingSort(v)
			decoded, n, err := unpackUintPreservingSort(encoded)
			if err != nil {
				t.Fatalf("unpackUintPreservingSort error: %v", err)
			}
			if decoded != v {
				t.Errorf("round-trip failed: got %d, want %d", decoded, v)
			}
			if n != len(encoded) {
				t.Errorf("byte count mismatch: %d != %d", n, len(encoded))
			}
		})
	}
}

func TestPackUintPreservingSort_Ordering(t *testing.T) {
	values := []uint64{0, 1, 10, 100, 0x8000, 0xFFFF, 0x10000, 0x3FFFFF, 0x400000, 0xFFFFFFFF}
	encoded := make([][]byte, len(values))
	for i, v := range values {
		encoded[i] = packUintPreservingSort(v)
	}
	for i := 1; i < len(encoded); i++ {
		if bytes.Compare(encoded[i-1], encoded[i]) >= 0 {
			t.Errorf("sort order violation: %d >= %d", values[i-1], values[i])
		}
	}
}

func TestPackStringPreservingSort_RoundTrip(t *testing.T) {
	strings := []string{"", "hello", "hello\x00world", "test\x00\x00data",
		"ABCD", "abc", "the quick brown fox", "C/article.html"}
	for _, s := range strings {
		t.Run(fmt.Sprintf("str=%q", s), func(t *testing.T) {
			encoded := packStringPreservingSortTerminated(s)
			decoded, n, err := unpackStringPreservingSort(encoded)
			if err != nil {
				t.Fatalf("unpackStringPreservingSort error: %v", err)
			}
			if decoded != s {
				t.Errorf("round-trip failed: got %q, want %q", decoded, s)
			}
			if n != len(encoded) {
				t.Errorf("byte count mismatch: %d != %d", n, len(encoded))
			}
		})
	}
}

func TestPackBool_RoundTrip(t *testing.T) {
	if unpackBool(packBool(true)) != true || unpackBool(packBool(false)) != false {
		t.Error("bool round-trip failed")
	}
}

func TestBlock_EncodeDecode_RoundTrip(t *testing.T) {
	original := NewBlock()
	original.Level = 0
	original.Revision = 1
	original.Items = []BlockItem{
		{Key: []byte("key1"), Tag: []byte("value1"), FirstComponent: true, LastComponent: true},
		{Key: []byte("key2"), Tag: []byte("value2_longer"), FirstComponent: true, LastComponent: true},
		{Key: []byte("key3"), Tag: []byte("value3"), Compressed: true, FirstComponent: true, LastComponent: true},
	}
	encoded := original.Encode()
	decoded, err := ReadBlock(encoded)
	if err != nil {
		t.Fatalf("ReadBlock error: %v", err)
	}
	if len(decoded.Items) != len(original.Items) {
		t.Fatalf("item count mismatch: %d != %d", len(decoded.Items), len(original.Items))
	}
	for i := range decoded.Items {
		if !bytes.Equal(decoded.Items[i].Key, original.Items[i].Key) {
			t.Errorf("item %d key mismatch", i)
		}
		if !bytes.Equal(decoded.Items[i].Tag, original.Items[i].Tag) {
			t.Errorf("item %d tag mismatch", i)
		}
	}
}

func TestBlock_Empty(t *testing.T) {
	b := NewBlock()
	encoded := b.Encode()
	decoded, err := ReadBlock(encoded)
	if err != nil {
		t.Fatalf("ReadBlock error: %v", err)
	}
	if len(decoded.Items) != 0 {
		t.Error("expected 0 items")
	}
}

func TestBlock_FindItem(t *testing.T) {
	b := NewBlock()
	b.Items = []BlockItem{{Key: []byte("aaa")}, {Key: []byte("bbb")}, {Key: []byte("ccc")}}
	tests := []struct{ key string; expected int }{
		{"aaa", 0}, {"aab", 1}, {"bbb", 1}, {"ccc", 2}, {"ddd", 3},
	}
	for _, tt := range tests {
		if idx := b.FindItem([]byte(tt.key)); idx != tt.expected {
			t.Errorf("FindItem(%q) = %d, want %d", tt.key, idx, tt.expected)
		}
	}
}

func TestBlock_CompressDecompressTag(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"small", []byte("hello world")},
		{"medium", bytes.Repeat([]byte("ABCDEFGH"), 100)},
		{"large", bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 50)},
		{"binary", bytes.Repeat([]byte{0xAA, 0xBB, 0xCC}, 64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := CompressTag(tt.data)
			if err != nil {
				t.Fatalf("CompressTag error: %v", err)
			}
			decompressed, err := DecompressTag(compressed)
			if err != nil {
				t.Fatalf("DecompressTag error: %v", err)
			}
			if !bytes.Equal(decompressed, tt.data) {
				t.Error("round-trip failed")
			}
		})
	}
}

func TestBlock_ReadTooShort(t *testing.T) {
	if _, err := ReadBlock(make([]byte, 5)); err == nil {
		t.Error("expected error")
	}
}

func TestBTree_SingleInsertGet(t *testing.T) {
	btree := &BTree{BlockSize: BlockSize, Blocks: make(map[uint32]*Block)}
	_, _ = btree.Insert([]byte("key"), []byte("value"))
	if btree.NumEntries != 1 {
		t.Errorf("NumEntries = %d, want 1", btree.NumEntries)
	}
	tag, ok, _ := btree.Get([]byte("key"))
	if !ok || !bytes.Equal(tag, []byte("value")) {
		t.Errorf("Get failed: ok=%v, tag=%q", ok, tag)
	}
}

func TestBTree_Update(t *testing.T) {
	btree := &BTree{BlockSize: BlockSize, Blocks: make(map[uint32]*Block)}
	btree.Insert([]byte("key"), []byte("original"))
	btree.Insert([]byte("key"), []byte("updated"))
	tag, ok, _ := btree.Get([]byte("key"))
	if !ok || !bytes.Equal(tag, []byte("updated")) {
		t.Errorf("update failed: %q", tag)
	}
	if btree.NumEntries != 1 {
		t.Errorf("NumEntries after update = %d, want 1", btree.NumEntries)
	}
}

func TestBTree_MultipleInsertGet(t *testing.T) {
	btree := &BTree{BlockSize: BlockSize, Blocks: make(map[uint32]*Block)}
	kv := map[string]string{"key1": "value1", "key2": "value2", "key3": "value3", "aaa": "first", "zzz": "last"}
	for k, v := range kv {
		if _, err := btree.Insert([]byte(k), []byte(v)); err != nil {
			t.Fatalf("Insert(%q) error: %v", k, err)
		}
	}
	if btree.NumEntries != uint32(len(kv)) {
		t.Errorf("NumEntries = %d, want %d", btree.NumEntries, len(kv))
	}
	for k, v := range kv {
		tag, ok, _ := btree.Get([]byte(k))
		if !ok || !bytes.Equal(tag, []byte(v)) {
			t.Errorf("Get(%q) = %q, want %q", k, tag, v)
		}
	}
}

// TestBTree_SplitNodes exercises enough inserts to trigger block splits,
// verifying all keys remain retrievable.
func TestBTree_SplitNodes(t *testing.T) {
	btree := &BTree{BlockSize: BlockSize, Blocks: make(map[uint32]*Block)}

	// Insert many items to force splits. With ~64 byte items and 8192-byte
	// blocks, this should create multiple leaf blocks and at least one branch.
	numEntries := 200
	keys := make([]string, numEntries)
	for i := 0; i < numEntries; i++ {
		keys[i] = fmt.Sprintf("key%08d", i)
	}

	// Insert in shuffled order to avoid sequential patterns.
	perm := make([]int, numEntries)
	for i := range perm {
		perm[i] = i
	}
	for i := numEntries - 1; i > 0; i-- {
		j := int(randByte() * uint64(i+1) / 256)
		perm[i], perm[j] = perm[j], perm[i]
	}

	for _, idx := range perm {
		key := []byte(keys[idx])
		value := []byte(fmt.Sprintf("value_%d", idx))
		if _, err := btree.Insert(key, value); err != nil {
			t.Fatalf("Insert error at entry %d: %v", btree.NumEntries, err)
		}
	}

	if btree.NumEntries != uint32(numEntries) {
		t.Errorf("NumEntries = %d, want %d", btree.NumEntries, numEntries)
	}
	if btree.Level == 0 {
		t.Log("all entries fit in one leaf (no splits needed)")
	}
	t.Logf("tree has %d levels after %d entries", btree.Level+1, numEntries)

	// Verify every key.
	for _, key := range keys {
		tag, ok, _ := btree.Get([]byte(key))
		if !ok {
			t.Errorf("key %q not found", key)
		}
		if tag == nil {
			t.Errorf("tag for key %q is nil", key)
		}
	}
}

// TestBTree_CursorFull verifies cursor traverses all items in sorted order
// even with splits.
func TestBTree_CursorFull(t *testing.T) {
	btree := &BTree{BlockSize: BlockSize, Blocks: make(map[uint32]*Block)}

	numEntries := 100
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("k%04d", i)
		btree.Insert([]byte(key), []byte("v"))
	}

	cursor, err := btree.NewCursor([]byte("k0000"))
	if err != nil {
		t.Fatalf("NewCursor error: %v", err)
	}

	count := 0
	var prev string
	for cursor.Valid() {
		cur := string(cursor.Key())
		if prev != "" && cur <= prev {
			t.Errorf("cursor out of order: %q then %q", prev, cur)
		}
		prev = cur
		count++
		cursor.Next()
	}
	if count != numEntries {
		t.Errorf("cursor returned %d entries, want %d", count, numEntries)
	}
}

// TestBTree_UpdateAfterSplit ensures updates work after the tree has split.
func TestBTree_UpdateAfterSplit(t *testing.T) {
	btree := &BTree{BlockSize: BlockSize, Blocks: make(map[uint32]*Block)}

	// Fill enough to cause splits.
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("key%08d", i)
		newEntry, _ := btree.Insert([]byte(key), []byte(fmt.Sprintf("orig_%d", i)))
		if !newEntry && i > 0 {
			t.Errorf("unexpected update at %d", i)
		}
	}

	// Update a key that was inserted early (in leftmost leaf).
	_, _ = btree.Insert([]byte("key00000000"), []byte("UPDATED"))
	tag, ok, _ := btree.Get([]byte("key00000000"))
	if !ok || !bytes.Equal(tag, []byte("UPDATED")) {
		t.Errorf("update after split failed: got %q", tag)
	}

	// Update a key in the middle.
	_, _ = btree.Insert([]byte("key00000100"), []byte("UPDATED_MID"))
	tag2, ok2, _ := btree.Get([]byte("key00000100"))
	if !ok2 || !bytes.Equal(tag2, []byte("UPDATED_MID")) {
		t.Errorf("mid update after split failed: got %q", tag2)
	}

	if btree.NumEntries != 200 {
		t.Errorf("NumEntries = %d, want 200", btree.NumEntries)
	}
}

func TestBTree_NotFound(t *testing.T) {
	btree := &BTree{BlockSize: BlockSize, Blocks: make(map[uint32]*Block)}
	btree.Insert([]byte("existing"), []byte("value"))
	if ok, _ := btreeGetOk(btree, []byte("nonexistent")); ok {
		t.Error("expected not found")
	}
}

func TestBTree_EmptyRoot(t *testing.T) {
	btree := &BTree{BlockSize: BlockSize, Blocks: make(map[uint32]*Block)}
	if ok, _ := btreeGetOk(btree, []byte("anything")); ok {
		t.Error("expected not found in empty tree")
	}
}

func btreeGetOk(bt *BTree, key []byte) (bool, error) {
	_, ok, err := bt.Get(key)
	return ok, err
}

func TestTable_InsertGet_RoundTrip(t *testing.T) {
	tbl := NewTable(TableDocdata)
	tbl.Insert([]byte("key1"), []byte("value1"))
	tbl.Insert([]byte("key2"), []byte("value2 with more content"))
	tbl.Insert([]byte("key3"), []byte("value3"))
	for _, k := range []string{"key1", "key2", "key3"} {
		tag, ok, err := tbl.Get([]byte(k))
		if err != nil {
			t.Fatalf("Get(%q) error: %v", k, err)
		}
		if !ok {
			t.Errorf("key %q not found", k)
		}
		if tag == nil {
			t.Errorf("tag for %q is nil", k)
		}
	}
}

func TestTable_SerializeDeserialize_RoundTrip(t *testing.T) {
	tbl := NewTable(TableDocdata)
	entries := map[string]string{"aaaa": "value_aaaa", "bbbb": "value_bbbb_longer", "cccc": "value_cccc"}
	for k, v := range entries {
		tbl.Insert([]byte(k), []byte(v))
	}
	root, level, numEntries := tbl.RootInfo()
	serialized, err := tbl.Serialize()
	if err != nil {
		t.Fatalf("Serialize error: %v", err)
	}
	if serialized == nil {
		t.Skip("no blocks to serialize")
	}
	t.Logf("serialized %d entries into %d blocks", numEntries, len(serialized)/BlockSize)
	deserialized, err := DeserializeTable(TableDocdata, root, level, numEntries, serialized, 18)
	if err != nil {
		t.Fatalf("DeserializeTable error: %v", err)
	}
	for k, v := range entries {
		tag, ok, err := deserialized.Get([]byte(k))
		if err != nil {
			t.Fatalf("Get(%q) error: %v", k, err)
		}
		if !ok || !bytes.Equal(tag, []byte(v)) {
			t.Errorf("key %q: got %q, want %q", k, tag, v)
		}
	}
}

func TestPosting_EncodeDecodeFirstChunk(t *testing.T) {
	postings := []Posting{
		{DocumentID: 1, WDF: 3}, {DocumentID: 3, WDF: 2},
		{DocumentID: 5, WDF: 4}, {DocumentID: 10, WDF: 1},
	}
	encoded, err := EncodeFirstChunk(4, 10, postings, true)
	if err != nil {
		t.Fatalf("EncodeFirstChunk error: %v", err)
	}
	chunk, _, err := DecodeFirstChunk(encoded)
	if err != nil {
		t.Fatalf("DecodeFirstChunk error: %v", err)
	}
	if chunk.TermFreq != 4 || chunk.CollFreq != 10 {
		t.Errorf("freq mismatch: %d/%d", chunk.TermFreq, chunk.CollFreq)
	}
	if chunk.FirstDocID != 1 || chunk.LastDocID != 10 {
		t.Errorf("docID range: [%d, %d]", chunk.FirstDocID, chunk.LastDocID)
	}
	if len(chunk.Postings) != len(postings) {
		t.Errorf("posting count = %d, want %d", len(chunk.Postings), len(postings))
	}
	for i := range chunk.Postings {
		if chunk.Postings[i].DocumentID != postings[i].DocumentID {
			t.Errorf("posting %d docid mismatch: %d != %d", i, chunk.Postings[i].DocumentID, postings[i].DocumentID)
		}
		if chunk.Postings[i].WDF != postings[i].WDF {
			t.Errorf("posting %d WDF mismatch: %d != %d", i, chunk.Postings[i].WDF, postings[i].WDF)
		}
	}
}

func TestPosting_EncodeDecodeSubsequentChunk(t *testing.T) {
	encoded, err := EncodeSubsequentChunk([]Posting{{DocumentID: 100, WDF: 1}}, true)
	if err != nil {
		t.Fatalf("EncodeSubsequentChunk error: %v", err)
	}
	if len(encoded) == 0 {
		t.Error("encoded should not be empty")
	}
}

func TestPosting_EncodeFirstChunk_Empty(t *testing.T) {
	if _, err := EncodeFirstChunk(0, 0, nil, true); err == nil {
		t.Error("expected error")
	}
}

func TestParseValuesMap(t *testing.T) {
	vm := ParseValuesMap("title:0;wordcount:1;geo.position:2")
	if len(vm.Entries) != 3 || vm.Entries["title"] != 0 || vm.Entries["wordcount"] != 1 || vm.Entries["geo.position"] != 2 {
		t.Errorf("ParseValuesMap failed: %+v", vm.Entries)
	}
}

func TestParseValuesMap_Empty(t *testing.T) {
	if len(ParseValuesMap("").Entries) != 0 {
		t.Error("expected 0 entries")
	}
}

func TestParseValuesMap_Invalid(t *testing.T) {
	if len(ParseValuesMap("title;wordcount:notanum").Entries) != 0 {
		t.Error("expected 0 entries")
	}
}

func TestValuesMap_Encode(t *testing.T) {
	vm := ParseValuesMap("title:0;wordcount:1;geo.position:2")
	encoded := vm.Encode()
	if encoded == "" {
		t.Error("Encode returned empty")
	}
	vm2 := ParseValuesMap(encoded)
	if len(vm2.Entries) != len(vm.Entries) {
		t.Errorf("re-decoded count mismatch: %d != %d", len(vm2.Entries), len(vm.Entries))
	}
}

func TestMetadata_WriteRead(t *testing.T) {
	tbl := NewTable(TablePostlist)
	meta := &Metadata{ValuesMap: "title:0;wordcount:1", Kind: "fulltext", Data: "data",
		Language: "eng", Stopwords: "the,is,at"}
	if err := WriteMetadata(tbl, meta); err != nil {
		t.Fatalf("WriteMetadata error: %v", err)
	}
	readMeta, _ := ReadMetadata(tbl)
	if readMeta.Kind != "fulltext" || readMeta.Language != "eng" || readMeta.ValuesMap != "title:0;wordcount:1" {
		t.Error("metadata round-trip failed")
	}
}

func TestMetadata_Empty(t *testing.T) {
	tbl := NewTable(TablePostlist)
	WriteMetadata(tbl, &Metadata{})
	rm, _ := ReadMetadata(tbl)
	if rm.ValuesMap != "" || rm.Kind != "" || rm.Language != "" {
		t.Error("expected empty metadata")
	}
}

func TestVersion_EncodeDecode_RoundTrip(t *testing.T) {
	original := &VersionInfo{Revision: 1, DocCount: 5, LastDocID: 5, TotalDocLen: 12345}
	rand.Read(original.UUID[:])
	for i := 0; i < numTables; i++ {
		original.RootInfos[i].Root = 1
		original.RootInfos[i].Level = byte(i % 3)
		original.RootInfos[i].NumEntries = uint32(i * 10)
		original.RootInfos[i].BlockSize = 8192
		original.RootInfos[i].CompressMin = 4
	}
	encoded := EncodeVersion(original)
	decoded, consumed, err := ReadVersion(encoded)
	if err != nil {
		t.Fatalf("ReadVersion error: %v", err)
	}
	if consumed != len(encoded) {
		t.Errorf("consumed %d != %d", consumed, len(encoded))
	}
	if decoded.UUID != original.UUID || decoded.Revision != original.Revision ||
		decoded.DocCount != original.DocCount || decoded.TotalDocLen != original.TotalDocLen {
		t.Error("version field mismatch")
	}
	for i := 0; i < numTables; i++ {
		if decoded.RootInfos[i].Root != original.RootInfos[i].Root ||
			decoded.RootInfos[i].Level != original.RootInfos[i].Level ||
			decoded.RootInfos[i].NumEntries != original.RootInfos[i].NumEntries {
			t.Errorf("RootInfos[%d] mismatch", i)
		}
	}
}

func TestVersion_InvalidMagic(t *testing.T) {
	if _, _, err := ReadVersion([]byte("invalid")); err == nil {
		t.Error("expected error")
	}
}

func TestVersion_TooShort(t *testing.T) {
	if _, _, err := ReadVersion(make([]byte, 5)); err == nil {
		t.Error("expected error")
	}
}

func TestDatabase_Create_Add_Compact_Open_SingleDoc(t *testing.T) {
	var uuid [16]byte
	rand.Read(uuid[:])
	db := CreateDatabase(uuid)

	doc := &Document{Data: "C/article.html", Terms: []TermEntry{{Term: "hello", WDF: 3}, {Term: "world", WDF: 2}}}
	id, err := db.AddDocument(doc)
	if err != nil {
		t.Fatalf("AddDocument error: %v", err)
	}
	if id != 1 {
		t.Errorf("expected docID=1, got %d", id)
	}
	if db.DocCount() != 1 {
		t.Errorf("DocCount = %d, want 1", db.DocCount())
	}

	retrieved, err := db.GetDocument(id)
	if err != nil {
		t.Fatalf("GetDocument error: %v", err)
	}
	if retrieved.Data != "C/article.html" {
		t.Errorf("Data = %q", retrieved.Data)
	}

	compacted, err := db.Compact()
	if err != nil {
		t.Fatalf("Compact error: %v", err)
	}
	t.Logf("compacted: %d bytes", len(compacted))

	reopened, err := OpenDatabase(compacted)
	if err != nil {
		t.Fatalf("OpenDatabase error: %v", err)
	}
	if reopened == nil {
		t.Fatal("reopened is nil")
	}
	if reopened.DocCount() != 1 {
		t.Errorf("reopened DocCount = %d", reopened.DocCount())
	}
	reopenedRetrieved, err := reopened.GetDocument(id)
	if err != nil {
		t.Fatalf("reopened GetDocument error: %v", err)
	}
	if reopenedRetrieved.Data != "C/article.html" {
		t.Errorf("reopened Data = %q", reopenedRetrieved.Data)
	}
}

func TestDatabase_Empty(t *testing.T) {
	var uuid [16]byte
	rand.Read(uuid[:])
	db := CreateDatabase(uuid)
	if db.DocCount() != 0 {
		t.Error("DocCount != 0")
	}
	compacted, _ := db.Compact()
	reopened, err := OpenDatabase(compacted)
	if err != nil {
		t.Fatalf("OpenDatabase error: %v", err)
	}
	if reopened.DocCount() != 0 {
		t.Error("reopened DocCount != 0")
	}
}

func TestDatabase_DocDataTableHelpers(t *testing.T) {
	tbl := NewTable(TableDocdata)
	DocDataTable_set(tbl, 42, "C/test.html")
	path, _ := DocDataTable_get(tbl, 42)
	if path != "C/test.html" {
		t.Errorf("got %q", path)
	}
}

func TestDatabase_Create_WithZeroUUID(t *testing.T) {
	db := CreateDatabase([16]byte{})
	compacted, _ := db.Compact()
	reopened, _ := OpenDatabase(compacted)
	if reopened == nil {
		t.Fatal("reopened is nil")
	}
}

func TestDatabase_Commit(t *testing.T) {
	var uuid [16]byte
	rand.Read(uuid[:])
	db := CreateDatabase(uuid)
	db.AddDocument(&Document{Data: "C/test.html", Terms: []TermEntry{{Term: "commit", WDF: 1}}})
	db.Commit()
	if db.version.DocCount != 1 {
		t.Errorf("version.DocCount = %d, want 1", db.version.DocCount)
	}
}

func TestDatabase_AddDocument_NilDocdata(t *testing.T) {
	db := &Database{postlist: NewTable(TablePostlist)}
	if _, err := db.AddDocument(&Document{Data: "test"}); err == nil {
		t.Error("expected error")
	}
}

func BenchmarkPackUint(b *testing.B) {
	for i := 0; i < b.N; i++ {
		packUint(1234567890)
	}
}

func BenchmarkUnpackUint(b *testing.B) {
	encoded := packUint(1234567890)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		unpackUint(encoded)
	}
}

func randByte() uint64 {
	var b [1]byte
	rand.Read(b[:])
	return uint64(b[0])
}

func FuzzPackUint(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(12345))
	f.Fuzz(func(t *testing.T, value uint64) {
		encoded := packUint(value)
		decoded, _, err := unpackUint(encoded)
		if err != nil {
			t.Fatalf("unpackUint error: %v", err)
		}
		if decoded != value {
			t.Fatalf("round-trip mismatch: %d != %d", decoded, value)
		}
	})
}

func FuzzPackUintPreservingSort(f *testing.F) {
	f.Add(uint64(0))
	f.Fuzz(func(t *testing.T, value uint64) {
		encoded := packUintPreservingSort(value)
		decoded, _, err := unpackUintPreservingSort(encoded)
		if err != nil {
			t.Fatalf("unpack error for %d: %v", value, err)
		}
		if decoded != value {
			t.Fatalf("round-trip mismatch: %d != %d", decoded, value)
		}
	})
}
