package rux

package rux

import (
	"strings"
	"sync"
)

// HandlersChain handler chain type
type HandlersChain []HandlerFunc

// nodeType 节点类型
type nodeType int8

const (
	nodeTypeStatic   nodeType = iota // 静态节点
	nodeTypeParam                   // 参数节点 :param
	nodeTypeWildcard               // 通配符节点 *path
	nodeTypeRoot                   // 根节点
)

// HandlerFunc handler function
type HandlerFunc func(c *Context)

// nodeType 节点类型
type nodeType int8

const (
	nodeTypeStatic   nodeType = iota // 静态节点
	nodeTypeParam                    // 参数节点 :param
	nodeTypeWildcard                 // 通配符节点 *path
	nodeTypeRoot                     // 根节点
)

// Radix Tree 节点
type radixNode struct {
	prefix        string
	nType         nodeType
	handlers      map[string]HandlersChain
	children      map[string]*radixNode
	paramChild    *radixNode
	wildcardChild *radixNode
	paramName     string
	priority      uint32
	isLeaf        bool
	parent        *radixNode
}

// methodTrees 按方法分离的路由树
type methodTrees struct {
	trees map[string]*radixTree
	mu    sync.RWMutex
}

// radixTree 单个 HTTP 方法的路由树
type radixTree struct {
	root *radixNode
}

// normalizePath 标准化路径
func normalizePath(path string) string {
	if len(path) == 0 {
		return "/"
	}

	path = strings.TrimSpace(path)

	if path[0] != '/' {
		path = "/" + path
	}

	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	return path
}

// longestCommonPrefix 查找最长公共前缀
func longestCommonPrefix(a, b string) int {
	maxLen := len(a)
	if len(b) < maxLen {
		maxLen = len(b)
	}

	i := 0
	for i < maxLen && a[i] == b[i] {
		i++
	}

	return i
}

// setHandler 设置 handler
func (n *radixNode) setHandler(method string, handlers HandlersChain) {
	if n.handlers == nil {
		n.handlers = make(map[string]HandlersChain)
	}
	n.handlers[method] = handlers
	n.isLeaf = true
}

// findNextNode 查找下一个节点
func (t *radixTree) findNextNode(node *radixNode, path string) *radixNode {
	if len(path) == 0 {
		return nil
	}

	if path[0] == '*' {
		return node.wildcardChild
	}

	if path[0] == ':' {
		return node.paramChild
	}

	nextSlash := strings.IndexByte(path, '/')
	if nextSlash == -1 {
		nextSlash = len(path)
	}
	segment := path[:nextSlash]

	return node.children[segment]
}

// createChild 创建子节点
func (t *radixTree) createChild(parent *radixNode, path string, method string, handlers HandlersChain) {
	node := &radixNode{
		prefix:   path,
		nType:    nodeTypeStatic,
		children: make(map[string]*radixNode),
		handlers: make(map[string]HandlersChain),
	}

	node.setHandler(method, handlers)

	if len(path) > 0 && path[0] == ':' {
		node.nType = nodeTypeParam
		paramEnd := strings.IndexByte(path, '/')
		if paramEnd == -1 {
			paramEnd = len(path)
		}
		node.paramName = path[1:paramEnd]
		node.prefix = path[:paramEnd]
		parent.paramChild = node
		return
	}

	if len(path) > 0 && path[0] == '*' {
		node.nType = nodeTypeWildcard
		node.paramName = path[1:]
		parent.wildcardChild = node
		return
	}

	parent.children[path] = node
	node.parent = parent
}

// splitNode 分裂节点
func (t *radixTree) splitNode(node *radixNode, splitIndex int) {
	splitPrefix := node.prefix[:splitIndex]
	remainingPrefix := node.prefix[splitIndex:]

	newNode := &radixNode{
		prefix:   splitPrefix,
		nType:    nodeTypeStatic,
		children: make(map[string]*radixNode),
		handlers: make(map[string]HandlersChain),
	}

	node.prefix = remainingPrefix
	newNode.children[remainingPrefix] = node

	if node.parent != nil {
		if node.parent.paramChild == node {
			node.parent.paramChild = newNode
		} else if node.parent.wildcardChild == node {
			node.parent.wildcardChild = newNode
		} else {
			newNode.parent = node.parent
			for k, v := range node.parent.children {
				if v == node {
					delete(node.parent.children, k)
					node.parent.children[splitPrefix] = newNode
					break
				}
			}
		}
	} else {
		t.root = newNode
	}

	node.parent = newNode
}

// AddRoute 添加路由
func (t *radixTree) AddRoute(path string, handlers HandlersChain, methods []string) {
	path = normalizePath(path)

	for _, method := range methods {
		t.addHandler(method, path, handlers)
	}
}

// addHandler 添加单个方法的路由
func (t *radixTree) addHandler(method string, path string, handlers HandlersChain) {
	node := t.root

	for {
		commonPrefix := longestCommonPrefix(node.prefix, path)

		if commonPrefix == node.prefix {
			remaining := path[len(commonPrefix):]

			if len(remaining) == 0 {
				node.setHandler(method, handlers)
				return
			}

			nextNode := t.findNextNode(node, remaining)
			if nextNode != nil {
				node = nextNode
				path = remaining
				continue
			}

			t.createChild(node, remaining, method, handlers)
			return
		}

		if commonPrefix > 0 && commonPrefix < len(node.prefix) {
			t.splitNode(node, commonPrefix)

			node = node.parent
			remaining := path[len(commonPrefix):]

			if len(remaining) == 0 {
				node.setHandler(method, handlers)
				return
			}

			nextNode := t.findNextNode(node, remaining)
			if nextNode != nil {
				node = nextNode
				path = remaining
				continue
			} else {
				t.createChild(node, remaining, method, handlers)
				return
			}
		}

		if commonPrefix == 0 {
			t.createChild(node.parent, path, method, handlers)
			return
		}

		panic("unreachable")
	}
}

// FindRoute 查找路由
func (t *radixTree) FindRoute(method, path string) (handlers HandlersChain, params Params, found bool) {
	node := t.root
	params := make(Params, 0, 8)
	path = normalizePath(path)

	for {
		if node.wildcardChild != nil {
			if h, exists := node.wildcardChild.handlers[method]; exists {
				params[node.wildcardChild.paramName] = path
				return h, params, true
			}
		}

		if h, exists := node.handlers[method]; exists && len(node.prefix) == len(path) {
			return h, params, true
		}

		if !strings.HasPrefix(path, node.prefix) {
			return nil, params, false
		}

		path = path[len(node.prefix):]

		if len(path) == 0 {
			if h, exists := node.handlers[method]; exists {
				return h, params, true
			}
			return nil, params, false
		}

		if node.paramChild != nil {
			paramEnd := strings.IndexByte(path, '/')
			if paramEnd == -1 {
				paramEnd = len(path)
			}
			paramValue := path[:paramEnd]

			if h, exists := node.paramChild.handlers[method]; exists && paramEnd == len(path) {
				params[node.paramChild.paramName] = paramValue
				return h, params, true
			}

			params[node.paramChild.paramName] = paramValue
			path = path[paramEnd:]
			node = node.paramChild
			continue
		}

		nextSlash := strings.IndexByte(path, '/')
		if nextSlash == -1 {
			nextSlash = len(path)
		}
		segment := path[:nextSlash]

		child, ok := node.children[segment]
		if !ok {
			return nil, params, false
		}

		path = path[nextSlash:]
		node = child
	}
}
