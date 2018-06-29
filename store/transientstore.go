package store

import (
	dbm "github.com/tendermint/tendermint/libs/db"
)

var _ KVStore = (*transientStore)(nil)

// transientStore is a wrapper for a MemDB with Commiter implementation
type transientStore struct {
	dbStoreAdapter
}

// Constructs new MemDB adapter
func newTransientStore() *transientStore {
	return &transientStore{dbStoreAdapter{dbm.NewMemDB()}}
}
