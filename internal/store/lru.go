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
