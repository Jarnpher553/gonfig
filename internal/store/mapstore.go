package store

import (
	"errors"
	"strings"
	"sync"
	"unsafe"
)

//MapStore store use map
type MapStore struct {
	store map[string]string
	mux   *sync.RWMutex
}

//NewMapStore 新建字典存储
func NewMapStore() *MapStore {
	return &MapStore{store: make(map[string]string), mux: &sync.RWMutex{}}
}

//Put set key/value
func (m *MapStore) Put(key []byte, val []byte) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	k := *(*string)(unsafe.Pointer(&key))
	v := *(*string)(unsafe.Pointer(&val))
	m.store[k] = v
	return nil
}

//Get get key/value
func (m *MapStore) Get(key []byte) ([]byte, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	k := *(*string)(unsafe.Pointer(&key))
	v, ok := m.store[k]
	if !ok {
		return nil, errors.New("no item of key")
	}
	return *(*[]byte)(unsafe.Pointer(&v)), nil
}

//Delete delete key/value
func (m *MapStore) Delete(key []byte) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	k := *(*string)(unsafe.Pointer(&key))
	delete(m.store, k)
	return nil
}

//Items get key/value pairs by key's prefix
func (m *MapStore) Items(prefix ...string) ([]*KeyValuePair, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	out := make([]*KeyValuePair, 0)
	var p string
	if len(prefix) != 0 && prefix[0] != "" {
		p = prefix[0]
	}
	for k, v := range m.store {
		if strings.Index(k, p) == 0 {
			out = append(out, &KeyValuePair{
				Key:   *(*[]byte)(unsafe.Pointer(&k)),
				Value: *(*[]byte)(unsafe.Pointer(&v)),
			})
		}
	}
	return out, nil
}
