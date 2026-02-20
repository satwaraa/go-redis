package store

import (
	"errors"
	"sync"
)

var (
	ErrKeyNotFound  = errors.New("key not found")
	ErrStoreFull    = errors.New("store is full")
	ErrInvalidKey   = errors.New("invalid key: cannot be empty")
	ErrInvalidValue = errors.New("invalid value")
)

type Store struct {
	mu       sync.RWMutex
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
func (str *Store) Set(key string, value string) error {
	if key == "" {
		return ErrInvalidKey
	}
	str.mu.Lock()
	defer str.mu.Unlock()
	temp := &Node{
		value: value,
		key:   key,
	}
	if node, ok := str.data[key]; ok {
		node.value = value
		str.lru.MoveToHead(node)
		return nil
	}

	if len(str.data) < str.capacity {

		if str.lru.Head == nil && str.lru.Tail == nil {
			str.data[key] = temp
			str.lru.Head = temp
			str.lru.Tail = temp

			return nil
		}

		str.data[key] = temp
		temp.next = str.lru.Head
		str.lru.Head.prev = temp
		str.lru.Head = temp

		return nil
	} else {

		if str.lru.Tail != nil {
			keyToRemove := str.lru.Tail.key
			str.lru.RemoveLeastUsed()
			delete(str.data, keyToRemove)
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

		return nil
	}
}
func (str *Store) Get(key string) (string, error) {
	if key == "" {
		return "", ErrInvalidKey
	}
	str.mu.Lock()
	defer str.mu.Unlock()
	node, ok := str.data[key]
	if !ok {

		return "", ErrKeyNotFound
	}

	// If already at head, no need to move
	if node == str.lru.Head {

		return node.value, nil
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

	return node.value, nil
}

// deleteInternal removes a key without locking (for internal use only)
func (str *Store) deleteInternal(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	if _, ok := str.data[key]; !ok {
		return ErrKeyNotFound
	}
	// Case 1: Deleting middle Node
	if str.data[key] == str.lru.Head && str.data[key] == str.lru.Tail {
		str.lru.Head = nil
		str.lru.Tail = nil

		// Case 2: Deleting Head
	} else if str.lru.Head == str.data[key] {
		str.lru.Head = str.lru.Head.next
		if str.lru.Head != nil {
			str.lru.Head.prev = nil
		}

		// Case 3: Deleting Tail
	} else if str.lru.Tail == str.data[key] {
		str.lru.Tail = str.lru.Tail.prev
		if str.lru.Tail != nil {
			str.lru.Tail.next = nil
		}
		// Case 4: Node is Both Head and Tail
	} else {
		str.data[key].prev.next = str.data[key].next
		str.data[key].next.prev = str.data[key].prev
	}

	delete(str.data, key)
	return nil
}

func (str *Store) Delete(key string) error {
	str.mu.Lock()
	defer str.mu.Unlock()
	return str.deleteInternal(key)
}
