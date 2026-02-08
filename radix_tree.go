package rux

import (
	"strings"
	"sync"
)

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

	// 确保以 '/' 开头
	if path[0] != '/' {
		path = "/" + path
	}

	// 去除尾随 '/'（除非是根路径）
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	// 处理连续的斜杠，压缩为单个斜杠
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
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
	// 检查是否是参数节点
	if len(path) > 0 && path[0] == ':' {
		node := &radixNode{
			nType:    nodeTypeParam,
			children: make(map[string]*radixNode),
			handlers: make(map[string]HandlersChain),
		}
		node.setHandler(method, handlers)
		// 参数名从 : 开始，到 / 或结束
		paramEnd := strings.IndexByte(path, '/')
		if paramEnd == -1 {
			paramEnd = len(path)
		}
		node.paramName = path[1:paramEnd]
		// 参数节点的 prefix 为空
		node.prefix = ""
		parent.paramChild = node
		node.parent = parent
		return
	}

	// 检查是否是通配符节点
	if len(path) > 0 && path[0] == '*' {
		node := &radixNode{
			nType:    nodeTypeWildcard,
			children: make(map[string]*radixNode),
			handlers: make(map[string]HandlersChain),
		}
		node.setHandler(method, handlers)
		node.paramName = path[1:]
		// 通配符节点的 prefix 为空
		node.prefix = ""
		parent.wildcardChild = node
		node.parent = parent
		return
	}

	// 检查路径中是否有 '/'，需要分段处理
	if idx := strings.IndexByte(path, '/'); idx != -1 {
		// 创建静态子节点
		staticPrefix := path[:idx]
		staticNode := &radixNode{
			prefix:   staticPrefix,
			nType:    nodeTypeStatic,
			children: make(map[string]*radixNode),
			handlers: make(map[string]HandlersChain),
		}
		parent.children[staticPrefix] = staticNode
		staticNode.parent = parent

		// 递归处理剩余路径
		t.createChild(staticNode, path[idx+1:], method, handlers)
		return
	}

	// 静态节点
	node := &radixNode{
		prefix:   path,
		nType:    nodeTypeStatic,
		children: make(map[string]*radixNode),
		handlers: make(map[string]HandlersChain),
	}
	node.setHandler(method, handlers)
	parent.children[path] = node
	node.parent = parent
}
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

		if commonPrefix == len(node.prefix) {
			remaining := path[commonPrefix:]

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
			remaining := path[commonPrefix:]

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
			// 对于根节点，直接创建子节点
			if node.parent == nil {
				t.createChild(node, path, method, handlers)
				return
			}
			t.createChild(node.parent, path, method, handlers)
			return
		}

		panic("unreachable")
	}
}

// FindRoute 查找路由
func (t *radixTree) FindRoute(method, path string) (handlers HandlersChain, params Params, found bool) {
	node := t.root
	params = make(Params, 8)
	path = normalizePath(path)

	for {
		// 先检查是否已经完全匹配路径（包括根路径情况）
		if h, exists := node.handlers[method]; exists && (len(path) == 0 || node.prefix == path) {
			return h, params, true
		}

		// 处理通配符（最高优先级）
		// 通配符会匹配所有剩余路径
		if node.wildcardChild != nil {
			if h, exists := node.wildcardChild.handlers[method]; exists {
				params[node.wildcardChild.paramName] = path
				return h, params, true
			}
		}

		// 去掉匹配的前缀（对于静态节点）
		if node.prefix != "" {
			if !strings.HasPrefix(path, node.prefix) {
				return nil, params, false
			}
			path = path[len(node.prefix):]
		}

		// 路径已用完
		if len(path) == 0 {
			if h, exists := node.handlers[method]; exists {
				return h, params, true
			}
			return nil, params, false
		}

		// 处理参数节点
		// 参数节点不需要前缀匹配，直接提取参数值
		if node.paramChild != nil {
			// 参数值到下一个 '/' 或结束
			paramEnd := strings.IndexByte(path, '/')
			if paramEnd == -1 {
				paramEnd = len(path)
			}
			paramValue := path[:paramEnd]

			// 检查参数子节点是否有 handler
			if h, exists := node.paramChild.handlers[method]; exists && paramEnd == len(path) {
				params[node.paramChild.paramName] = paramValue
				return h, params, true
			}

			// 继续在参数子树中查找
			params[node.paramChild.paramName] = paramValue
			path = path[paramEnd:]
			node = node.paramChild
			continue
		}

		// 处理静态子节点
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
