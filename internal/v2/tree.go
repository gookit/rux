package v2

// nodeType classifies a Radix Tree node.
type nodeType uint8

const (
	nodeStatic nodeType = iota
	nodeParam
	nodeWildcard
	nodeRoot
)

// node is a single Radix Tree node. Each node belongs to exactly one
// method tree, so it stores a single handler chain (no method indirection).
//
// Invariants:
//   - For nType == nodeStatic/nodeRoot: prefix is the literal path segment.
//   - For nType == nodeParam: prefix == ":" + paramName, paramName != "".
//   - For nType == nodeWildcard: prefix == "*" + paramName, paramName != "".
//   - len(indices) == len(children).
//   - At most one paramChild and one wildcardChild per node.
type node struct {
	prefix    string
	nType     nodeType
	paramName string

	// Static children, kept sorted by priority desc.
	indices  []byte
	children []*node

	// Dynamic children (one each).
	paramChild    *node
	wildcardChild *node

	// Final handler chain. nil iff non-leaf.
	chain HandlersChain

	// Route metadata for the leaf. nil iff non-leaf.
	route *Route

	// Lookup priority — number of routes registered through this subtree.
	priority uint32
}

// addStaticChild appends a static child node and updates indices.
// Caller is responsible for ensuring no existing static child shares
// the same first byte.
func (n *node) addStaticChild(child *node) {
	n.indices = append(n.indices, child.prefix[0])
	n.children = append(n.children, child)
}

// staticChildIndex returns the index of the static child whose prefix
// starts with b, or -1 if none.
func (n *node) staticChildIndex(b byte) int {
	for i := 0; i < len(n.indices); i++ {
		if n.indices[i] == b {
			return i
		}
	}
	return -1
}

// radixTree wraps the root node and tracks max param count.
type radixTree struct {
	root      *node
	maxParams uint8
}

// newRadixTree creates an empty Radix Tree with a root node.
func newRadixTree() *radixTree {
	return &radixTree{
		root: &node{
			prefix: "/",
			nType:  nodeRoot,
		},
	}
}
