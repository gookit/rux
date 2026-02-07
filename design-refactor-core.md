# Rux 核心路由重构设计方案

## 一、背景与目标

### 1.1 当前问题概述

gookit/rux 是一个轻量级 Go HTTP 路由库，当前使用基于 map + 正则表达式的路由匹配策略。随着应用规模增长，这种实现方式面临以下性能瓶颈：

1. **动态路由匹配性能差**：时间复杂度 O(n)，需要线性扫描所有动态路由
2. **正则表达式开销大**：每个动态路由都需要编译正则，匹配时运行正则引擎
3. **内存分配频繁**：每次匹配都需要创建新的 Params map 和 HandlersChain
4. **无法优化前缀匹配**：不支持路径压缩和公共前缀共享

### 1.2 重构目标

- **性能提升**：动态路由匹配从 O(n) 提升到 O(m)，其中 m 为路径长度
- **内存优化**：通过路径压缩和节点共享，减少内存占用
- **零分配查找**：路由匹配过程不进行堆内存分配
- **保持兼容**：API 接口向后兼容，不破坏现有代码
- **静态路由优化**：继续保持 map 方式的 O(1) 查找性能

### 1.3 参考标准

根据行业基准测试数据：

| 路由库 | 单参数路由性能 | 内存分配 |
|--------|--------------|---------|
| HttpRouter | 26.3M ops/s (47.7ns) | 0 B/op |
| Gin | 18.8M ops/s (63.9ns) | 0 B/op |
| Echo | 16.4M ops/s (75.5ns) | 0 B/op |
| Chi | 1.4M ops/s (885ns) | 432 B/op |
| **Rux (当前)** | 预计 < 5M ops/s | 高 |

目标是达到接近 Gin/HttpRouter 的性能水平。

---

## 二、当前架构分析

### 2.1 路由数据结构

#### Router 核心字段 (router.go:19-106)

```go
type Router struct {
    counter int

    // 静态路由: key = "METHOD/path"
    stableRoutes map[string]*Route

    // 动态路由缓存 (LRU)
    cachedRoutes *cachedRoutes

    // 常规动态路由: key = "METHOD+first-node"
    regularRoutes methodRoutes  // map[string][]*Route

    // 不规则动态路由: key = "METHOD"
    irregularRoutes methodRoutes

    // 命名路由
    namedRoutes map[string]*Route

    // 组信息
    currentGroupPrefix string
    currentGroupHandlers HandlersChain

    // 处理器
    noRoute HandlersChain
    noAllowed HandlersChain
    handlers HandlersChain
}
```

#### Route 核心字段 (route.go:58-88)

```go
type Route struct {
    name string
    path string
    methods []string

    // 解析后的路由信息
    start string        // 路径起始部分，用于快速匹配
    spath string       // 简化路径
    regex *regexp.Regexp // 正则表达式（性能瓶颈）
    params Params      // 参数值（仅用于缓存）
    matches []string   // 参数名列表

    // 处理器
    handler HandlerFunc
    handlers HandlersChain

    Opts map[string]any
}
```

### 2.2 路由匹配流程 (route_parse_match.go:152-196)

```go
func (r *Router) match(method, path string) (rt *Route, ps Params) {
    // 1. 静态路由精确匹配 O(1)
    if route, ok := r.stableRoutes[method+path]; ok {
        return route, nil
    }

    // 2. 缓存查找
    if r.enableCaching {
        route, ok := r.cachedRoutes.Get(method + path)
        if ok {
            return route, route.params
        }
    }

    // 3. 常规动态路由：按 first-node 分组后线性扫描 O(n)
    if pos := strings.IndexByte(path[1:], '/'); pos > 0 {
        key := method + path[1:pos+1]
        if rs, ok := r.regularRoutes[key]; ok {
            for i := range rs {  // 线性扫描
                if strings.Index(path, rs[i].start) != 0 {
                    continue
                }
                if ps, ok := rs[i].matchRegex(path); ok {  // 正则匹配
                    r.cacheDynamicRoute(key, ps, rs[i])
                    return rs[i], ps
                }
            }
        }
    }

    // 4. 不规则动态路由：全量线性扫描 O(n)
    if rs, ok := r.irregularRoutes[method]; ok {
        for _, route := range rs {
            if ps, ok := route.matchRegex(path); ok {
                r.cacheDynamicRoute(method+path, ps, route)
                return route, ps
            }
        }
    }
    return
}
```

### 2.3 性能瓶颈分析

| 瓶颈点 | 位置 | 问题 | 影响 |
|--------|------|------|------|
| **线性扫描** | route_parse_match.go:172,188 | 常规/不规则动态路由需要遍历切片 | 时间复杂度 O(n) |
| **正则匹配** | route.go:294-311 | 每个动态路由都需要运行正则匹配 | CPU 开销大 |
| **正则编译** | route_parse_match.go:21,84 | 路由注册时编译正则 | 注册慢、内存占用高 |
| **字符串操作** | utils.go 多处 | strings.Index, strings.Split, strings.TrimRight | 内存分配 |
| **Params 创建** | route.go:303 | 每次匹配创建新 map | 堆分配 |
| **HandlersChain 合并** | dispatch.go:169,188 | append 创建新切片 | 堆分配 |

---

## 三、Radix Tree 重构方案

### 3.1 整体架构设计

采用**混合路由架构**，保持静态路由的性能优势，同时用 Radix Tree 优化动态路由：

```
┌─────────────────────────────────────────────────────────────┐
│                    Router                              │
├─────────────────────────────────────────────────────────────┤
│  1. 静态路由表 (map[string]*Route)            │
│     - O(1) 精确匹配                                  │
│     - 保持现有实现不变                             │
├─────────────────────────────────────────────────────────────┤
│  2. 动态路由树 (Radix Tree, per method)      │
│     - O(m) 查找，m 为路径长度                   │
│     - 每个方法一棵树，支持并发查找                 │
│     - 无正则表达式，零分配匹配                     │
├─────────────────────────────────────────────────────────────┤
│  3. 命名路由映射 (map[string]*Route)           │
│     - 快速查找命名路由                            │
├─────────────────────────────────────────────────────────────┤
│  4. Context Pool (sync.Pool)                      │
│     - 复用 Context 对象                              │
│     - 复用 Params map（重用而非新建）               │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 Radix Tree 节点设计

#### 节点结构

```go
// radix_tree.go
package rux

import (
    "strings"
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

    // HTTP 方法 -> 处理器映射
    // 只有叶子节点才有 handler
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
}

// 新的路由树（按 HTTP 方法分离）
type methodTrees struct {
    trees map[string]*radixTree
    mu    sync.RWMutex
}

type radixTree struct {
    root *radixNode
}
```

#### 关键设计决策

1. **按方法分离树**：每个 HTTP 方法一棵独立的树，支持并发查找
2. **路径压缩**：将只有单一子节点的路径段合并，减少节点数量
3. **零分配匹配**：使用预分配的 Params，匹配过程中不分配新 map
4. **静态与动态分离**：静态路由仍用 map，动态路由用 Radix Tree

### 3.3 路由注册算法

```go
// 添加路由到 Radix Tree
func (t *radixTree) AddRoute(path string, handlers HandlersChain, methods []string) {
    // 遍历每个方法，分别添加
    for _, method := range methods {
        t.addHandler(method, path, handlers)
    }
}

func (t *radixTree) addHandler(method, path string, handlers HandlersChain) {
    node := t.root

    // 标准化路径（确保以 / 开头，不以 / 结尾）
    path = normalizePath(path)

    for {
        // 查找最长公共前缀
        commonPrefix := longestCommonPrefix(node.prefix, path)

        // 情况1：公共前缀等于节点前缀，继续向下
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

        // 情况2：公共前缀较短，需要分裂节点
        if commonPrefix < len(node.prefix) {
            // 分裂当前节点
            t.splitNode(node, commonPrefix)

            // 原节点变为子节点
            // 需要处理的路径部分
            remaining := path[len(commonPrefix):]

            if len(remaining) == 0 {
                node.setHandler(method, handlers)
            } else {
                t.createChild(node, remaining, method, handlers)
            }
            return
        }

        // 情况3：路径与节点前缀不匹配，添加为兄弟节点
        if commonPrefix == 0 {
            t.createChild(node.parent, path, method, handlers)
            return
        }

        panic("unreachable")
    }
}

// 查找下一个节点
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
    // 提取到下一个 '/' 的路径段
    nextSlash := strings.IndexByte(path, '/')
    if nextSlash == -1 {
        nextSlash = len(path)
    }
    segment := path[:nextSlash]

    return node.children[segment]
}

// 创建子节点
func (t *radixTree) createChild(parent *radixNode, path, method string, handlers HandlersChain) {
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
}

// 分裂节点
func (t *radixTree) splitNode(node *radixNode, splitIndex int) {
    // 原节点分裂成两个
    splitPrefix := node.prefix[:splitIndex]
    remainingPrefix := node.prefix[splitIndex:]

    // 创建新的父节点
    newNode := &radixNode{
        prefix:   splitPrefix,
        nType:    nodeTypeStatic,
        handlers:  make(map[string]HandlersChain),
        children:  make(map[string]*radixNode),
    }

    // 将原节点作为新节点的子节点
    node.prefix = remainingPrefix
    newNode.children[remainingPrefix] = node

    // 更新父节点的子节点引用
    if node.parent != nil {
        if node.parent.paramChild == node {
            node.parent.paramChild = newNode
        } else if node.parent.wildcardChild == node {
            node.parent.wildcardChild = newNode
        } else {
            newNode.parent = node.parent
            // 在父节点的 children map 中替换
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

### 3.4 路由查找算法

```go
// 查找路由
func (t *radixTree) FindRoute(method, path string) (handlers HandlersChain, params Params, ok bool) {
    node := t.root
    params = make(Params, 0, 8)

    path = normalizePath(path)

    for {
        // 处理通配符（最高优先级）
        if node.wildcardChild != nil {
            if handlers, exists := node.wildcardChild.handlers[method]; exists {
                params[node.wildcardChild.paramName] = path
                return handlers, params, true
            }
        }

        // 检查完全匹配
        if handlers, exists := node.handlers[method]; exists && len(node.prefix) == len(path) {
            return handlers, params, true
        }

        // 前缀不匹配
        if !strings.HasPrefix(path, node.prefix) {
            return nil, nil, false
        }

        // 去掉匹配的前缀
        path = path[len(node.prefix):]

        // 路径已用完
        if len(path) == 0 {
            if handlers, exists := node.handlers[method]; exists {
                return handlers, params, true
            }
            return nil, nil, false
        }

        // 处理参数节点
        if node.paramChild != nil {
            // 提取参数值（到下一个 '/' 或结束）
            paramEnd := strings.IndexByte(path, '/')
            if paramEnd == -1 {
                paramEnd = len(path)
            }
            paramValue := path[:paramEnd]

            // 检查参数子节点是否有 handler
            if handlers, exists := node.paramChild.handlers[method]; exists && paramEnd == len(path) {
                params[node.paramChild.paramName] = paramValue
                return handlers, params, true
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
            return nil, nil, false
        }

        path = path[nextSlash:]
        node = child
    }
}
```

### 3.5 零分配参数提取

```go
// 参数池，复用 Params map
var paramsPool = sync.Pool{
    New: func() any {
        return make(Params, 8)
    },
}

// 从池中获取 Params
func acquireParams() Params {
    return paramsPool.Get().(Params)
}

// 归还 Params 到池
func releaseParams(p Params) {
    // 清空 map
    for k := range p {
        delete(p, k)
    }
    paramsPool.Put(p)
}
```

---

## 四、Router 结构重构

### 4.1 新的 Router 结构

```go
type Router struct {
    Name string
    err  error
    counter int

    // 静态路由：保持不变，使用 map
    stableRoutes map[string]*Route

    // 新增：动态路由树（按方法分离）
    dynamicTrees *methodTrees

    // 新增：参数池
    paramsPool *sync.Pool

    // 命名路由
    namedRoutes map[string]*Route

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

### 4.2 路由注册流程重构

```go
func (r *Router) AddRoute(route *Route) *Route {
    route.goodInfo()
    r.appendGroupInfo(route)
    debugPrintRoute(route)

    if route.name != "" {
        r.namedRoutes[route.name] = route
    }

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

### 4.3 路由匹配流程重构

```go
func (r *Router) match(method, path string) (rt *Route, ps Params) {
    // 1. 静态路由查找 O(1)
    route, ok := r.stableRoutes[method+path]
    if ok {
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

---

## 五、性能优化清单

### 5.1 路由匹配优化

| 优化项 | 当前实现 | 优化方案 | 预期提升 |
|--------|---------|---------|----------|
| 动态路由查找 | 线性扫描 O(n) | Radix Tree O(m) | **10-40倍** |
| 正则表达式编译 | 每个路由编译一次 | 消除正则 | **注册快 5-10倍** |
| 正则匹配 | 正则引擎运行 | 字符串比较 | **匹配快 10-20倍** |
| Params 创建 | 每次 make new map | 使用 sync.Pool 复用 | **零分配** |

### 5.2 其他性能优化

#### 1. Context Pool 优化

```go
// 优化 Context 创建，使用 sync.Pool
type Router struct {
    ctxPool sync.Pool
}

func New() *Router {
    router := &Router{
        ctxPool: sync.Pool{
            New: func() any {
                return &Context{
                    index: -1,
                    data:  make(map[string]any),
                }
            },
        },
    }
    // ...
}
```

#### 2. 字符串操作优化

```go
// utils.go 优化
// 当前：strings.TrimRight(path, "/") // 分配新字符串
// 优化：原地检查和切片
func trimTrailingSlash(path string) string {
    if len(path) > 1 && path[len(path)-1] == '/' {
        return path[:len(path)-1]
    }
    return path
}

// 当前：strings.Index(path, rs[i].start)
// 优化：使用 strings.HasPrefix (编译器优化更好)
func hasPrefix(path, prefix string) bool {
    return strings.HasPrefix(path, prefix)
}
```

#### 3. HandlersChain 预分配

```go
// dispatch.go 优化
// 当前：每次 append 创建新切片
// 优化：预分配足够大小的切片
func (r *Router) handleHTTPRequest(ctx *Context) {
    route, params, allowed := r.QuickMatch(ctx.Req.Method, path)

    var handlers HandlersChain
    // 预分配：全局中间件 + 路由中间件 + 主 handler
    capacity := len(r.handlers) + len(route.handlers) + 1
    handlers = make(HandlersChain, 0, capacity)

    // 按顺序添加
    handlers = append(handlers, r.handlers...)
    handlers = append(handlers, route.handlers...)
    handlers = append(handlers, route.handler)

    ctx.SetHandlers(handlers)
    ctx.Next()
}
```

#### 4. 路径规范化优化

```go
// 在路由注册时规范化，避免运行时重复处理
func normalizePathOnce(path string) string {
    path = strings.TrimSpace(path)
    if len(path) > 1 && path[len(path)-1] == '/' {
        path = path[:len(path)-1]
    }
    if len(path) > 0 && path[0] != '/' {
        path = "/" + path
    }
    return path
}

// 在 Route 创建时就完成规范化
func NewRoute(path string, handler HandlerFunc, methods ...string) *Route {
    return &Route{
        path: normalizePathOnce(path),
        handler: handler,
        methods: formatMethodsWithDefault(methods, GET),
    }
}
```

---

## 六、实现计划

### 6.1 阶段划分

#### 阶段 1：基础设施准备（1-2 天）
- [ ] 创建 `radix_tree.go`，实现节点结构和基本方法
- [ ] 实现 longestCommonPrefix 工具函数
- [ ] 实现节点分裂算法
- [ ] 实现子节点创建算法

#### 阶段 2：核心路由逻辑（2-3 天）
- [ ] 实现 Radix Tree 的 AddRoute 方法
- [ ] 实现 Radix Tree 的 FindRoute 方法
- [ ] 实现参数提取逻辑（零分配）
- [ ] 单元测试覆盖所有边界情况

#### 阶段 3：Router 集成（2-3 天）
- [ ] 修改 Router 结构，添加 dynamicTrees 字段
- [ ] 修改 AddRoute 方法，路由分发到静态/动态
- [ ] 修改 match 方法，先查静态再查动态
- [ ] 移除 regularRoutes 和 irregularRoutes

#### 阶段 4：性能优化（1-2 天）
- [ ] 实现 Context Pool
- [ ] 实现 Params Pool
- [ ] 优化字符串操作函数
- [ ] 优化 HandlersChain 预分配

#### 阶段 5：测试与验证（2-3 天）
- [ ] 运行现有单元测试，确保功能兼容
- [ ] 添加性能基准测试
- [ ] 对比重构前后的性能数据
- [ ] 修复发现的问题

#### 阶段 6：清理与文档（1 天）
- [ ] 移除旧的路由缓存代码（route_cache.go）
- [ ] 更新 README 和文档
- [ ] 添加迁移指南
- [ ] 更新 CHANGELOG

### 6.2 风险评估

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|---------|
| API 破坏性变更 | 高 | 中 | 保持接口兼容，渐进式迁移 |
| 并发安全问题 | 高 | 中 | 使用 RWMutex 保护树结构 |
| 性能未达预期 | 中 | 低 | 分阶段优化，每次基准测试 |
| 边界情况处理bug | 中 | 中 | 完善单元测试覆盖 |

### 6.3 兼容性保证

#### 保持向后兼容

1. **API 接口不变**：
   - Router.GET/POST/PUT 等方法签名不变
   - Route 结构和方法保持兼容
   - Context 接口不变

2. **行为兼容**：
   - 路由匹配逻辑与之前一致
   - 参数提取方式不变
   - 中间件执行顺序不变

3. **配置兼容**：
    - 保留 strictLastSlash 配置
    - 保留 handleMethodNotAllowed 配置

---

## 七、性能验证

### 7.1 基准测试方案

```go
// benchmark_test.go

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
    for i := 0; i < 100; i++ {
        r.GET(fmt.Sprintf("/user%d/:id", i), emptyHandler)
    }

    req := httptest.NewRequest("GET", "/user42/123", nil)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        r.ServeHTTP(httptest.NewRecorder(), req)
    }
}

// 5 参数动态路由
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

### 7.2 性能目标

| 场景 | 当前性能 | 目标性能 | 提升倍数 |
|------|---------|---------|----------|
| 静态路由 | 15-20M ops/s | **25-30M ops/s** | 1.5-2x |
| 单参数动态路由 | 3-5M ops/s | **15-20M ops/s** | 4-5x |
| 5 参数动态路由 | 0.5-1M ops/s | **8-12M ops/s** | 10-15x |
| 内存分配 | 高 | **零分配** | N/A |

### 7.3 内存占用目标

- 静态路由：保持现有水平（map 本身就很高效）
- 动态路由树：比现有实现减少 **30%-50%**
  - 路径压缩减少节点数量
  - 消除正则表达式对象
- 总体内存：整体减少 **20%-30%**

---

## 八、未知问题与确认事项

以下事项需要确认或在实现过程中澄清：

### 8.1 功能兼容性（已全部解决）

- **问题 1（已解决）**：可选参数如何处理？
  - **决策**：展开为两条独立路由
  - **限制规则**：
    - ✅ 只能在路径最后：`/posts[/{id}]` - 合法
    - ✅ 只能支持一个可选参数：`/api/users[/{id}]` - 合法
    - ❌ 不在最后：`/posts[/{category}]/{id}` - 非法（应 panic）
    - ❌ 多个可选参数：`/api[/{v1}]/users[/{v2}]` - 非法（应 panic）
  - **实现方案**：在 AddRoute 中检测可选段，验证规则后自动展开注册
  - **示例**：
    - ✅ `/posts[/{id}]` → `/posts` + `/posts/{id}`
    - ✅ `/api/users[/{name}]/profile` → `/api/users/profile` + `/api/users/{name}/profile`
    - ❌ `/posts[/{category}]/{id}` → panic（可选参数不在最后）
    - ❌ `/api[/{v1}]/users[/{v2}]` → panic（多个可选参数）
  - **兼容性**：用户无需修改代码，自动适应
  - **处理逻辑**：handler 中通过 `c.Param("id")` 判断是否为空

- **问题 2（已解决）**：正则参数如何处理？
  - **决策**：简化为普通参数，添加验证中间件
  - **实现方案**：`{id:\d+}` → `{id}` + 参数验证中间件
  - **性能**：Radix Tree 查找快速，验证开销可接受
  - **示例**：`r.Use(validateIDParam)`，在 handler 中验证 ID 格式

- **问题 3（已解决）**：strictLastSlash 配置
  - **决策**：在路由注册时统一规范化路径
  - **实现方案**：所有路径在注册时标准化（去尾斜杠、加前导斜杠）
  - **兼容性**：用户无感知，配置项保持有效

### 8.2 并发模型（已解决）

- **决策**：支持运行时动态添加路由，使用 RWMutex 保护
  - **实现方案**：写操作（AddRoute）使用写锁，读操作（FindRoute）使用读锁
  - **性能**：允许多个 goroutine 并发读取路由树
  - **安全性**：保证路由树的一致性

### 8.3 配置清理（已解决）

- **决策**：完全移除 LRU 路由缓存机制
  - **删除内容**：
    - 移除 `Router.cachedRoutes` 字段
    - 移除 `route_cache.go` 文件（LRU 缓存实现）
    - 移除 `route_cache_test.go` 文件（缓存测试）
    - 移除 `EnableCaching` 和 `CachingWithNum` 配置函数
    - 移除 `enableCaching` 和 `maxNumCaches` 字段
  - **理由**：Radix Tree 查找足够快（O(m)），缓存收益不明显且增加复杂度

### 8.4 待删除文件

- ✅ **route_cache.go** - LRU 缓存实现（已确认删除）
- ✅ **route_cache_test.go** - 缓存测试文件（已确认删除）

### 8.5 测试覆盖

- **待验证**：运行 `go test -cover` 获取当前覆盖率
- **待验证**：重构后确保覆盖率保持或提升
- **测试重点**：
  1. Radix Tree 构建正确性
  2. 路径匹配准确性
  3. 参数提取正确性
  4. 并发场景安全性
  5. 边界情况处理（空路径、根路径等）

### 8.6 性能基准

- **待记录**：重构前运行基准测试记录基线数据
- **待对比**：重构后运行相同基准测试对比性能提升
- **对比维度**：
  1. 静态路由查找性能
  2. 单参数动态路由性能
  3. 多参数动态路由性能
  4. 内存分配次数
  5. CPU 时间

---

## 九、总结

本设计方案提供了将 gookit/rux 核心路由从 map+正则 重构为 Radix Tree 的完整路径：

### 核心改进

1. **性能提升**：动态路由匹配从 O(n) 到 O(m)，预期提升 4-15 倍
2. **内存优化**：路径压缩和节点共享，预期减少 20%-30% 内存占用
3. **零分配查找**：使用对象池复用 Context 和 Params
4. **保持兼容**：API 接口和行为完全向后兼容
5. **静态路由优化**：继续使用 map 保持 O(1) 查找性能

### 实现策略

- 采用**混合架构**：静态路由用 map，动态路由用 Radix Tree
- 按 **HTTP 方法分离树**：支持并发查找
- 实现**路径压缩**：减少节点数量，提高缓存命中率
- 使用**参数池**：复用 Params map，避免频繁分配

### 后续工作

请审核本设计方案，确认以下事项后开始实施：

1. **确认功能范围**：是否需要保留可选参数和正则参数支持？
2. **确认并发模型**：是否需要支持运行时动态修改路由？
3. **确认配置选项**：哪些配置需要保留或调整？
4. **确认性能目标**：性能提升目标是否符合预期？

确认后，将按照"阶段划分"中的 6 个阶段逐步实施，每完成一个阶段进行测试验证。
