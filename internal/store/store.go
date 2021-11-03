package store

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type KeyValuePair struct {
	Key   []byte
	Value []byte
}

type Store interface {
	Put([]byte, []byte) error
	Get([]byte) ([]byte, error)
	Delete([]byte) error
	Items(prefix ...string) ([]*KeyValuePair, error)
}

type LeveldbStore struct {
	DB *leveldb.DB
}

func (store *LeveldbStore) Put(k []byte, v []byte) error {
	return store.DB.Put(k, v, nil)
}

func (store *LeveldbStore) Get(k []byte) ([]byte, error) {
	return store.DB.Get(k, nil)
}

func (store *LeveldbStore) Delete(k []byte) error {
	return store.DB.Delete(k, nil)
}

func (store *LeveldbStore) Items(prefix ...string) ([]*KeyValuePair, error) {
	out := make([]*KeyValuePair, 0)
	var iter iterator.Iterator
	if len(prefix) != 0 && prefix[0] != "" {
		iter = store.DB.NewIterator(util.BytesPrefix([]byte(prefix[0])), nil)
	} else {
		iter = store.DB.NewIterator(nil, nil)
	}
	for iter.Next() {
		kv := &KeyValuePair{
			Key:   iter.Key(),
			Value: iter.Value(),
		}
		out = append(out, kv)
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return out, nil
}
