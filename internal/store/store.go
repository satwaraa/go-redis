package store

type Store struct {
	data     map[string]*Node
	capacity int
}

func NewStore(capacity int) *Store {
	return &Store{data: make(map[string]*Node),
		capacity: capacity}

}
func (str *Store) Set(lru *LruList, key string, value string) bool {
	temp := &Node{
		value: value,
		key:   key,
	}

	if len(str.data) < str.capacity {

		if lru.Head == nil && lru.Tail == nil {
			str.data[key] = temp
			lru.Head = temp
			lru.Tail = temp
			return true
		}

		str.data[key] = temp
		temp.next = lru.Head
		lru.Head.prev = temp
		lru.Head = temp
		return true
	} else {

		if lru.Tail != nil {
			keyToRemove := lru.Tail.key
			lru.RemoveLeastUsed()
			str.Delete(keyToRemove)
		}

		str.data[key] = temp
		if lru.Head != nil {
			temp.next = lru.Head
			lru.Head.prev = temp
		}
		lru.Head = temp
		if lru.Tail == nil {
			lru.Tail = temp
		}
		return true
	}
}
func (str *Store) Get(lru *LruList, key string) (string, bool) {
	node, ok := str.data[key]
	if !ok {
		return "", ok
	}

	// If already at head, no need to move
	if node == lru.Head {
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
	if node == lru.Tail {
		lru.Tail = node.prev
	}

	// Move to head
	node.prev = nil
	node.next = lru.Head
	if lru.Head != nil {
		lru.Head.prev = node
	}
	lru.Head = node

	// If list was empty, set tail
	if lru.Tail == nil {
		lru.Tail = node
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
