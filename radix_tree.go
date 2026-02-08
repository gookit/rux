package rux

import (
	"strings"
	"sync"
)

/*************************************************************
 * Radix Tree 路由树实现
 *************************************************************/

// nodeType 节点类型
type nodeType int8

const (
	nodeTypeStatic   nodeType = iota // 静态节点
	nodeTypeParam                    // 参数节点 :param
	nodeTypeWildcard                 // 通配符节点 *path
	nodeTypeRoot                     // 根节点
)

// radixNode Radix Tree 节点
type radixNode struct {
	// 路径前缀（压缩后的公共前缀）
	prefix string

	// 节点类型
	nType nodeType

	// HTTP 方法 -> 路由处理器映射（仅叶子节点有值）
	handlers map[string]HandlersChain

	// HTTP 方法 -> Route 引用映射（用于快速访问 Route 对象）
	routes map[string]*Route

	// 静态子节点 map[路径前缀]节点
	children map[string]*radixNode

	// 参数子节点（用于 :param）
	paramChild *radixNode

	// 通配符子节点（用于 *path）
	wildcardChild *radixNode

	// 参数名（仅参数节点和通配符节点有值）
	paramName string

	// 父节点引用（用于向上遍历）
	parent *radixNode

	// 是否为叶子节点（有 handler）
	isLeaf bool
}

// radixTree 单个 HTTP 方法的路由树
type radixTree struct {
	root *radixNode
}

// methodTrees 按方法分离的路由树集合
type methodTrees struct {
	trees map[string]*radixTree
	mu    sync.RWMutex
}

// newMethodTrees 创建新的方法路由树集合
func newMethodTrees() *methodTrees {
	return &methodTrees{
		trees: make(map[string]*radixTree),
	}
}

// getTree 获取指定方法的路由树（读锁保护）
func (mt *methodTrees) getTree(method string) (*radixTree, bool) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	tree, ok := mt.trees[method]
	return tree, ok
}

// ensureTree 确保指定方法的路由树存在（写锁保护）
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
		},
	}
	mt.trees[method] = tree
	return tree
}

// newRadixTree 创建新的 Radix Tree
func newRadixTree() *radixTree {
	return &radixTree{
		root: &radixNode{
			prefix:   "/",
			nType:    nodeTypeRoot,
			children: make(map[string]*radixNode),
			handlers: make(map[string]HandlersChain),
		},
	}
}

/*************************************************************
 * 路由插入实现
 *************************************************************/

// AddRoute 添加路由到 Radix Tree（用于测试，不传 Route）
// 支持可选参数展开：/posts[/{id}] 会展开为 /posts 和 /posts/{id}
func (t *radixTree) AddRoute(path string, handlers HandlersChain, methods []string) {
	t.addRouteInternal(path, handlers, methods, nil)
}

// AddRouteWithRoute 添加路由到 Radix Tree，同时存储 Route 引用
func (t *radixTree) AddRouteWithRoute(path string, handlers HandlersChain, methods []string, route *Route) {
	t.addRouteInternal(path, handlers, methods, route)
}

// addRouteInternal 内部路由添加方法
func (t *radixTree) addRouteInternal(path string, handlers HandlersChain, methods []string, route *Route) {
	path = normalizePath(path)

	// 检查并处理可选参数（[ 不在 {} 内）
	if hasOptionalSegment(path) {
		validateOptionalSegments(path)
		expandedPaths := parseOptionalSegments(path)

		// 为每个展开的路径注册路由
		for _, expandedPath := range expandedPaths {
			expandedPath = normalizePath(expandedPath)
			for _, method := range methods {
				t.addHandlerWithRoute(method, expandedPath, handlers, route)
			}
		}
		return
	}

	for _, method := range methods {
		t.addHandlerWithRoute(method, path, handlers, route)
	}
}

// hasOptionalSegment 检查路径是否包含可选段
// 可选段格式：[/{param}] 或 [/static]
// 不包括 regex 模式如 [1-9]
func hasOptionalSegment(path string) bool {
	inBraces := false
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '{':
			inBraces = true
		case '}':
			inBraces = false
		case '[':
			if !inBraces {
				// 检查是否是可选段格式 [/{...}] 或 [/...]
				// 可选段应该以 "/" 开头
				if i+1 < len(path) && path[i+1] == '/' {
					return true
				}
			}
		}
	}
	return false
}

// addHandler 添加单个方法的路由处理器
func (t *radixTree) addHandler(method, path string, handlers HandlersChain) {
	t.addHandlerWithRoute(method, path, handlers, nil)
}

// addHandlerWithRoute 添加单个方法的路由处理器，同时存储 Route 引用
func (t *radixTree) addHandlerWithRoute(method, path string, handlers HandlersChain, route *Route) {
	// 特殊情况：根路径
	if path == "/" {
		t.root.setHandlerWithRoute(method, handlers, route)
		return
	}

	node := t.root
	remaining := path

	for len(remaining) > 0 {
		// 查找最长公共前缀
		commonLen := longestCommonPrefix(node.prefix, remaining)

		// 情况1：当前节点前缀完全匹配，继续向下查找
		if commonLen == len(node.prefix) {
			remaining = remaining[commonLen:]

			// 跳过前导斜杠
			if len(remaining) > 0 && remaining[0] == '/' {
				remaining = remaining[1:]
			}

			// 路径已用完，在当前节点设置 handler
			if len(remaining) == 0 {
				node.setHandlerWithRoute(method, handlers, route)
				return
			}

			// 尝试查找下一个节点
			nextNode := t.findNextNode(node, remaining)
			if nextNode != nil {
				node = nextNode
				continue
			}

			// 没有匹配的子节点，创建新子节点（迭代处理剩余路径）
			node = t.createChildIterativeWithRoute(node, remaining, method, handlers, route)
			return
		}

		// 情况2：公共前缀小于当前节点前缀，需要分裂节点
		if commonLen < len(node.prefix) {
			// 分裂当前节点
			t.splitNode(node, commonLen)

			// 原节点变为子节点，继续在剩余路径上处理
			remaining = remaining[commonLen:]

			if len(remaining) == 0 {
				// 在当前（分裂后的）节点设置 handler
				node.parent.setHandlerWithRoute(method, handlers, route)
				return
			}

			// 创建新子节点存放剩余路径（迭代处理）
			node = t.createChildIterativeWithRoute(node.parent, remaining, method, handlers, route)
			return
		}

		panic("unreachable code in addHandler")
	}
}

// findNextNode 查找下一个匹配的子节点
func (t *radixTree) findNextNode(node *radixNode, path string) *radixNode {
	if len(path) == 0 {
		return nil
	}

	// 跳过前导斜杠
	if path[0] == '/' {
		path = path[1:]
		if len(path) == 0 {
			return nil
		}
	}

	// 处理通配符 *
	if path[0] == '*' {
		return node.wildcardChild
	}

	// 处理参数 :
	if path[0] == ':' {
		return node.paramChild
	}

	// 处理静态子节点 - 提取到下一个 '/' 的路径段
	nextSlash := strings.IndexByte(path, '/')
	if nextSlash == -1 {
		nextSlash = len(path)
	}
	segment := path[:nextSlash]

	return node.children[segment]
}

// createChildIterative 迭代创建子节点（避免递归）
// 返回最后一个创建的节点
func (t *radixTree) createChildIterative(parent *radixNode, path string, method string, handlers HandlersChain) *radixNode {
	return t.createChildIterativeWithRoute(parent, path, method, handlers, nil)
}

// createChildIterativeWithRoute 迭代创建子节点（避免递归），同时存储 Route 引用
// 返回最后一个创建的节点
func (t *radixTree) createChildIterativeWithRoute(parent *radixNode, path string, method string, handlers HandlersChain, route *Route) *radixNode {
	currentParent := parent
	currentPath := path

	for len(currentPath) > 0 {
		// 跳过前导斜杠
		if currentPath[0] == '/' {
			currentPath = currentPath[1:]
			if len(currentPath) == 0 {
				break
			}
		}

		// 处理通配符节点
		if currentPath[0] == '*' {
			paramEnd := len(currentPath)
			paramName := ""
			if paramEnd > 1 {
				paramName = currentPath[1:]
			}

			node := &radixNode{
				prefix:    currentPath,
				nType:     nodeTypeWildcard,
				handlers:  make(map[string]HandlersChain),
				paramName: paramName,
				parent:    currentParent,
			}
			node.setHandlerWithRoute(method, handlers, route)
			currentParent.wildcardChild = node
			return node
		}

		// 处理参数节点
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
				children:  make(map[string]*radixNode),
				paramName: paramName,
				parent:    currentParent,
			}

			remaining := currentPath[paramEnd:]
			node.setHandlerWithRoute(method, handlers, route)
			currentParent.paramChild = node

			// 如果参数后还有路径，继续迭代
			if len(remaining) > 0 {
				currentParent = node
				currentPath = remaining
				continue
			}
			return node
		}

		// 处理静态节点 - 提取第一个路径段
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
			children: make(map[string]*radixNode),
			parent:   currentParent,
		}

		if len(remaining) == 0 {
			node.setHandlerWithRoute(method, handlers, route)
		}

		currentParent.children[segment] = node

		// 如果还有剩余路径，继续迭代
		if len(remaining) > 0 {
			currentParent = node
			currentPath = remaining
			continue
		}
		return node
	}

	return currentParent
}

// splitNode 分裂节点
func (t *radixTree) splitNode(node *radixNode, splitIndex int) {
	if splitIndex <= 0 || splitIndex >= len(node.prefix) {
		return
	}

	splitPrefix := node.prefix[:splitIndex]
	remainingPrefix := node.prefix[splitIndex:]

	// 创建新的父节点（分裂出来的中间节点）
	newNode := &radixNode{
		prefix:   splitPrefix,
		nType:    nodeTypeStatic,
		handlers: make(map[string]HandlersChain),
		routes:   make(map[string]*Route),
		children: make(map[string]*radixNode),
		parent:   node.parent,
	}

	// 将原节点变为新节点的子节点
	node.prefix = remainingPrefix
	node.parent = newNode

	// 将原节点的子节点转移给新节点
	// 注意：原节点的 paramChild, wildcardChild, children 都需要处理
	newNode.children[remainingPrefix] = node

	// 更新父节点的引用
	if newNode.parent != nil {
		parent := newNode.parent

		// 根据原节点的类型更新父节点的引用
		if parent.paramChild == node {
			parent.paramChild = newNode
		} else if parent.wildcardChild == node {
			parent.wildcardChild = newNode
		} else {
			// 在 children map 中替换
			for k, v := range parent.children {
				if v == node {
					delete(parent.children, k)
					parent.children[splitPrefix] = newNode
					break
				}
			}
		}
	} else {
		// 原节点是根节点
		t.root = newNode
	}
}

// setHandler 设置处理器
func (n *radixNode) setHandler(method string, handlers HandlersChain) {
	n.setHandlerWithRoute(method, handlers, nil)
}

// setHandlerWithRoute 设置处理器和 Route 引用
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
 * 路由查找实现
 *************************************************************/

// FindRoute 查找路由
func (t *radixTree) FindRoute(method, path string, strictLastSlash ...bool) (handlers HandlersChain, params Params, found bool) {
	handlers, params, _, found = t.FindRouteWithRoute(method, path, strictLastSlash...)
	return
}

// FindRouteWithRoute 查找路由并返回 Route 引用
// strictLastSlash: 如果为 true，则严格匹配末尾斜杠（/path 和 /path/ 被视为不同）
func (t *radixTree) FindRouteWithRoute(method, path string, strictLastSlash ...bool) (handlers HandlersChain, params Params, route *Route, found bool) {
	strict := false
	if len(strictLastSlash) > 0 {
		strict = strictLastSlash[0]
	}

	params = make(Params)

	// 在非严格模式下，尝试两种形式（有斜杠和没有斜杠）
	if !strict {
		// 先尝试 normalize 后的路径（无斜杠）
		normalizedPath := normalizePath(path)
		handlers, params, route, found = t.findRouteInternal(method, normalizedPath, params, strict)
		if found {
			return
		}

		// 尝试带斜杠的版本
		if normalizedPath != "/" && !strings.HasSuffix(normalizedPath, "/") {
			withSlash := normalizedPath + "/"
			params = make(Params)
			return t.findRouteInternal(method, withSlash, params, strict)
		}
		return nil, params, nil, false
	}

	// 严格模式：保留末尾斜杠
	path = normalizePathStrict(path)
	return t.findRouteInternal(method, path, params, strict)
}

// findRouteInternal 内部路由查找实现
func (t *radixTree) findRouteInternal(method, path string, params Params, strict bool) (handlers HandlersChain, ps Params, route *Route, found bool) {
	// 特殊情况：根路径
	if path == "/" {
		if h, exists := t.root.handlers[method]; exists {
			r := t.root.routes[method]
			return h, params, r, true
		}
		return nil, params, nil, false
	}

	// 从根节点开始查找
	node := t.root
	remaining := path

	// 用于标记是否是第一次循环（根节点）
	isFirst := true

	for node != nil {
		// 对于非根节点，remaining 已经包含了需要匹配的内容
		// 只有根节点需要显式检查 prefix
		if isFirst {
			nodeLen := len(node.prefix)
			if len(remaining) < nodeLen || remaining[:nodeLen] != node.prefix {
				return nil, params, nil, false
			}
			remaining = remaining[nodeLen:]
			isFirst = false
		} else {
			// 对于非根节点，我们已经通过 segment 匹配了 prefix
			// 只需要跳过前导斜杠即可
			if len(remaining) > 0 && remaining[0] == '/' {
				remaining = remaining[1:]
			}
		}

		// 检查是否有通配符子节点（优先级最高）
		if node.wildcardChild != nil {
			if h, exists := node.wildcardChild.handlers[method]; exists {
				params[node.wildcardChild.paramName] = remaining
				r := node.wildcardChild.routes[method]
				return h, params, r, true
			}
		}

		// 路径已用完，检查当前节点是否有 handler
		if len(remaining) == 0 {
			if h, exists := node.handlers[method]; exists {
				r := node.routes[method]
				return h, params, r, true
			}
			return nil, params, nil, false
		}

		// 跳过前导斜杠
		if remaining[0] == '/' {
			remaining = remaining[1:]
			if len(remaining) == 0 {
				// 路径以斜杠结尾
				// 如果 strictLastSlash 为 true，则不匹配（返回失败）
				if strict {
					return nil, params, nil, false
				}
				if h, exists := node.handlers[method]; exists {
					r := node.routes[method]
					return h, params, r, true
				}
				return nil, params, nil, false
			}
		}

		// 尝试匹配参数子节点
		if node.paramChild != nil {
			// 提取参数值（到下一个 '/' 或结束）
			paramEnd := strings.IndexByte(remaining, '/')
			if paramEnd == -1 {
				paramEnd = len(remaining)
			}
			paramValue := remaining[:paramEnd]
			params[node.paramChild.paramName] = paramValue

			// 检查是否是路径的最后部分
			if paramEnd == len(remaining) {
				if h, exists := node.paramChild.handlers[method]; exists {
					r := node.paramChild.routes[method]
					return h, params, r, true
				}
				return nil, params, nil, false
			}

			// 如果 strictLastSlash 为 true 且剩余部分只是斜杠，则不应匹配
			if strict && paramEnd < len(remaining) && remaining[paramEnd] == '/' {
				// 检查斜杠后是否还有其他内容
				if paramEnd+1 == len(remaining) {
					// 路径以斜杠结尾，且 strict 为 true，不匹配
					return nil, params, nil, false
				}
			}

			// 继续在参数子树中查找（跳过已匹配的参数值）
			remaining = remaining[paramEnd:]
			node = node.paramChild
			continue
		}

		// 尝试匹配静态子节点
		nextSlash := strings.IndexByte(remaining, '/')
		if nextSlash == -1 {
			nextSlash = len(remaining)
		}
		segment := remaining[:nextSlash]

		child, ok := node.children[segment]
		if !ok {
			return nil, params, nil, false
		}

		// 找到子节点，继续循环
		// remaining 保持为去掉 segment 后的部分（包括斜杠）
		// 下一轮循环会处理子节点的 prefix
		remaining = remaining[nextSlash:]
		node = child
	}

	return nil, params, nil, false
}

// printTree 打印树结构（用于调试）
func (t *radixTree) printTree(node *radixNode, depth int) string {
	if node == nil {
		node = t.root
	}

	var sb strings.Builder
	indent := strings.Repeat("  ", depth)

	typeStr := "static"
	switch node.nType {
	case nodeTypeParam:
		typeStr = "param"
	case nodeTypeWildcard:
		typeStr = "wildcard"
	case nodeTypeRoot:
		typeStr = "root"
	}

	handlerCount := len(node.handlers)
	sb.WriteString(indent + "[" + typeStr + "] " + node.prefix)
	if handlerCount > 0 {
		sb.WriteString(" (handlers: " + string(rune('0'+handlerCount)) + ")")
	}
	if node.paramName != "" {
		sb.WriteString(" [paramName: " + node.paramName + "]")
	}
	sb.WriteString("\n")

	for _, child := range node.children {
		sb.WriteString(t.printTree(child, depth+1))
	}

	if node.paramChild != nil {
		sb.WriteString(indent + "  [paramChild]\n")
		sb.WriteString(t.printTree(node.paramChild, depth+1))
	}

	if node.wildcardChild != nil {
		sb.WriteString(indent + "  [wildcardChild]\n")
		sb.WriteString(t.printTree(node.wildcardChild, depth+1))
	}

	return sb.String()
}
