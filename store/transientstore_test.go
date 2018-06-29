package store

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var k, v = []byte("hello"), []byte("world")

func setKV(t *testing.T, store KVStore) {
	tstore := store.Transient()

	assert.Nil(t, store.Get(k))
	assert.Nil(t, tstore.Get(k))

	store.Set(k, v)
	tstore.Set(k, v)

	assert.Equal(t, v, store.Get(k))
	assert.Equal(t, v, tstore.Get(k))

}

func commitKV(t *testing.T, store CommitKVStore) {
	store.Commit()

	tstore := store.Transient()
	assert.Equal(t, v, store.Get(k))
	assert.Nil(t, tstore.Get(k))
}

func TestTransientIAVLStore(t *testing.T) {
	db := dbm.NewMemDB()
	tree := iavl.NewVersionedTree(db, cacheSize)
	iavlStore := newIAVLStore(tree, numRecent, storeEvery)
	setKV(t, iavlStore)
	commitKV(t, iavlStore)
}

func TestTransientCacheKVStore(t *testing.T) {
	db := dbm.NewMemDB()
	tree := iavl.NewVersionedTree(db, cacheSize)
	iavlStore := newIAVLStore(tree, numRecent, storeEvery)
	cacheStore := NewCacheKVStore(iavlStore)

	setKV(t, cacheStore)
	cacheStore.Write()
	assert.Equal(t, v, iavlStore.Get(k))
	assert.Equal(t, v, iavlStore.Transient().Get(k))
	commitKV(t, iavlStore)
}
