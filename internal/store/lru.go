package store

type Node struct {
	key   string
	value string
	prev  *Node
	next  *Node
}
type LruList struct {
	Head *Node
	Tail *Node
}

func NewLru() *LruList {
	return &LruList{}
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
