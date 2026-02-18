package store

type Node struct {
	value string
	prev  *Node
	next  *Node
}
type LruList struct {
	head *Node
	tail *Node
}

func (st *LruList) AddToHead(node *Node) {

}
func (st *Node) RemoveNode(node *Node) bool {
	if node == nil {
		return false
	}
	return true

}
