package store

import (
	"fmt"
	"strings"
	"testing"
)

func TestNewMapStore(t *testing.T) {

	t.Run("NewMapStore", func(t *testing.T) {
		s := NewMapStore()
		if s.store == nil && s.mux == nil {
			t.FailNow()
		}
	})
}

func TestMapStore_Put(t *testing.T) {
	type args struct {
		key []byte
		val []byte
	}
	tests := []struct {
		args args
	}{
		{args: args{key: []byte("key1"), val: []byte("value1")}},
		{args: args{key: []byte("key2"), val: []byte("value2")}},
		{args: args{key: []byte("key3"), val: []byte("value3")}},
	}

	s := NewMapStore()
	for i, v := range tests {
		t.Run(fmt.Sprintf("MapStore_Put_%d", i), func(t *testing.T) {
			_ = s.Put(v.args.key, v.args.val)
			t.Log(s.store[string(v.args.key)])
			if s.store[string(v.args.key)] != string(v.args.val) {
				t.FailNow()
			}
		})
	}
}

func TestMapStore_Get(t *testing.T) {
	type args struct {
		key []byte
		val []byte
	}
	tests := []struct {
		args args
	}{
		{args: args{key: []byte("key1"), val: []byte("value1")}},
		{args: args{key: []byte("key2"), val: []byte("value2")}},
		{args: args{key: []byte("key3"), val: []byte("value3")}},
	}

	s := NewMapStore()
	s.Put(tests[0].args.key, tests[0].args.val)
	s.Put(tests[1].args.key, tests[1].args.val)
	s.Put(tests[2].args.key, tests[2].args.val)
	for i, v := range tests {
		t.Run(fmt.Sprintf("MapStore_Get_%d", i), func(t *testing.T) {
			v, _ := s.Get(v.args.key)
			if string(v) != string(tests[i].args.val) {
				t.FailNow()
			}
		})
	}
	t.Run(fmt.Sprintf("MapStore_Get_NoItem"), func(t *testing.T) {
		_, err := s.Get([]byte("no_item"))
		if err == nil {
			t.FailNow()
		}
	})
}

func TestNewMapStore_Delete(t *testing.T) {
	type args struct {
		key []byte
		val []byte
	}
	tests := []struct {
		args args
	}{
		{args: args{key: []byte("key1"), val: []byte("value1")}},
		{args: args{key: []byte("key2"), val: []byte("value2")}},
		{args: args{key: []byte("key3"), val: []byte("value3")}},
	}

	s := NewMapStore()
	s.Put(tests[0].args.key, tests[0].args.val)
	s.Put(tests[1].args.key, tests[1].args.val)
	s.Put(tests[2].args.key, tests[2].args.val)

	for i, v := range tests {
		t.Run(fmt.Sprintf("MapStore_Delete_%d", i), func(t *testing.T) {
			_ = s.Delete(v.args.key)
			_, err := s.Get(v.args.key)
			if err == nil {
				t.FailNow()
			}
		})
	}
}

func TestNewMapStore_Items(t *testing.T) {
	type args struct {
		key []byte
		val []byte
	}
	tests := []struct {
		args args
	}{
		{args: args{key: []byte("key1"), val: []byte("value1")}},
		{args: args{key: []byte("key2"), val: []byte("value2")}},
		{args: args{key: []byte("key3"), val: []byte("value3")}},
	}

	s := NewMapStore()
	s.Put(tests[0].args.key, tests[0].args.val)
	s.Put(tests[1].args.key, tests[1].args.val)
	s.Put(tests[2].args.key, tests[2].args.val)

	t.Run(fmt.Sprintf("MapStore_Items_%d", 1), func(t *testing.T) {
		pair, _ := s.Items("key")
		for _, kv := range pair {
			if !strings.Contains(string(kv.Key), "key") {
				t.FailNow()
			}
		}
	})
	t.Run(fmt.Sprintf("MapStore_Items_%d", 2), func(t *testing.T) {
		pair, _ := s.Items("")
		for _, kv := range pair {
			if !strings.Contains(string(kv.Key), "key") {
				t.FailNow()
			}
		}
	})
	t.Run(fmt.Sprintf("MapStore_Items_%d", 3), func(t *testing.T) {
		pair, _ := s.Items()
		for _, kv := range pair {
			if !strings.Contains(string(kv.Key), "key") {
				t.FailNow()
			}
		}
	})
}
