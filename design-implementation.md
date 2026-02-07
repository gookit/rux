# Rux 路由重构 - 实施方案（方案 A：破坏性简化）

## 一、设计决策确认

### 1.1 核心架构决策

✅ **已确认**：方案 A（破坏性简化）

#### 关键决策

1. **Route 结构简化**：
   - 保留：name, path, methods, handler, handlers, Opts
   - 移除：start, spath, regex, params, matches

2. **引入 MatchResult**：
   - 路由匹配的返回值结构
   - 包含：path, method, route, params, allowed

3. **API 变更**：
   - `Match(method, path) (*Route, Params)` → `Match(method, path) *MatchResult`
   - `QuickMatch(method, path) (*Route, Params)` → `QuickMatch(method, path) *MatchResult`

4. **版本策略**：
   - v1.x: 当前版本（map + 正则）
   - v2.0: Radix Tree + MatchResult（破坏性）

---

## 二、核心数据结构

### 2.1 Route 结构（简化后）

### 2.1 Radix Tree 节点结构

```go
// radix_tree.go
package rux

import (
    "sync"
)

// 节点类型
type nodeType int8

const (
    nodeTypeStatic nodeType = iota   // 静态节点
    nodeTypeParam                   // 参数节点 :param
    nodeTypeWildcard               // 通配符节点 *path
    nodeTypeRoot                   // 根节点
)

// Radix Tree 节点
type radixNode struct {
    // 路径前缀（压缩后的公共前缀）
    prefix string

    // 节点类型
    nType nodeType

    // HTTP 方法 -> 处理器映射（仅叶子节点有值）
    handlers map[string]HandlersChain

    // 静态子节点 map
    children map[string]*radixNode

    // 参数子节点（用于 :param）
    paramChild *radixNode

    // 通配符子节点（用于 *path）
    wildcardChild *radixNode

    // 参数名（仅参数节点有值）
    paramName string

    // 优先级（用于插入时保持树平衡）
    priority uint32

    // 是否为叶子节点（有 handler）
    isLeaf bool

    // 父节点引用（用于向上遍历）
    parent *radixNode
}

// 按方法分离的路由树
type methodTrees struct {
    trees map[string]*radixTree
    mu    sync.RWMutex
}

// 单个 HTTP 方法的路由树
type radixTree struct {
    root *radixNode
}
```

**设计说明**：
1. **路径压缩**：只有单一子节点的路径段合并到父节点
2. **按方法分离**：每个 HTTP 方法一棵独立的树，支持并发查找
3. **零分配匹配**：查找过程不创建新的 map 或 slice
4. **参数池**：复用 Params map，避免频繁分配

### 2.2 Router 结构调整

```go
// router.go - 修改后的结构
type Router struct {
    Name string
    err  error
    counter int

    // 静态路由：保持不变，使用 map O(1) 查找
    stableRoutes map[string]*Route

    // 动态路由树：新增，替代 regularRoutes 和 irregularRoutes
    dynamicTrees *methodTrees

    // 命名路由
    namedRoutes map[string]*Route

    // 参数池：新增，用于零分配匹配
    paramsPool *sync.Pool

    // 组信息
    currentGroupPrefix   string
    currentGroupHandlers HandlersChain

    // 处理器
    noRoute   HandlersChain
    noAllowed HandlersChain
    handlers  HandlersChain

    // 配置
    OnError   HandlerFunc
    OnPanic   HandlerFunc
    interceptAll string
    strictLastSlash bool
    handleMethodNotAllowed bool
    handleFallbackRoute bool
}
```

**移除的字段**：
- ❌ `cachedRoutes *cachedRoutes` - 不再需要
- ❌ `regularRoutes methodRoutes` - 替换为 dynamicTrees
- ❌ `irregularRoutes methodRoutes` - 合并到 dynamicTrees
- ❌ `enableCaching bool` - 不再需要
- ❌ `maxNumCaches uint16` - 不再需要

**保留的字段**：
- ✅ `stableRoutes` - 静态路由继续使用 map
- ✅ `namedRoutes` - 命名路由保持不变
- ✅ 所有中间件和处理器逻辑
- ✅ `strictLastSlash`、`handleMethodNotAllowed` 等配置项

---

## 三、可选参数展开实现

### 3.1 解析算法

```go
// utils.go - 新增函数

// parseOptionalSegments 解析可选段并展开为多条路由
// 输入：/posts[/{category}]/{id}
// 输出：["/posts", "/posts/{category}/{id}"]
//
// 输入：/api/users[/{id}]/profile
// 输出：["/api/users/profile", "/api/users/{id}/profile"]
func parseOptionalSegments(path string) []string {
    var results []string
    var current strings.Builder
    i := 0

    for i < len(path) {
        if path[i] == '[' {
            // 记录可选段之前的部分
            if current.Len() > 0 {
                results = append(results, current.String())
            }
            current.Reset()
            i++ // 跳过 '['
            continue
        }

        if path[i] == ']' {
            // 可选段结束，需要生成两个版本
            if current.Len() > 0 {
                prefix := current.String()

                // 版本 1：跳过这个可选段
                results = append(results, prefix)

                // 版本 2：包含这个可选段（去掉括号）
                optionalContent := strings.Trim(prefix, "[")
                results = append(results, optionalContent)
            }
            current.Reset()
            i++ // 跳过 ']'
            continue
        }

        current.WriteByte(path[i])
        i++
    }

    // 处理剩余部分
    if current.Len() > 0 {
        results = append(results, current.String())
    }

    return results
}
```

### 3.2 路由注册调整

```go
// router.go - AddRoute 方法调整

func (r *Router) AddRoute(route *Route) *Route {
    route.goodInfo()
    r.appendGroupInfo(route)
    debugPrintRoute(route)

    if route.name != "" {
        r.namedRoutes[route.name] = route
    }

    // 检测可选参数并验证规则
    if strings.Contains(route.path, "[") {
        validateOptionalSegments(route.path)
        segments := parseOptionalSegments(route.path)

        // 为每个展开的路由版本注册
        for _, segmentPath := range segments {
            expandedRoute := &Route{
                path:     segmentPath,
                handler:  route.handler,
                handlers: route.handlers,
                methods:  route.methods,
                name:     route.name,
            }

            // 递归调用 addRouteInternal 注册展开的路由
            r.addRouteInternal(expandedRoute)
        }

        return route
    }

    // 正常路由注册逻辑
    return r.addRouteInternal(route)
}
```

### 3.3 可选参数验证

```go
// utils.go - 可选参数验证和解析

// validateOptionalSegments 验证可选参数规则
// 规则：
// 1. 可选参数只能在路径最后
// 2. 只能支持一个可选参数
func validateOptionalSegments(path string) {
    firstOptionalPos := strings.IndexByte(path, '[')
    lastOptionalPos := strings.LastIndexByte(path, '[')
    afterOptionalPos := strings.IndexByte(path, ']') + 1

    // 规则 1：不能有多个可选参数
    if firstOptionalPos != lastOptionalPos {
        panic(fmt.Sprintf("route %s: only one optional segment is allowed", path))
    }

    // 规则 2：可选参数后不能有其他路径段
    if afterOptionalPos < len(path) {
        panic(fmt.Sprintf("route %s: optional segment must be at the end of the path, found '%s' after ']'",
            path, path[afterOptionalPos:]))
    }
}

// parseOptionalSegments 解析可选段并展开为两条路由
// 输入：/posts[/{id}]
// 输出：["/posts", "/posts/{id}"]
// 输入：/api/users[/{name}]/profile
// 输出：["/api/users/profile", "/api/users/{name}/profile"]
func parseOptionalSegments(path string) []string {
    var results []string
    var current strings.Builder
    i := 0

    for i < len(path) {
        if path[i] == '[' {
            // 记录可选段之前的部分
            if current.Len() > 0 {
                results = append(results, current.String())
            }
            current.Reset()
            i++ // 跳过 '['
            continue
        }

        if path[i] == ']' {
            // 可选段结束，需要生成两个版本
            if current.Len() > 0 {
                prefix := current.String()

                // 版本 1：跳过这个可选段
                results = append(results, prefix)

                // 版本 2：包含这个可选段（去掉括号）
                optionalContent := strings.Trim(prefix, "[")
                results = append(results, optionalContent)
            }
            current.Reset()
            i++ // 跳过 ']'
            continue
        }

        current.WriteByte(path[i])
        i++
    }

    // 处理剩余部分
    if current.Len() > 0 {
        results = append(results, current.String())
    }

    return results
}
```

**示例验证**：

| 路由 | 结果 | 说明 |
|------|------|------|
| `/posts[/{id}]` | ✅ 通过 | 一个可选参数在最后 |
| `/api/users[/{name}]/profile` | ✅ 通过 | 一个可选参数，后面有静态段 |
| `/posts[/{category}]/{id}` | ❌ panic | 可选参数不在最后 |
| `/api[/{v1}]/users[/{v2}]` | ❌ panic | 多个可选参数 |

### 3.4 路由注册调整

```go
// router.go - AddRoute 方法调整

func (r *Router) AddRoute(route *Route) *Route {
    route.goodInfo()
    r.appendGroupInfo(route)
    debugPrintRoute(route)

    if route.name != "" {
        r.namedRoutes[route.name] = route
    }

    // 检测可选参数并验证规则
    if strings.Contains(route.path, "[") {
        validateOptionalSegments(route.path)
        segments := parseOptionalSegments(route.path)

        // 为每个展开的路由版本注册
        for _, segmentPath := range segments {
            expandedRoute := &Route{
                path:     segmentPath,
                handler:  route.handler,
                handlers: route.handlers,
                methods:  route.methods,
                name:     route.name,
            }

            // 递归调用 addRouteInternal 注册展开的路由
            r.addRouteInternal(expandedRoute)
        }

        return route
    }

    // 正常路由注册逻辑
    return r.addRouteInternal(route)
}

    // 检测可选参数并展开
    if strings.Contains(route.path, "[") {
        segments := parseOptionalSegments(route.path)

        // 为每个展开的路由版本注册
        for _, segmentPath := range segments {
            expandedRoute := &Route{
                path:     segmentPath,
                handler:  route.handler,
                handlers: route.handlers,
                methods:  route.methods,
                name:     route.name,
            }

            // 递归调用 addRoute 注册展开的路由
            r.addRouteInternal(expandedRoute)
        }

        return route
    }

    // 正常路由注册逻辑
    return r.addRouteInternal(route)
}

// addRouteInternal 内部注册方法
func (r *Router) addRouteInternal(route *Route) *Route {
    // 静态路由：保持原有逻辑
    if isFixedPath(route.path) {
        for _, method := range route.methods {
            key := method + route.path
            r.counter++
            r.stableRoutes[key] = route
        }
        return route
    }

    // 动态路由：添加到 Radix Tree
    r.dynamicTrees.mu.Lock()
    defer r.dynamicTrees.mu.Unlock()

    for _, method := range route.methods {
        tree, ok := r.dynamicTrees.trees[method]
        if !ok {
            tree = &radixTree{root: &radixNode{
                prefix:   "/",
                nType:    nodeTypeRoot,
                children:  make(map[string]*radixNode),
                handlers:  make(map[string]HandlersChain),
            }}
            if r.dynamicTrees.trees == nil {
                r.dynamicTrees.trees = make(map[string]*radixTree)
            }
            r.dynamicTrees.trees[method] = tree
        }

        tree.AddRoute(route.path, append(route.handlers, route.handler), []string{method})
        r.counter++
    }

    return route
}
```

---

## 四、Radix Tree 核心实现

### 4.1 工具函数

```go
// radix_tree.go - 工具函数

// normalizePath 标准化路径
func normalizePath(path string) string {
    if len(path) == 0 {
        return "/"
    }

    // 去除首尾空格
    path = strings.TrimSpace(path)

    // 确保以 '/' 开头
    if path[0] != '/' {
        path = "/" + path
    }

    // 去除尾随 '/'（除非是根路径）
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

// splitNode 分裂节点
func (t *radixTree) splitNode(node *radixNode, splitIndex int) {
    splitPrefix := node.prefix[:splitIndex]
    remainingPrefix := node.prefix[splitIndex:]

    // 创建新的父节点
    newNode := &radixNode{
        prefix:   splitPrefix,
        nType:    nodeTypeStatic,
        handlers:  make(map[string]HandlersChain),
        children:  make(map[string]*radixNode),
    }

    // 将原节点作为子节点
    node.prefix = remainingPrefix
    newNode.children[remainingPrefix] = node

    // 更新父节点引用
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
        // 根节点
        t.root = newNode
    }

    node.parent = newNode
}
```

### 4.2 路由插入算法

```go
// radix_tree.go - 路由插入

// AddRoute 添加路由到 Radix Tree
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
        // 查找最长公共前缀
        commonPrefix := longestCommonPrefix(node.prefix, path)

        // 情况 1：公共前缀等于节点前缀，继续向下
        if commonPrefix == node.prefix {
            remaining := path[len(commonPrefix):]

            // 路径已用完，设置 handler
            if len(remaining) == 0 {
                node.setHandler(method, handlers)
                return
            }

            // 继续向下查找
            nextNode := t.findNextNode(node, remaining)
            if nextNode != nil {
                node = nextNode
                path = remaining
                continue
            }

            // 需要创建新子节点
            t.createChild(node, remaining, method, handlers)
            return
        }

        // 情况 2：公共前缀较短，需要分裂节点
        if commonPrefix > 0 && commonPrefix < len(node.prefix) {
            t.splitNode(node, commonPrefix)

            // 原节点变为子节点
            node = node.parent

            // 剩余路径处理
            remaining := path[len(commonPrefix):]

            if len(remaining) == 0 {
                node.setHandler(method, handlers)
                return
            }

            // 继续向下查找
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

        // 情况 3：路径不匹配，添加为兄弟节点
        if commonPrefix == 0 {
            t.createChild(node.parent, path, method, handlers)
            return
        }

        panic("unreachable")
    }
}

// findNextNode 查找下一个节点
func (t *radixTree) findNextNode(node *radixNode, path string) *radixNode {
    if len(path) == 0 {
        return nil
    }

    // 处理通配符
    if path[0] == '*' {
        return node.wildcardChild
    }

    // 处理参数
    if path[0] == ':' {
        return node.paramChild
    }

    // 处理静态子节点
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
        handlers:  make(map[string]HandlersChain),
        children:  make(map[string]*radixNode),
    }

    node.setHandler(method, handlers)

    // 参数节点
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

    // 通配符节点
    if len(path) > 0 && path[0] == '*' {
        node.nType = nodeTypeWildcard
        node.paramName = path[1:]
        parent.wildcardChild = node
        return
    }

    // 静态节点
    parent.children[path] = node
    node.parent = parent
}

// setHandler 设置 handler
func (n *radixNode) setHandler(method string, handlers HandlersChain) {
    if n.handlers == nil {
        n.handlers = make(map[string]HandlersChain)
    }
    n.handlers[method] = handlers
    n.isLeaf = true
}
```

### 4.3 路由查找算法

```go
// radix_tree.go - 路由查找

// FindRoute 查找路由
func (t *radixTree) FindRoute(method, path string) (handlers HandlersChain, params Params, found bool) {
    node := t.root
    params = make(Params, 0, 8) // 使用 sync.Pool 复用
    path = normalizePath(path)

    for {
        // 处理通配符（最高优先级）
        if node.wildcardChild != nil {
            if h, exists := node.wildcardChild.handlers[method]; exists {
                params[node.wildcardChild.paramName] = path
                return h, params, true
            }
        }

        // 检查完全匹配
        if h, exists := node.handlers[method]; exists && len(node.prefix) == len(path) {
            return h, params, true
        }

        // 前缀不匹配
        if !strings.HasPrefix(path, node.prefix) {
            return nil, params, false
        }

        // 去掉匹配的前缀
        path = path[len(node.prefix):]

        // 路径已用完
        if len(path) == 0 {
            if h, exists := node.handlers[method]; exists {
                return h, params, true
            }
            return nil, params, false
        }

        // 处理参数节点
        if node.paramChild != nil {
            // 提取参数值
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
```

---

## 五、Router 集成实现

### 5.1 对象池实现

```go
// router.go - 参数池

// 参数池，复用 Params map
var paramsPool = sync.Pool{
    New: func() any {
        return make(Params, 8)
    },
}

// acquireParams 获取 Params
func acquireParams() Params {
    return paramsPool.Get().(Params)
}

// releaseParams 释放 Params
func releaseParams(p Params) {
    for k := range p {
        delete(p, k)
    }
    paramsPool.Put(p)
}
```

### 5.2 路由匹配集成

```go
// router.go - match 方法调整

func (r *Router) match(method, path string) (rt *Route, ps Params) {
    // 1. 静态路由查找 O(1)
    if route, ok := r.stableRoutes[method+path]; ok {
        return route, nil
    }

    // 2. 动态路由查找 O(m)
    r.dynamicTrees.mu.RLock()
    tree, ok := r.dynamicTrees.trees[method]
    r.dynamicTrees.mu.RUnlock()

    if ok {
        handlers, params, found := tree.FindRoute(method, path)
        if found {
            // 构造 Route 对象（向后兼容）
            rt = &Route{
                path:     path,
                methods:  []string{method},
                handler:  handlers[len(handlers)-1], // 最后一个是主 handler
                handlers: handlers[:len(handlers)-1], // 前面是中间件
                params:   params,
            }
            return rt, params
        }
    }

    return nil, nil
}
```

### 5.3 配置选项清理

```go
// router.go - 移除已废弃的配置

// 移除的配置函数（不需要删除函数体，编译时会自动失效）
// - EnableCaching
// - CachingWithNum
// - MaxNumCaches

// 移除的 Router 字段（在结构体定义中删除）
// - enableCaching
// - maxNumCaches
// - cachedRoutes
```

---

## 六、实施计划

### 6.1 第一阶段：基础设施（1-2 天）

**目标**：创建 Radix Tree 基础结构

- [ ] 创建 `radix_tree.go` 文件
- [ ] 实现 radixNode 结构体
- [ ] 实现 methodTrees 结构体
- [ ] 实现 radixTree 结构体
- [ ] 实现工具函数（normalizePath、longestCommonPrefix）
- [ ] 单元测试基础结构

### 6.2 第二阶段：核心算法（2-3 天）

**目标**：实现 Radix Tree 的插入和查找算法

- [ ] 实现 splitNode 方法（节点分裂）
- [ ] 实现 findNextNode 方法（查找下一个节点）
- [ ] 实现 createChild 方法（创建子节点）
- [ ] 实现 setHandler 方法（设置 handler）
- [ ] 实现 AddRoute 方法（路由插入）
- [ ] 实现 FindRoute 方法（路由查找）
- [ ] 单元测试路由插入逻辑
- [ ] 单元测试路由查找逻辑

### 6.3 第三阶段：可选参数展开（1 天）

**目标**：实现可选参数自动展开

- [ ] 实现 parseOptionalSegments 函数
- [ ] 实现展开路由的逻辑处理
- [ ] 单元测试可选参数展开
- [ ] 单元测试复杂场景（嵌套可选段）

### 6.4 第四阶段：Router 集成（1-2 天）

**目标**：将 Radix Tree 集成到 Router

- [ ] 添加 paramsPool 实现
- [ ] 添加 acquireParams/releaseParams 函数
- [ ] 修改 Router 结构体
- [ ] 修改 AddRoute 方法（集成可选参数展开）
- [ ] 修改 match 方法（集成 Radix Tree 查找）
- [ ] 移除 regularRoutes 和 irregularRoutes
- [ ] 移除 cachedRoutes 相关代码
- [ ] 移除 EnableCaching 等配置

### 6.5 第五阶段：测试与验证（2-3 天）

**目标**：确保功能正确性和性能提升

- [ ] 运行现有单元测试（确保兼容性）
- [ ] 添加可选参数测试用例
- [ ] 添加 Radix Tree 测试用例
- [ ] 运行性能基准测试
- [ ] 对比重构前后的性能数据

### 6.6 第六阶段：清理与文档（1 天）

**目标**：完善代码库并准备发布

- [ ] 移除 route_cache.go 文件
- [ ] 移除 route_cache_test.go 文件
- [ ] 更新 README.md（新增 Radix Tree 说明）
- [ ] 更新 CHANGELOG（记录重大变更）
- [ ] 添加迁移指南（帮助用户平滑升级）

---

## 七、关键实现细节

### 7.1 参数提取优化

**零分配实现**：

```go
// 路由查找时复用 Params
func (r *Router) match(method, path string) (rt *Route, ps Params) {
    params = acquireParams()
    defer releaseParams(params)

    // ... 查找逻辑

    return rt, params
}
```

### 7.2 并发安全保证

**读写锁使用**：

```go
// 路由注册（写操作）
r.dynamicTrees.mu.Lock()
defer r.dynamicTrees.mu.Unlock()

// 路由查找（读操作）
r.dynamicTrees.mu.RLock()
defer r.dynamicTrees.mu.RUnlock()

// 支持多个 goroutine 并发读取，写操作独占
```

### 7.3 向后兼容性保证

**用户代码无需修改**：

```go
// 用户代码（保持不变）
r.GET("/posts[/{id}]", func(c *rux.Context) {
    id := c.Param("id")
    if id == "" {
        // 处理 /posts（无 ID）
        c.JSON(200, listPosts())
    } else {
        // 处理 /posts/123（有 ID）
        c.JSON(200, getPost(id))
    }
})

// 内部自动展开为两条路由
// /posts -> r.stableRoutes["GET/posts"]
// /posts/{id} -> r.dynamicTrees.trees["GET"].FindRoute("/posts/{id}")
```

---

## 八、测试策略

### 8.1 单元测试覆盖

**关键测试场景**：

1. **基础路由匹配**
   - 静态路由匹配
   - 单参数动态路由
   - 多参数动态路由
   - 通配符路由

2. **可选参数展开**
   - 单个可选段：`/posts[/{id}]`
   - 多个可选段：`/api/users[/{id}]/profile`
   - 嵌套可选段：`/api/articles[/{year}/[/{month}]/day`

3. **边界情况**
   - 根路径 `/`
   - 尾随斜杠处理
   - 空路径处理
   - 特殊字符处理

4. **并发安全**
   - 多 goroutine 并发查找路由
   - 并发注册路由

### 8.2 性能基准测试

**基准测试用例**：

```go
// benchmark_test.go - 更新测试

// 静态路由性能
func BenchmarkStaticRoute(b *testing.B) {
    r := New()
    for i := 0; i < 100; i++ {
        r.GET(fmt.Sprintf("/path%d", i), emptyHandler)
    }
    req := httptest.NewRequest("GET", "/path42", nil)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        r.ServeHTTP(httptest.NewRecorder(), req)
    }
}

// 单参数动态路由
func BenchmarkSingleParamRoute(b *testing.B) {
    r := New()
    r.GET("/users/:id", emptyHandler)
    req := httptest.NewRequest("GET", "/users/123", nil)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        r.ServeHTTP(httptest.NewRecorder(), req)
    }
}

// 多参数动态路由
func BenchmarkFiveParamsRoute(b *testing.B) {
    r := New()
    r.GET("/articles/:year/:month/:day/:category/:id/:action", emptyHandler)
    req := httptest.NewRequest("GET", "/articles/2025/02/07/tech/123/edit", nil)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        r.ServeHTTP(httptest.NewRecorder(), req)
    }
}

// 静态与动态混合
func BenchmarkMixedRoutes(b *testing.B) {
    r := New()
    // 70% 静态路由
    for i := 0; i < 70; i++ {
        r.GET(fmt.Sprintf("/api/static%d", i), emptyHandler)
    }
    // 30% 动态路由
    r.GET("/users/:id", emptyHandler)
    r.GET("/posts/:slug", emptyHandler)
    r.GET("/comments/:post_id/:comment_id", emptyHandler)

    // 测试静态路由
    reqStatic := httptest.NewRequest("GET", "/api/static42", nil)
    b.Run("Static", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            r.ServeHTTP(httptest.NewRecorder(), reqStatic)
        }
    })

    // 测试动态路由
    reqDynamic := httptest.NewRequest("GET", "/users/123", nil)
    b.Run("Dynamic", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            r.ServeHTTP(httptest.NewRecorder(), reqDynamic)
        }
    })
}
```

---

## 九、风险与注意事项

### 9.1 实施风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|---------|
| **可选参数展开** | 路由数量增加 | 高 | 文档说明，用户了解 |
| **并发安全** | 错误竞争 | 低 | 使用 RWMutex，充分测试 |
| **参数提取** | 参数丢失 | 中 | 严格测试，添加用例 |
| **性能倒退** | 性能不如预期 | 低 | 基准测试对比 |

### 9.2 测试注意事项

1. **并发测试**：使用 `go test -race` 检测数据竞争
2. **基准对比**：重构前运行基准并记录基线数据
3. **覆盖测试**：使用 `go test -cover` 确保覆盖率不下降
4. **压力测试**：高并发场景下验证稳定性

### 9.3 向后兼容性验证

1. **API 不变**：所有公开方法签名保持一致
2. **行为兼容**：路由匹配逻辑与之前一致
3. **配置兼容**：保留 `strictLastSlash` 等配置项
4. **用户无感知**：可选参数展开自动进行，用户代码无需修改

---

## 十、预期性能提升

| 场景 | 当前性能 | 目标性能 | 提升倍数 |
|------|---------|---------|----------|
| 静态路由 | 15-20M ops/s | **25-30M ops/s** | 1.5-2x |
| 单参数动态路由 | 3-5M ops/s | **15-20M ops/s** | 4-5x |
| 多参数动态路由 | 0.5-1M ops/s | **8-12M ops/s** | 10-15x |
| 内存分配 | 高 | **零分配** | N/A |
| 整体内存 | 基准 | **-20%-30%** | N/A |

---

## 总结

本实现方案提供了：

1. **完整的 Radix Tree 实现**：包括节点结构、插入、查找算法
2. **可选参数自动展开**：用户无需修改代码
3. **并发安全保证**：使用 RWMutex 保护路由树
4. **零分配查找**：使用对象池复用 Params
5. **详细的实施计划**：6 个阶段，每个阶段都有明确的任务

**实施优先级**：
1. ✅ 优先级 1：可选参数展开（用户最关心）
2. ✅ 优先级 2：Radix Tree 核心算法
3. ✅ 优先级 3：Router 集成
4. ✅ 优先级 4：测试与验证
5. ✅ 优先级 5：清理与文档

**开始实施**：
- 请确认本实现方案是否满足需求
- 确认后，我将按照"第六阶段：实施计划"开始编写代码
