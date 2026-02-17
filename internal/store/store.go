package store

type Store struct {
	data map[string]string
}

func NewStore() *Store {
	return &Store{data: make(map[string]string)}

}
func (str *Store) Set(key, value string) {
	str.data[key] = value
}
func (str *Store) Get(key string) (string, bool) {
	value, ok := str.data[key]
	return value, ok
}
func (str *Store) Delete(key string) bool {
	if _, ok := str.data[key]; !ok {
		return false
	}
	delete(str.data, key)
	return true
}
