package store

type Store struct {
	data     map[string]*Node
	capacity int
	lru      *LruList
}

func NewStore(capacity int) *Store {
	return &Store{data: make(map[string]*Node),
		capacity: capacity,
		lru:      NewLru(),
	}

}
func (str *Store) Set(key string, value string) bool {
	temp := &Node{
		value: value,
		key:   key,
	}

	if len(str.data) < str.capacity {

		if str.lru.Head == nil && str.lru.Tail == nil {
			str.data[key] = temp
			str.lru.Head = temp
			str.lru.Tail = temp
			return true
		}

		str.data[key] = temp
		temp.next = str.lru.Head
		str.lru.Head.prev = temp
		str.lru.Head = temp
		return true
	} else {

		if str.lru.Tail != nil {
			keyToRemove := str.lru.Tail.key
			str.lru.RemoveLeastUsed()
			str.Delete(keyToRemove)
		}

		str.data[key] = temp
		if str.lru.Head != nil {
			temp.next = str.lru.Head
			str.lru.Head.prev = temp
		}
		str.lru.Head = temp
		if str.lru.Tail == nil {
			str.lru.Tail = temp
		}
		return true
	}
}
func (str *Store) Get(key string) (string, bool) {
	node, ok := str.data[key]
	if !ok {
		return "", ok
	}

	// If already at head, no need to move
	if node == str.lru.Head {
		return node.value, ok
	}

	// Remove node from current position
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}

	// If removing tail, update tail pointer
	if node == str.lru.Tail {
		str.lru.Tail = node.prev
	}

	// Move to head
	node.prev = nil
	node.next = str.lru.Head
	if str.lru.Head != nil {
		str.lru.Head.prev = node
	}
	str.lru.Head = node

	// If list was empty, set tail
	if str.lru.Tail == nil {
		str.lru.Tail = node
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
