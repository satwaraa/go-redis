package store

import (
	"errors"
	"time"
)

// checks if node is expired
func (n *Node) isExpired() bool {
	if n.expireAt == nil {
		return false
	}
	return time.Now().After(*n.expireAt)

}

// hasExpiration returns true if node has TTL set
func (n *Node) hasExpiration() bool {
	return n.expireAt != nil
}

// remainingTTL returns time until expiration
func (n *Node) remainingTTL() time.Duration {
	if n.expireAt == nil {
		return -1 // No expiration
	}
	remaining := time.Until(*n.expireAt)
	if remaining < 0 {
		return 0 // Expired
	}
	return remaining
}

func (st *Store) SetWithTTL(key, value string, ttl time.Duration) error {
	if key == "" {
		return ErrInvalidKey
	}
	if ttl == 0 {
		return errors.New("TTL must be greater than 0")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	expiresAt := time.Now().Add(ttl)
	// check if node exists
	if node, exists := st.data[key]; exists {
		node.value = value
		node.expireAt = &expiresAt
		st.lru.MoveToHead(node)
		return nil

	}
	if len(st.data) >= st.capacity {
		keyToRemove := st.lru.Tail.key
		st.lru.RemoveLeastUsed()
		delete(st.data, keyToRemove)
	}
	node := &Node{
		key:      key,
		value:    value,
		expireAt: &expiresAt,
	}
	st.data[key] = node
	st.lru.MoveToHead(node)
	return nil

}

// Update key expiry
func (st *Store) SetExpiry(key string, ttl time.Duration) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	node, exists := st.data[key]
	if !exists {
		return ErrKeyNotFound
	}
	if ttl <= 0 {
		node.expireAt = nil
	} else {
		expiresAt := time.Now().Add(ttl)
		node.expireAt = &expiresAt
	}
	return nil
}

// StartTTLCleaner starts background goroutine to clean expired keys
func (st *Store) StartTTLCleaner(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			st.cleanExpiredKeys()
		}
	}()
}

// clean Expired Keys
func (st *Store) cleanExpiredKeys() {
	st.mu.Lock()
	defer st.mu.Unlock()
	node := st.lru.Head
	for node != nil {
		next := node.next // save next before potential removal
		if node.isExpired() {
			st.lru.RemoveNode(node)
			delete(st.data, node.key)
		}
		node = next
	}
}

func (st *Store) GetTTL(key string) (time.Duration, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	node, exists := st.data[key]
	if !exists {
		return 0, ErrKeyNotFound
	}
	return node.remainingTTL(), nil
}
