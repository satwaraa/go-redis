package store

import (
	"errors"
	"time"
)

var (
	ErrKeyExpired = errors.New("Key has expired")
)

type Node struct {
	key      string
	value    string
	prev     *Node
	next     *Node
	expireAt *time.Time // nil  = no expiration
}
type LruList struct {
	Head *Node
	Tail *Node
}

func NewLru() *LruList {
	return &LruList{}
}

func (st *LruList) AddToHead(node *Node) {
	node.prev = nil
	node.next = st.Head
	if st.Head != nil {
		st.Head.prev = node
	}
	st.Head = node
	if st.Tail == nil {
		st.Tail = node
	}
}
func (st *LruList) MoveToHead(node *Node) {
	if node == st.Head {
		return
	}

	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}

	if node == st.Tail {
		st.Tail = node.prev
	}

	node.prev = nil
	node.next = st.Head
	if st.Head != nil {
		st.Head.prev = node
	}
	st.Head = node

	if st.Tail == nil {
		st.Tail = node
	}
}

func (st *LruList) RemoveLeastUsed() bool {
	if st.Tail == nil {
		return false
	}
	temp := st.Tail
	st.Tail = temp.prev
	if st.Tail != nil {
		st.Tail.next = nil
	}
	temp = nil
	return true
}

func (st *LruList) RemoveNode(nd *Node) {

	if nd == st.Head {
		st.Head = nd.next
		if st.Head != nil {
			st.Head.prev = nil
		}
		return
	}
	if nd == st.Tail {
		st.Tail = nd.prev
		if st.Tail != nil {
			st.Tail.next = nil
		}
		return
	}
	nd.prev.next = nd.next
	nd.next.prev = nd.prev
}
