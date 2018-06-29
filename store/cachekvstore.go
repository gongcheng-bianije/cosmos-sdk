package store

import (
	"bytes"
	"sort"
	"sync"

	cmn "github.com/tendermint/tendermint/libs/common"
)

var _ CacheKVStore = (*cacheKVStore)(nil)

type cacheKVStore struct {
	*cache
	transient *cache
}

// nolint
func NewCacheKVStore(parent KVStore) CacheKVStore {
	ci := &cache{
		cache:  make(map[string]cValue),
		parent: parent,
	}
	return &cacheKVStore{ci, nil}
}

func (ci *cacheKVStore) Transient() KVStore {
	if ci.transient == nil {
		ci.transient = &cache{
			cache:  make(map[string]cValue),
			parent: ci.parent.Transient(),
		}
	}
	return ci.transient
}

func (ci *cacheKVStore) Write() {
	ci.cache.Write()
	if ci.transient != nil {
		ci.transient.Write()
	}
}

var _ CacheKVStore = (*cache)(nil)

// If value is nil but deleted is false, it means the parent doesn't have the
// key.  (No need to delete upon Write())
type cValue struct {
	value   []byte
	deleted bool
	dirty   bool
}

// cache wraps an in-memory cache around an underlying KVStore.
type cache struct {
	mtx    sync.Mutex
	cache  map[string]cValue
	parent KVStore
}

// Implements Store.
func (ci *cache) GetStoreType() StoreType {
	return ci.parent.GetStoreType()
}

// Implements KVStore.
func (ci *cache) Get(key []byte) (value []byte) {
	ci.mtx.Lock()
	defer ci.mtx.Unlock()
	ci.assertValidKey(key)

	cacheValue, ok := ci.cache[string(key)]
	if !ok {
		value = ci.parent.Get(key)
		ci.setCacheValue(key, value, false, false)
	} else {
		value = cacheValue.value
	}

	return value
}

// Implements KVStore.
func (ci *cache) Set(key []byte, value []byte) {
	ci.mtx.Lock()
	defer ci.mtx.Unlock()
	ci.assertValidKey(key)

	ci.setCacheValue(key, value, false, true)
}

// Implements KVStore.
func (ci *cache) Has(key []byte) bool {
	value := ci.Get(key)
	return value != nil
}

// Implements KVStore.
func (ci *cache) Delete(key []byte) {
	ci.mtx.Lock()
	defer ci.mtx.Unlock()
	ci.assertValidKey(key)

	ci.setCacheValue(key, nil, true, true)
}

// Implements KVStore
func (ci *cache) Prefix(prefix []byte) KVStore {
	return prefixStore{ci, prefix}
}

// Implements KVStore
func (ci *cache) Gas(meter GasMeter, config GasConfig) KVStore {
	return NewGasKVStore(meter, config, ci)
}

// Implements KVStore
func (ci *cache) Transient() KVStore {
	panic("Transient() called on cache")
}

// Implements CacheKVStore.
func (ci *cache) Write() {
	ci.mtx.Lock()
	defer ci.mtx.Unlock()

	// We need a copy of all of the keys.
	// Not the best, but probably not a bottleneck depending.
	keys := make([]string, 0, len(ci.cache))
	for key, dbValue := range ci.cache {
		if dbValue.dirty {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	// TODO: Consider allowing usage of Batch, which would allow the write to
	// at least happen atomically.
	for _, key := range keys {
		cacheValue := ci.cache[key]
		if cacheValue.deleted {
			ci.parent.Delete([]byte(key))
		} else if cacheValue.value == nil {
			// Skip, it already doesn't exist in parent.
		} else {
			ci.parent.Set([]byte(key), cacheValue.value)
		}
	}

	// Clear the cache
	ci.cache = make(map[string]cValue)
}

//----------------------------------------
// To cache-wrap this cache further.

// Implements CacheWrapper.
func (ci *cache) CacheWrap() CacheWrap {
	return NewCacheKVStore(ci)
}

//----------------------------------------
// Iteration

// Implements KVStore.
func (ci *cache) Iterator(start, end []byte) Iterator {
	return ci.iterator(start, end, true)
}

// Implements KVStore.
func (ci *cache) ReverseIterator(start, end []byte) Iterator {
	return ci.iterator(start, end, false)
}

func (ci *cache) iterator(start, end []byte, ascending bool) Iterator {
	var parent, cache Iterator
	if ascending {
		parent = ci.parent.Iterator(start, end)
	} else {
		parent = ci.parent.ReverseIterator(start, end)
	}
	items := ci.dirtyItems(ascending)
	cache = newMemIterator(start, end, items)
	return newCacheMergeIterator(parent, cache, ascending)
}

// Constructs a slice of dirty items, to use w/ memIterator.
func (ci *cache) dirtyItems(ascending bool) []cmn.KVPair {
	items := make([]cmn.KVPair, 0, len(ci.cache))
	for key, cacheValue := range ci.cache {
		if !cacheValue.dirty {
			continue
		}
		items = append(items,
			cmn.KVPair{[]byte(key), cacheValue.value})
	}
	sort.Slice(items, func(i, j int) bool {
		if ascending {
			return bytes.Compare(items[i].Key, items[j].Key) < 0
		}
		return bytes.Compare(items[i].Key, items[j].Key) > 0
	})
	return items
}

//----------------------------------------
// etc

func (ci *cache) assertValidKey(key []byte) {
	if key == nil {
		panic("key is nil")
	}
}

// Only entrypoint to mutate ci.cache.
func (ci *cache) setCacheValue(key, value []byte, deleted bool, dirty bool) {
	cacheValue := cValue{
		value:   value,
		deleted: deleted,
		dirty:   dirty,
	}
	ci.cache[string(key)] = cacheValue
}
