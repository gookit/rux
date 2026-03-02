package fastrux

import (
	"strings"
	"sync"
)

/*************************************************************
 * Radix Tree route tree implementation
 *************************************************************/

// nodeType node type
type nodeType int8

const (
	nodeTypeStatic   nodeType = iota // static node
	nodeTypeParam                    // parameter node :param
	nodeTypeWildcard                 // wildcard node *path
	nodeTypeRoot                     // root node
)

// radixNode Radix Tree node
type radixNode struct {
	// path prefix (compressed common prefix)
	prefix string

	// node type
	nType nodeType

	// HTTP method -> route handlers (only leaf nodes have values)
	handlers map[string]HandlersChain

	// HTTP method -> Route reference
	routes map[string]*Route

	// static child nodes map[path prefix]node
	children map[string]*radixNode

	// parameter child node (for :param)
	paramChild *radixNode

	// wildcard child node (for *path)
	wildcardChild *radixNode

	// parameter name (only param and wildcard nodes)
	paramName string

	// parent node reference
	parent *radixNode

	// is leaf node (has handler)
	isLeaf bool
}

// radixTree single HTTP method route tree
type radixTree struct {
	root *radixNode
}

// methodTrees route tree collection separated by HTTP method
type methodTrees struct {
	trees map[string]*radixTree
	mu    sync.RWMutex
}

// newMethodTrees creates a new method route tree collection
func newMethodTrees() *methodTrees {
	return &methodTrees{
		trees: make(map[string]*radixTree),
	}
}

// getTree gets the route tree for a specific method (read lock protected)
func (mt *methodTrees) getTree(method string) (*radixTree, bool) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	tree, ok := mt.trees[method]
	return tree, ok
}

// ensureTree ensures the route tree for a method exists (write lock protected)
func (mt *methodTrees) ensureTree(method string) *radixTree {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if tree, ok := mt.trees[method]; ok {
		return tree
	}

	tree := &radixTree{
		root: &radixNode{
			prefix:   "/",
			nType:    nodeTypeRoot,
			children: make(map[string]*radixNode),
			handlers: make(map[string]HandlersChain),
			routes:   make(map[string]*Route),
		},
	}
	mt.trees[method] = tree
	return tree
}

// newRadixTree creates a new Radix Tree
func newRadixTree() *radixTree {
	return &radixTree{
		root: &radixNode{
			prefix:   "/",
			nType:    nodeTypeRoot,
			children: make(map[string]*radixNode),
			handlers: make(map[string]HandlersChain),
			routes:   make(map[string]*Route),
		},
	}
}

/*************************************************************
 * Route insertion
 *************************************************************/

// AddRoute adds a route to the Radix Tree (without Route reference, for testing)
func (t *radixTree) AddRoute(path string, handlers HandlersChain, methods []string) {
	t.addRouteInternal(path, handlers, methods, nil)
}

// AddRouteWithRoute adds a route to the Radix Tree with Route reference
func (t *radixTree) AddRouteWithRoute(path string, handlers HandlersChain, methods []string, route *Route) {
	t.addRouteInternal(path, handlers, methods, route)
}

// addRouteInternal internal route addition method
func (t *radixTree) addRouteInternal(path string, handlers HandlersChain, methods []string, route *Route) {
	path = normalizePath(path)

	for _, method := range methods {
		t.addHandlerWithRoute(method, path, handlers, route)
	}
}

// addHandlerWithRoute adds route handler for a single method, with Route reference
func (t *radixTree) addHandlerWithRoute(method, path string, handlers HandlersChain, route *Route) {
	// special case: root path
	if path == "/" {
		t.root.setHandlerWithRoute(method, handlers, route)
		return
	}

	node := t.root
	remaining := path

	for len(remaining) > 0 {
		// find longest common prefix
		commonLen := longestCommonPrefix(node.prefix, remaining)

		// case 1: current node prefix fully matches, continue down
		if commonLen == len(node.prefix) {
			remaining = remaining[commonLen:]

			// skip leading slash
			if len(remaining) > 0 && remaining[0] == '/' {
				remaining = remaining[1:]
			}

			// path exhausted, set handler at current node
			if len(remaining) == 0 {
				node.setHandlerWithRoute(method, handlers, route)
				return
			}

			// try to find next matching node
			nextNode := t.findNextNode(node, remaining)
			if nextNode != nil {
				node = nextNode
				continue
			}

			// no matching child found, create new child node (iteratively process remaining path)
			t.createChildIterativeWithRoute(node, remaining, method, handlers, route)
			return
		}

		// case 2: common prefix is shorter than current node prefix, need to split node
		if commonLen < len(node.prefix) {
			t.splitNode(node, commonLen)

			remaining = remaining[commonLen:]

			if len(remaining) == 0 {
				// set handler at current (split) node
				node.parent.setHandlerWithRoute(method, handlers, route)
				return
			}

			// create new child node for remaining path
			t.createChildIterativeWithRoute(node.parent, remaining, method, handlers, route)
			return
		}

		panic("unreachable code in addHandlerWithRoute")
	}
}

// findNextNode finds the next matching child node
func (t *radixTree) findNextNode(node *radixNode, path string) *radixNode {
	if len(path) == 0 {
		return nil
	}

	// skip leading slash
	if path[0] == '/' {
		path = path[1:]
		if len(path) == 0 {
			return nil
		}
	}

	// handle wildcard *
	if path[0] == '*' {
		return node.wildcardChild
	}

	// handle parameter :
	if path[0] == ':' {
		return node.paramChild
	}

	// handle static child nodes - extract path segment up to next '/'
	nextSlash := strings.IndexByte(path, '/')
	if nextSlash == -1 {
		nextSlash = len(path)
	}
	segment := path[:nextSlash]

	return node.children[segment]
}

// createChildIterativeWithRoute iteratively creates child nodes, with Route reference
func (t *radixTree) createChildIterativeWithRoute(parent *radixNode, path string, method string, handlers HandlersChain, route *Route) *radixNode {
	currentParent := parent
	currentPath := path

	for len(currentPath) > 0 {
		// skip leading slash
		if currentPath[0] == '/' {
			currentPath = currentPath[1:]
			if len(currentPath) == 0 {
				break
			}
		}

		// handle wildcard node
		if currentPath[0] == '*' {
			paramName := ""
			if len(currentPath) > 1 {
				paramName = currentPath[1:]
			}

			node := &radixNode{
				prefix:    currentPath,
				nType:     nodeTypeWildcard,
				handlers:  make(map[string]HandlersChain),
				routes:    make(map[string]*Route),
				paramName: paramName,
				parent:    currentParent,
			}
			node.setHandlerWithRoute(method, handlers, route)
			currentParent.wildcardChild = node
			return node
		}

		// handle parameter node
		if currentPath[0] == ':' {
			paramEnd := strings.IndexByte(currentPath, '/')
			if paramEnd == -1 {
				paramEnd = len(currentPath)
			}

			paramName := currentPath[1:paramEnd]
			prefix := currentPath[:paramEnd]

			node := &radixNode{
				prefix:    prefix,
				nType:     nodeTypeParam,
				handlers:  make(map[string]HandlersChain),
				routes:    make(map[string]*Route),
				children:  make(map[string]*radixNode),
				paramName: paramName,
				parent:    currentParent,
			}

			remaining := currentPath[paramEnd:]
			node.setHandlerWithRoute(method, handlers, route)
			currentParent.paramChild = node

			// if there's more path after the param, continue iterating
			if len(remaining) > 0 {
				currentParent = node
				currentPath = remaining
				continue
			}
			return node
		}

		// handle static node - extract first path segment
		nextSlash := strings.IndexByte(currentPath, '/')
		if nextSlash == -1 {
			nextSlash = len(currentPath)
		}
		segment := currentPath[:nextSlash]
		remaining := currentPath[nextSlash:]

		node := &radixNode{
			prefix:   segment,
			nType:    nodeTypeStatic,
			handlers: make(map[string]HandlersChain),
			routes:   make(map[string]*Route),
			children: make(map[string]*radixNode),
			parent:   currentParent,
		}

		if len(remaining) == 0 {
			node.setHandlerWithRoute(method, handlers, route)
		}

		currentParent.children[segment] = node

		// if there's more path remaining, continue iterating
		if len(remaining) > 0 {
			currentParent = node
			currentPath = remaining
			continue
		}
		return node
	}

	return currentParent
}

// splitNode splits a node
func (t *radixTree) splitNode(node *radixNode, splitIndex int) {
	if splitIndex <= 0 || splitIndex >= len(node.prefix) {
		return
	}

	splitPrefix := node.prefix[:splitIndex]
	remainingPrefix := node.prefix[splitIndex:]

	// create new parent node (intermediate node from split)
	newNode := &radixNode{
		prefix:   splitPrefix,
		nType:    nodeTypeStatic,
		handlers: make(map[string]HandlersChain),
		routes:   make(map[string]*Route),
		children: make(map[string]*radixNode),
		parent:   node.parent,
	}

	// make original node a child of new node
	node.prefix = remainingPrefix
	node.parent = newNode

	newNode.children[remainingPrefix] = node

	// update parent's reference to the node
	if newNode.parent != nil {
		parent := newNode.parent

		if parent.paramChild == node {
			parent.paramChild = newNode
		} else if parent.wildcardChild == node {
			parent.wildcardChild = newNode
		} else {
			for k, v := range parent.children {
				if v == node {
					delete(parent.children, k)
					parent.children[splitPrefix] = newNode
					break
				}
			}
		}
	} else {
		// original node was root
		t.root = newNode
	}
}

// setHandlerWithRoute sets handler and Route reference
func (n *radixNode) setHandlerWithRoute(method string, handlers HandlersChain, route *Route) {
	if n.handlers == nil {
		n.handlers = make(map[string]HandlersChain)
	}
	if n.routes == nil {
		n.routes = make(map[string]*Route)
	}
	n.handlers[method] = handlers
	n.routes[method] = route
	n.isLeaf = true
}

/*************************************************************
 * Route lookup
 *************************************************************/

// FindRoute finds a route (without Route reference)
func (t *radixTree) FindRoute(method, path string, strictLastSlash ...bool) (handlers HandlersChain, params Params, found bool) {
	handlers, params, _, found = t.FindRouteWithRoute(method, path, strictLastSlash...)
	return
}

// FindRouteWithRoute finds a route and returns the Route reference.
// Expects the caller (QuickMatch/formatPath) to have already normalized the path.
// strictLastSlash: if true, strictly matches trailing slash (/path and /path/ are different)
func (t *radixTree) FindRouteWithRoute(method, path string, strictLastSlash ...bool) (handlers HandlersChain, params Params, route *Route, found bool) {
	strict := false
	if len(strictLastSlash) > 0 {
		strict = strictLastSlash[0]
	}

	// Ensure path starts with '/'; no heavy normalization since caller handles it.
	if len(path) == 0 {
		path = "/"
	} else if path[0] != '/' {
		path = "/" + path
	}

	if strict {
		// In strict mode we still need to preserve trailing slash as-is.
		return t.findRouteInternal(method, path, nil)
	}

	// Non-strict: path already normalized by formatPath (trailing slash stripped).
	return t.findRouteInternal(method, path, nil)
}

// findRouteInternal internal route lookup implementation
// params can be nil - will be lazy allocated only when needed
func (t *radixTree) findRouteInternal(method, path string, params Params) (handlers HandlersChain, ps Params, route *Route, found bool) {
	// special case: root path
	if path == "/" {
		if h, exists := t.root.handlers[method]; exists {
			r := t.root.routes[method]
			return h, params, r, true
		}
		return nil, params, nil, false
	}

	node := t.root
	remaining := path
	isFirst := true

	for node != nil {
		if isFirst {
			nodeLen := len(node.prefix)
			if len(remaining) < nodeLen || remaining[:nodeLen] != node.prefix {
				return nil, params, nil, false
			}
			remaining = remaining[nodeLen:]
			isFirst = false
		} else {
			// for non-root nodes, skip leading slash
			if len(remaining) > 0 && remaining[0] == '/' {
				remaining = remaining[1:]
			}
		}

		// check for wildcard child (highest priority)
		if node.wildcardChild != nil {
			if h, exists := node.wildcardChild.handlers[method]; exists {
				// Lazy allocate params only when we have a wildcard
				if params == nil {
					params = make(Params)
				}
				params[node.wildcardChild.paramName] = remaining
				r := node.wildcardChild.routes[method]
				return h, params, r, true
			}
		}

		// path exhausted, check if current node has handler
		if len(remaining) == 0 {
			if h, exists := node.handlers[method]; exists {
				r := node.routes[method]
				return h, params, r, true
			}
			return nil, params, nil, false
		}

		// skip leading slash
		if remaining[0] == '/' {
			remaining = remaining[1:]
			if len(remaining) == 0 {
				if h, exists := node.handlers[method]; exists {
					r := node.routes[method]
					return h, params, r, true
				}
				return nil, params, nil, false
			}
		}

		// try parameter child
		if node.paramChild != nil {
			paramEnd := strings.IndexByte(remaining, '/')
			if paramEnd == -1 {
				paramEnd = len(remaining)
			}
			paramValue := remaining[:paramEnd]

			// Lazy allocate params only when we have a param
			if params == nil {
				params = make(Params)
			}
			params[node.paramChild.paramName] = paramValue

			if paramEnd == len(remaining) {
				if h, exists := node.paramChild.handlers[method]; exists {
					r := node.paramChild.routes[method]
					return h, params, r, true
				}
				return nil, params, nil, false
			}

			remaining = remaining[paramEnd:]
			node = node.paramChild
			continue
		}

		// try static children
		nextSlash := strings.IndexByte(remaining, '/')
		if nextSlash == -1 {
			nextSlash = len(remaining)
		}
		segment := remaining[:nextSlash]

		child, ok := node.children[segment]
		if !ok {
			return nil, params, nil, false
		}

		remaining = remaining[nextSlash:]
		node = child
	}

	return nil, params, nil, false
}
