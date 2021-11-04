package store

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
