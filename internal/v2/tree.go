package v2

import (
	"fmt"
	"strings"
)

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

// insert adds a normalized path -> route mapping to the tree.
// path must already be normalized (leading '/', no trailing '/', no '//').
// Panics on duplicate path.
//
// Invariants this function depends on:
//   - param/wildcard child nodes have prefix == "" (their semantic prefix
//     ":id" / "*all" is stored in paramName for display only). The empty
//     prefix lets the lookup walk pass through them without trying to
//     prefix-match against the path text.
//   - Static children's `prefix` is the literal byte string they cover.
//     When we create a new static child, we DON'T manually advance
//     `remaining` — we let the next loop iteration's prefix-match consume
//     the matching bytes uniformly.
func (t *radixTree) insert(path string, route *Route) {
	n := t.root
	remaining := path

	for {
		// STEP A: Match n.prefix against remaining.
		cp := longestCommonPrefix(n.prefix, remaining)
		if cp < len(n.prefix) {
			t.splitNode(n, cp)
			// After split, n.prefix == remaining[:cp].
		}

		// STEP B: Consume matched prefix.
		remaining = remaining[cp:]

		// STEP C: Path exhausted at this node.
		if len(remaining) == 0 {
			if n.route != nil {
				panic("rux: duplicate route registration: " + path)
			}
			n.route = route
			n.chain = route.chain
			return
		}

		// STEP D: Dispatch on first byte of remaining.
		c := remaining[0]

		switch c {
		case ':':
			// Param: consume the ":name" segment, descend into paramChild.
			end := strings.IndexByte(remaining, '/')
			if end == -1 {
				end = len(remaining)
			}
			name := remaining[1:end]
			if name == "" {
				panic("rux: empty param name in path " + path)
			}
			if n.paramChild == nil {
				n.paramChild = &node{
					// Empty prefix — see invariants above.
					prefix:    "",
					nType:     nodeParam,
					paramName: name,
				}
			} else if n.paramChild.paramName != name {
				panic(fmt.Sprintf("rux: conflicting param names %q vs %q at %s",
					n.paramChild.paramName, name, path))
			}
			t.bumpMaxParams(t.countParams(path))
			n = n.paramChild
			// Manually advance past the param segment — the empty prefix
			// means the next iteration's prefix-match consumes nothing.
			remaining = remaining[end:]
			continue

		case '*':
			// Wildcard: matches the rest. Always terminal.
			name := remaining[1:]
			if name == "" {
				panic("rux: empty wildcard name in path " + path)
			}
			if n.wildcardChild != nil {
				panic("rux: conflicting wildcard at " + path)
			}
			n.wildcardChild = &node{
				prefix:    "",
				nType:     nodeWildcard,
				paramName: name,
				route:     route,
				chain:     route.chain,
			}
			t.bumpMaxParams(t.countParams(path))
			return

		default:
			// Static child: find by first byte or create.
			idx := n.staticChildIndex(c)
			if idx >= 0 {
				n = n.children[idx]
				// Don't advance remaining — next iteration's prefix-match
				// will consume exactly child.prefix's worth.
				continue
			}
			// Create a new static child. cut = how many bytes to consume.
			cut := indexOfDynamicMarker(remaining)
			if cut < 0 {
				cut = len(remaining)
			}
			child := &node{
				prefix: remaining[:cut],
				nType:  nodeStatic,
			}
			n.addStaticChild(child)
			n = child
			// Don't advance remaining — next iteration matches child.prefix
			// (== remaining[:cut]) uniformly via STEP A.
			continue
		}
	}
}

// splitNode splits node n at byte index splitIdx into a parent
// (n.prefix[:splitIdx]) with one child holding the remainder
// (n.prefix[splitIdx:]) along with all of n's previous children,
// paramChild, wildcardChild, route, and chain.
func (t *radixTree) splitNode(n *node, splitIdx int) {
	if splitIdx <= 0 || splitIdx >= len(n.prefix) {
		return
	}

	// Build the displaced child holding everything that was on n.
	child := &node{
		prefix:        n.prefix[splitIdx:],
		nType:         n.nType,
		paramName:     n.paramName,
		indices:       n.indices,
		children:      n.children,
		paramChild:    n.paramChild,
		wildcardChild: n.wildcardChild,
		chain:         n.chain,
		route:         n.route,
		priority:      n.priority,
	}

	// Reset n to be the new parent.
	n.prefix = n.prefix[:splitIdx]
	n.nType = nodeStatic
	n.paramName = ""
	n.indices = nil
	n.children = nil
	n.paramChild = nil
	n.wildcardChild = nil
	n.chain = nil
	n.route = nil

	n.addStaticChild(child)
}

// indexOfDynamicMarker returns the index of the first ':' or '*' in s, or -1.
func indexOfDynamicMarker(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' || s[i] == '*' {
			return i
		}
	}
	return -1
}

// countParams returns the number of ':' and '*' segments in path.
func (t *radixTree) countParams(path string) uint8 {
	var n uint8
	for i := 0; i < len(path); i++ {
		if path[i] == ':' || path[i] == '*' {
			n++
		}
	}
	return n
}

// bumpMaxParams updates the tree's max param count.
func (t *radixTree) bumpMaxParams(n uint8) {
	if n > t.maxParams {
		t.maxParams = n
	}
	if t.maxParams > MaxParams {
		panic(fmt.Sprintf("rux: route exceeds MaxParams=%d", MaxParams))
	}
}

// lookup searches for the route matching path. On success it appends matched
// path params to ps and returns the route.
//
// Priority: static > param > wildcard. (P-2)
//
// path must already be normalized.
func (t *radixTree) lookup(path string, ps *Params) (*Route, bool) {
	return walkNode(t.root, path, ps)
}

// walkNode is the recursive lookup. It returns (route, true) on hit;
// (nil, false) on miss. Backtracking is implicit via the call stack:
// when a static branch fails, the caller tries param/wildcard fallbacks
// at the same level.
//
// Note: param/wildcard child nodes have prefix == "" (see insert invariants).
// This means strings.HasPrefix(path, "") is trivially true, and we descend
// into their children using the same uniform walkNode call — no special
// "after-param" helper needed.
func walkNode(n *node, path string, ps *Params) (*Route, bool) {
	// Match the node's prefix against the start of path.
	if !strings.HasPrefix(path, n.prefix) {
		return nil, false
	}
	rest := path[len(n.prefix):]

	// Path consumed exactly at this node.
	if len(rest) == 0 {
		if n.route != nil {
			return n.route, true
		}
		return nil, false
	}

	// Snapshot params count for backtracking.
	snap := ps.n

	// 1. Try static children first (priority order: P-2).
	if i := n.staticChildIndex(rest[0]); i >= 0 {
		if r, ok := walkNode(n.children[i], rest, ps); ok {
			return r, true
		}
		ps.n = snap
	}

	// 2. Try param child. Param matches up to the next '/' (or end).
	if n.paramChild != nil {
		end := strings.IndexByte(rest, '/')
		if end == -1 {
			end = len(rest)
		}
		ps.append(n.paramChild.paramName, rest[:end])
		if end == len(rest) {
			// Param consumed the rest of the path.
			if n.paramChild.route != nil {
				return n.paramChild.route, true
			}
		} else {
			// More path remains — descend into paramChild. Empty prefix
			// means walkNode's prefix-match passes trivially, then it
			// dispatches on rest[end]'s first byte (typically '/').
			if r, ok := walkNode(n.paramChild, rest[end:], ps); ok {
				return r, true
			}
		}
		ps.n = snap
	}

	// 3. Try wildcard child — last resort, matches everything remaining.
	if n.wildcardChild != nil {
		ps.append(n.wildcardChild.paramName, rest)
		if n.wildcardChild.route != nil {
			return n.wildcardChild.route, true
		}
		ps.n = snap
	}

	return nil, false
}
