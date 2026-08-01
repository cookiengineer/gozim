package zim

import (
	"container/list"
	"sync"
)

const defaultCacheShards = 32

// Cache is a generic, sharded, thread-safe LRU cache.
// Keys must be comparable, values can be any type.
// Each shard has its own mutex to reduce contention.
type Cache[K comparable, V any] struct {
	shards    []*cacheShard[K, V]
	shardMask uint32
	maxSize   int64
}

type cacheEntry[V any] struct {
	key  string // Only used for string-keyed caches
	size int64
}

type cacheShard[K comparable, V any] struct {
	mu       sync.Mutex
	items    map[K]*list.Element
	order    *list.List
	size     int64
	maxSize  int64
}

// NewCache creates a new sharded LRU cache with the given total maximum size in bytes.
// The capacity is divided evenly across all shards.
func NewCache[K comparable, V any](maxSize int64) *Cache[K, V] {
	shardSize := maxSize / defaultCacheShards
	if shardSize < 1 {
		shardSize = 1
	}

	shards := make([]*cacheShard[K, V], defaultCacheShards)
	for i := range shards {
		shards[i] = &cacheShard[K, V]{
			items:   make(map[K]*list.Element),
			order:   list.New(),
			maxSize: shardSize,
		}
	}

	return &Cache[K, V]{
		shards:    shards,
		shardMask: defaultCacheShards - 1,
		maxSize:   maxSize,
	}
}

// Get retrieves a value from the cache. Returns the value and true if found.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	shard := c.shardForKey(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if elem, ok := shard.items[key]; ok {
		shard.order.MoveToFront(elem)
		entry := elem.Value.(*cacheItem[V])
		return entry.value, true
	}

	var zero V
	return zero, false
}

// Set stores a value in the cache with the given approximate size in bytes.
// If the cache is full, the least recently used entries are evicted.
func (c *Cache[K, V]) Set(key K, value V, size int64) {
	shard := c.shardForKey(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if elem, ok := shard.items[key]; ok {
		shard.order.MoveToFront(elem)
		entry := elem.Value.(*cacheItem[V])
		oldSize := entry.size
		entry.value = value
		entry.size = size
		shard.size += size - oldSize
	} else {
		item := &cacheItem[V]{value: value, size: size}
		elem := shard.order.PushFront(item)
		shard.items[key] = elem
		shard.size += size
	}

	for shard.size > shard.maxSize && shard.order.Len() > 0 {
		c.evictLocked(shard)
	}
}

// Remove deletes a key from the cache.
func (c *Cache[K, V]) Remove(key K) {
	shard := c.shardForKey(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if elem, ok := shard.items[key]; ok {
		shard.order.Remove(elem)
		delete(shard.items, key)
	}
}

// Len returns the total number of items in the cache.
func (c *Cache[K, V]) Len() int {
	total := 0
	for _, shard := range c.shards {
		shard.mu.Lock()
		total += len(shard.items)
		shard.mu.Unlock()
	}
	return total
}

// Clear removes all items from the cache.
func (c *Cache[K, V]) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		shard.items = make(map[K]*list.Element)
		shard.order.Init()
		shard.size = 0
		shard.mu.Unlock()
	}
}

func (c *Cache[K, V]) shardForKey(key K) *cacheShard[K, V] {
	hash := hashKey(key)
	return c.shards[hash&uint32(c.shardMask)]
}

func (c *Cache[K, V]) evictLocked(shard *cacheShard[K, V]) {
	elem := shard.order.Back()
	if elem == nil {
		return
	}

	entry := elem.Value.(*cacheItem[V])
	shard.order.Remove(elem)

	// Find the matching key and remove it from the map.
	// This requires scanning since we used the value object's key.
	for k, e := range shard.items {
		if e == elem {
			shard.size -= entry.size
			delete(shard.items, k)
			return
		}
	}
}

type cacheItem[V any] struct {
	value V
	size  int64
}

// hashKey computes a hash for a comparable key.
func hashKey[K comparable](key K) uint32 {
	switch k := any(key).(type) {
	case int:
		return uint32(k) * 2654435761
	case uint32:
		return k * 2654435761
	case int32:
		return uint32(k) * 2654435761
	case uint64:
		return uint32(k ^ (k >> 32))
	case string:
		return fnv32(k)
	default:
		// Use Go's internal string representation for unknown types.
		return 0
	}
}

func fnv32(s string) uint32 {
	var hash uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= 16777619
	}
	return hash
}
