package store

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
)

//LeveldbStore store use leveldb
type LeveldbStore struct {
	DB *leveldb.DB
}

const (
	StorageMem = iota + 1
	StorageFile
)

type StorageType int

func NewLeveldbStore(t StorageType) (*LeveldbStore, error) {
	var (
		db  *leveldb.DB
		err error
	)
	if t == StorageMem {
		db, err = leveldb.Open(storage.NewMemStorage(), nil)
	} else {
		db, err = leveldb.OpenFile("./db", nil)
	}
	if err != nil {
		return nil, err
	}
	return &LeveldbStore{DB: db}, nil
}

//Put set key/value
func (store *LeveldbStore) Put(k []byte, v []byte) error {
	return store.DB.Put(k, v, &opt.WriteOptions{
		Sync: true,
	})
}

//Get get key/value
func (store *LeveldbStore) Get(k []byte) ([]byte, error) {
	return store.DB.Get(k, nil)
}

//Delete delete key/value
func (store *LeveldbStore) Delete(k []byte) error {
	return store.DB.Delete(k, &opt.WriteOptions{
		Sync: true,
	})
}

//Items get key/value pairs by key's prefix
func (store *LeveldbStore) Items(prefix ...string) ([]*KeyValuePair, error) {
	out := make([]*KeyValuePair, 0)
	var iter iterator.Iterator
	if len(prefix) != 0 && prefix[0] != "" {
		iter = store.DB.NewIterator(util.BytesPrefix([]byte(prefix[0])), nil)
	} else {
		iter = store.DB.NewIterator(nil, nil)
	}
	for iter.Next() {
		key := make([]byte, len(iter.Key()))
		value := make([]byte, len(iter.Value()))

		copy(key, iter.Key())
		copy(value, iter.Value())

		kv := &KeyValuePair{
			Key:   key,
			Value: value,
		}
		out = append(out, kv)
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return out, nil
}
