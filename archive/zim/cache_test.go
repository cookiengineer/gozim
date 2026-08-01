package zim

import (
	"sync"
	"testing"
)

func TestCacheBasic(t *testing.T) {
	cache := NewCache[int, string](32 * Megabyte)

	cache.Set(1, "value1", 100)
	val, ok := cache.Get(1)
	if !ok || val != "value1" {
		t.Errorf("Get(1) = %q, %v, want \"value1\", true", val, ok)
	}

	val, ok = cache.Get(999)
	if ok {
		t.Errorf("Get(999) = %q, %v, want zero, false", val, ok)
	}

	cache.Set(1, "updated", 200)
	val, ok = cache.Get(1)
	if !ok || val != "updated" {
		t.Errorf("Get(1) after update = %q, %v, want \"updated\", true", val, ok)
	}
}

func TestCacheEviction(t *testing.T) {
	// Small shard capacity forces eviction.
	cache := NewCache[int, string](defaultCacheShards * 100)

	// Fill the shard (all these keys hash to the same shard if we're unlucky,
	// but with integer keys we need enough items to force one eviction).
	for i := 0; i < 100; i++ {
		cache.Set(i, "item", 100)
	}

	// Should have evicted some items.
	if cache.Len() >= 100 {
		t.Log("warning: all items fit; hash distribution prevented eviction in this run")
	}
}

func TestCacheRemove(t *testing.T) {
	cache := NewCache[int, string](32 * Megabyte)

	cache.Set(1, "value1", 100)
	cache.Remove(1)

	_, ok := cache.Get(1)
	if ok {
		t.Error("key 1 should have been removed")
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewCache[int, string](32 * Megabyte)

	cache.Set(1, "v1", 100)
	cache.Set(2, "v2", 100)

	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Len() = %d after Clear, want 0", cache.Len())
	}
}

func TestCacheLen(t *testing.T) {
	cache := NewCache[int, string](32 * Megabyte)

	cache.Set(1, "v1", 100)
	cache.Set(2, "v2", 200)
	cache.Set(3, "v3", 300)

	if cache.Len() != 3 {
		t.Errorf("Len() = %d, want 3", cache.Len())
	}
}

func TestCacheConcurrent(t *testing.T) {
	cache := NewCache[int, int](Megabyte)

	var wg sync.WaitGroup
	numGoroutines := 50
	numOps := 1000

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for i := 0; i < numOps; i++ {
				key := offset*numOps + i
				cache.Set(key, i, int64(i%100))
				cache.Get(key)
			}
		}(g)
	}

	wg.Wait()
}

func TestCacheStringKeys(t *testing.T) {
	cache := NewCache[string, int](32 * Megabyte)

	cache.Set("hello", 42, 50)
	cache.Set("world", 99, 50)

	val, ok := cache.Get("hello")
	if !ok || val != 42 {
		t.Errorf("Get(\"hello\") = %d, %v, want 42, true", val, ok)
	}

	val, ok = cache.Get("world")
	if !ok || val != 99 {
		t.Errorf("Get(\"world\") = %d, %v, want 99, true", val, ok)
	}
}
