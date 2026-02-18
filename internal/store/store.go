package store

type Store struct {
	data     map[string]*Node
	capacity int
}

func NewStore(capacity int) *Store {
	return &Store{data: make(map[string]*Node),
		capacity: capacity}

}
func (str *Store) Set(key, value string) bool {
	if str.capacity > 0 {

		str.data[key] = &Node{
			value: value,
		}
		str.capacity--

		return true
	}
	return false
}
func (str *Store) Get(key string) (string, bool) {
	node, ok := str.data[key]
	if !ok {
		return "", !ok
	}
	return node.value, ok
}
func (str *Store) Delete(key string) bool {
	if _, ok := str.data[key]; !ok {
		return false
	}
	delete(str.data, key)
	return true
}
