# Rux v2 设计方案（Clean Room Rewrite）

| 字段 | 内容 |
|---|---|
| 版本 | v2.0 |
| 日期 | 2026-05-15 |
| 作者 | inhere & Claude |
| 状态 | Draft — 待评审 |
| 取代文档 | `docs/design/design-refactor-core.md`、`docs/design/design-implementation.md`、`_tmp/HYBRID_ARCHITECTURE_PLAN.md` |
| 关联实现 | 主包 `rux/`（重写）、废弃 `fastrux/` 子包 |

---

## 0. 阅读指引

本文档定义 `gookit/rux` v2 的设计契约。v2 是**破坏性升级**：API、内部结构、行为细节都允许与 v1 不同。老用户走 `v1.x` 分支维护。

设计原则按优先级降序：

1. **性能极致**：对标 httprouter，至少不弱于 gin。
2. **零分配热路径**：静态路由 0 alloc，动态路由 ≤1 alloc（仅 Context 池命中失败时）。
3. **API 简洁**：能用类型化字段就不用 `map[string]any`，能注册时算就不在请求时算。
4. **行为可预测**：通配符/参数/静态优先级符合直觉，不再有"通配符吃掉静态"的惊喜。
5. **代码可读**：所有源码注释英文；模块边界清晰；单文件 ≤ 600 行。

---

## 1. 背景与目标

### 1.1 v1 现状回顾

v1 路由核心是 `map[string][]*Route` + 正则匹配，存在多项性能瓶颈：动态路由 O(n) 线性扫描、正则编译/匹配开销大、每次匹配创建 `Params map`、handler 链每请求重新 append。

`fea_v2` 分支已尝试用 Radix Tree 替换 v1 的 map+regex 方案，并在子包 `fastrux/` 中做了进一步原型。这两份工作识别了大方向（Radix Tree + 按 method 分树），但还有以下未解决问题（见 §1.2）。

### 1.2 现有实现遗留问题（v2 必须解决）

> 编号 P-x 在后文 §3 / §4 各小节会回引修复点。

- **P-1 双重 method 分发**：`methodTrees.trees[method]` 已经按方法分树，但 `radixNode.handlers map[string]HandlersChain` 又按方法分。叶子节点不可能既属于 GET 树又属于 POST 树，这是冗余且每次匹配多一次 hash。
- **P-2 通配符优先级 bug**：`radix_tree.go` 中 `wildcardChild` 在 `paramChild`/`children` 之前检查；注册 `/users/:id` + `/users/*all` 后，`/users/123` 会被通配符吃掉。正确顺序：静态 > 参数 > 通配符。
- **P-3 paramsPool 未生效**：dev-progress 标记完成，实际代码每次 `make(Params)`；fastrux 的 PERFORMANCE.md 明确放弃 pool（"用户可能在 handler 外持有 Params"），导致每个动态路由 ≥1 alloc。
- **P-4 `Params = map[string]string`**：参数少时（绝大多数 ≤3）切片 `[]Param{Key,Value}` 完胜 map。这是 fastrux 动态路由 3 alloc 中最大的一笔。
- **P-5 静态路由 key 拼接**：`r.stableRoutes[method+path]` 每请求一次字符串分配。
- **P-6 中间件链每请求重 append**：`buildHandlers` 调用 `append(route.handlers, route.handler)`、`handleHTTPRequest` 又 `append(r.handlers, handlers...)`，每请求 ≥2 次切片分配。
- **P-7 节点 `children map[string]*radixNode`**：子节点少时数组+首字节索引比 map hash 更 cache-friendly（httprouter 模式）。
- **P-8 RWMutex 长期开销**：注册后路由通常只读，但每次匹配仍有 RLock 原子操作。缺少 freeze 模式。
- **P-9 HEAD→GET fallback 二次匹配**：当前 dispatch 中 HEAD 没命中才回退到 GET 再走一遍 tree。注册时镜像即可。
- **P-10 路径规范化执行 ≥3 次**：register 一次，dispatch `formatPath` 一次，`FindRoute` 内部还补一次。
- **P-11 `node.handlers` 与 `node.routes` 双 map 镜像**：内存翻倍，可合一。
- **P-12 `Match`/`QuickMatch` 公开 API 半成品**：MatchResult pool 设计了但用户必须显式 `ReleaseMatchResult`，易漏。
- **P-13 fastrux 与主包重复实现**：两套 router/context/radix tree，长期维护负担。
- **P-14 regex 参数 `{id:\d+}` 静默丢失**：dev-progress 列为已知限制，但用户原本依赖 —— 必须有明确替代方案。
- **P-15 Context `c.Set/Get` 用 `map[string]any`**：每次首 `Set` 触发 map 分配 + 接口装箱，热点字段（routeName/routePath）应类型化。
- **P-16 节点子节点无优先级**：真实 radix tree（如 httprouter）按访问频次排序，热路径前置；当前 children map 无序。

### 1.3 目标与非目标

**目标**：

- 静态路由：≥ 30M ops/s，**0 alloc/op**
- 单参数动态路由：≥ 15M ops/s，**0–1 alloc/op**（视 Context 池）
- 5 参数动态路由：≥ 10M ops/s，**0–1 alloc/op**
- 404 路径：**0 alloc/op**
- 注册期接受 panic 式错误（路径不合法、可选段位置错），不暴露 error 返回值
- 单文件 ≤ 600 行

**非目标**：

- 不支持 regex 参数 `{id:\d+}`（用 §6.2 替代方案）
- 不支持运行时动态注册路由（freeze 后只读，违反 panic）
- 不追求与 v1 二进制兼容
- 不支持 Trie compression 之外的"FST/前缀自动机"等复杂结构

---

## 2. 整体架构

### 2.1 包结构

```
github.com/gookit/rux        # v2 主包（重写）
├── rux.go                   # 包级常量（method 表、index 函数）
├── router.go                # Router 定义、注册、freeze
├── route.go                 # Route 定义、URL 构造
├── tree.go                  # Radix Tree 节点 + 算法
├── context.go               # Context（含内联 Params）
├── context_render.go        # JSON/XML/Text/HTML/File 响应
├── context_binding.go       # 参数绑定（保留）
├── dispatch.go              # ServeHTTP / Listen / 错误处理
├── middleware.go            # 内置中间件
├── extends.go               # BuildRequestURL
├── response_writer.go       # 包装 http.ResponseWriter
├── utils.go                 # 路径处理、debug print
└── internal/util/           # 路径校验等内部工具
```

**移除**：`fastrux/` 整个子包（功能合并进主包后删除）。

**保留**（与 v1 一致，无需改）：`pkg/binding`、`pkg/render`、`pkg/handlers`、`pkg/pprof`、`pkg/websocket`、`pkg/adaptor`、`server/`。

### 2.2 顶层结构图

```
┌─────────────────────────────────────────────────────────────┐
│                       Router                                │
├─────────────────────────────────────────────────────────────┤
│  staticRoutes  [9]map[string]*Route        // P-5 修复       │
│                ↑ method index 0..8         // P-1 部分修复    │
├─────────────────────────────────────────────────────────────┤
│  dynamicTrees  [9]*radixTree               // P-1 修复       │
│                ↑ 按 method index 直接数组下标                 │
├─────────────────────────────────────────────────────────────┤
│  globalChain   HandlersChain               // P-6 修复       │
│                注册期累积；freeze 时与每个 route chain 合并    │
├─────────────────────────────────────────────────────────────┤
│  frozen        atomic.Bool                 // P-8 修复       │
│  ctxPool       sync.Pool[*Context]                          │
│  noRoute/      HandlersChain (404/405 chain)                │
│  noAllowed                                                   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
              ┌─────────────────────────────┐
              │     radixTree (per method)  │
              │  root *node                  │
              │  maxParams uint8             │
              └─────────────────────────────┘
                            │
                            ▼
   ┌─────────────────────────────────────────────────────┐
   │                       node                          │
   │ prefix      string                                  │
   │ nType       nodeType (static/param/wildcard/root)   │
   │ priority    uint32                  // P-16        │
   │ indices     []byte                  // P-7         │
   │ children    []*node                 // 与 indices 同序│
   │ paramChild  *node                                   │
   │ wildcardChild *node                                 │
   │ paramName   string                                  │
   │ chain       HandlersChain  // 单一 chain，无 method map │
   │ route       *Route         //          P-1/P-11 修复  │
   └─────────────────────────────────────────────────────┘
```

### 2.3 数据流

**注册期**（Router 未 frozen）：

```
r.GET("/users/:id", h, mw1)
  → newRoute(path, [GET], h)
  → appendGroupInfo(route)             // 拼前缀、合 group middleware
  → if hasOptionalSegment → expand     // /posts[/{id}] → 2 条
  → for each method:
      idx = methodIndex(method)
      if isStatic(path):
          staticRoutes[idx][path] = route
      else:
          dynamicTrees[idx].insert(path, route)
```

**Freeze 期**（首次 `ServeHTTP` 或显式 `r.Freeze()`）：

```
r.Freeze()
  → 遍历所有 route，把 globalChain 与 route.chain 合并；
     合并结果存到 route.finalChain（指针，无 copy 开销）
  → 遍历所有 GET 路由，镜像注册到 HEAD 树（P-9）
  → frozen.Store(true)
  → 之后任何 r.GET/POST/Use → panic("rux: router frozen")
```

**请求期**（frozen=true，全程无锁）：

```
ServeHTTP(w, req)
  → ctx = ctxPool.Get(); ctx.Reset(w, req)
  → idx = methodIndex(req.Method)
  → 1. 试 staticRoutes[idx][path]      // O(1) 单 map lookup, 无字符串拼接
  → 2. 否则 dynamicTrees[idx].lookup(path, &ctx.params)
  → 3. 命中：handlers = route.finalChain (无 append)
        ctx.handlers = handlers; ctx.Next()
  → 4. miss：noRoute / noAllowed
  → ctxPool.Put(ctx)
```

---

## 3. 核心数据结构

### 3.1 method index（P-1）

HTTP 方法只有 9 个固定值，用 `[9]` 数组下标访问：

```go
// rux.go
const (
    GET     = "GET"
    HEAD    = "HEAD"
    POST    = "POST"
    PUT     = "PUT"
    PATCH   = "PATCH"
    DELETE  = "DELETE"
    OPTIONS = "OPTIONS"
    CONNECT = "CONNECT"
    TRACE   = "TRACE"

    methodCount = 9
)

// methodIndex maps HTTP method string to a 0..8 array index.
// Returns -1 for unknown methods (caller should panic at register, 405 at runtime).
func methodIndex(m string) int {
    // Hot path: switch on first byte; fall through to full compare.
    if len(m) == 0 {
        return -1
    }
    switch m[0] {
    case 'G':
        if m == GET { return 0 }
    case 'H':
        if m == HEAD { return 1 }
    case 'P':
        if m == POST { return 2 }
        if m == PUT { return 3 }
        if m == PATCH { return 4 }
    case 'D':
        if m == DELETE { return 5 }
    case 'O':
        if m == OPTIONS { return 6 }
    case 'C':
        if m == CONNECT { return 7 }
    case 'T':
        if m == TRACE { return 8 }
    }
    return -1
}
```

> **取舍**：`switch` 比 map lookup 快约 5-10x（无 hash、无 mutex），且 `req.Method` 是 immutable string 头部，对比是单条 SIMD 指令。

### 3.2 节点（P-7、P-11、P-16）

```go
// tree.go
type nodeType uint8

const (
    nodeStatic nodeType = iota
    nodeParam
    nodeWildcard
    nodeRoot
)

// node is a Radix Tree node. One node belongs to exactly one method tree,
// so it stores a single handler chain (no method indirection).
type node struct {
    // Path prefix this node owns (compressed).
    prefix string

    // Node category. Determines child traversal semantics.
    nType nodeType

    // Parameter name for nodeParam/nodeWildcard.
    paramName string

    // Static children. indices[i] is the first byte of children[i].prefix.
    // Sorted by priority desc to put hot paths first.
    indices  []byte
    children []*node

    // Catch-all children. At most one of each per node.
    paramChild    *node
    wildcardChild *node

    // Final handler chain assigned to this leaf. nil means non-leaf.
    // After Freeze(), this includes global middleware merged in.
    chain HandlersChain

    // Route metadata (name, methods list, etc.). nil iff chain is nil.
    route *Route

    // Lookup priority — number of routes registered through this node.
    // Used to keep hot children at the front of the indices slice.
    priority uint32
}
```

**与现状对比**：
- 移除 `handlers map[string]HandlersChain`、`routes map[string]*Route` → 单一 `chain` + `route`（P-1、P-11）。
- `children map[string]*node` → `indices []byte + children []*node`（P-7、P-16）。
- 保留 `paramChild` / `wildcardChild`（独立字段比放到 indices 中更快，因为查找时它们的优先级与 byte 比较无关）。

### 3.3 方法树容器（P-1、P-8）

```go
// router.go
type Router struct {
    Name string

    // Static routes per method. methods are indexed via methodIndex().
    staticRoutes [methodCount]map[string]*Route

    // Dynamic routes per method. nil if no dynamic routes for that method.
    dynamicTrees [methodCount]*radixTree

    // Named routes for URL building.
    namedRoutes map[string]*Route

    // All routes in insertion order (for IterateRoutes / debug).
    routeList []*Route

    // Global middleware. Frozen into each route's finalChain on Freeze().
    globalChain HandlersChain

    // Group registration state (only meaningful before Freeze).
    currentGroupPrefix   string
    currentGroupHandlers HandlersChain

    // Misc handlers.
    noRoute   HandlersChain
    noAllowed HandlersChain

    // Settings (immutable after Freeze).
    OnError                HandlerFunc
    OnPanic                HandlerFunc
    interceptAll           string
    useEncodedPath         bool
    strictLastSlash        bool
    handleMethodNotAllowed bool
    handleFallbackRoute    bool

    // Frozen flag — atomic for fast read in hot path.
    frozen atomic.Bool

    // Object pools.
    ctxPool sync.Pool

    // Maximum params count across all routes (for sanity check).
    maxParams uint8
}

// radixTree wraps the root and tracks per-tree max params count.
type radixTree struct {
    root      *node
    maxParams uint8
}
```

### 3.4 Params（P-3、P-4）

**核心决策**：参数内联到 Context，最大 16 个，超过 panic（注册期就拒绝）。

```go
// context.go

// MaxParams is the maximum number of path params per route.
// Inlined storage avoids any heap allocation for params.
const MaxParams = 16

// Param is a single path parameter.
type Param struct {
    Key   string
    Value string
}

// Params is a small fixed-capacity inline array of params,
// stored directly in Context to avoid heap allocation.
type Params struct {
    data [MaxParams]Param
    n    uint8
}

// Get returns the value of the named param. Empty string if not found.
func (p *Params) Get(name string) string {
    for i := uint8(0); i < p.n; i++ {
        if p.data[i].Key == name {
            return p.data[i].Value
        }
    }
    return ""
}

// Has reports whether the named param exists.
func (p *Params) Has(name string) bool {
    for i := uint8(0); i < p.n; i++ {
        if p.data[i].Key == name {
            return true
        }
    }
    return false
}

// Int returns the param value parsed as int, 0 on miss/parse-error.
func (p *Params) Int(name string) int {
    if v := p.Get(name); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            return n
        }
    }
    return 0
}

// Len returns the number of params.
func (p *Params) Len() int { return int(p.n) }

// Reset clears the params (called by Context.Reset on pool return).
func (p *Params) Reset() { p.n = 0 }

// append adds a param. Caller must check n < MaxParams (asserted in tree.lookup).
func (p *Params) append(key, value string) {
    p.data[p.n].Key = key
    p.data[p.n].Value = value
    p.n++
}
```

**取舍说明**：
- **为什么不用 `[]Param`**：slice 头本身 24B + 底层数组堆分配（即使 pool 也涉及指针 indirection）。内联到 Context 后，Context pool 命中时全部零分配。
- **为什么 16**：实测 GitHub/AWS 等真实 API 路径最多 6-7 个参数，16 是 4x 安全边际；`16 * 32B (Param) = 512B` per Context，可接受。
- **超过 16**：注册时 `tree.maxParams > MaxParams` 直接 panic，不留运行期惊喜。
- **用户场景**：handler 内 `c.Param("id")` 一致工作；如需在 handler 外（如启动 goroutine）持有，用户显式 copy：`paramsCopy := append([]Param(nil), c.AllParams()...)`。

### 3.5 Context（P-15）

```go
// context.go
type Context struct {
    Req    *http.Request
    Resp   http.ResponseWriter
    writer responseWriter // wraps Resp, tracks status/size

    router *Router

    // Path params, inlined storage.
    params Params

    // Hot fields used by every request — typed, not in map.
    matchedRoute *Route
    matchedPath  string

    // Handler chain (already merged at Freeze time, no append per request).
    handlers HandlersChain
    index    int8

    // Errors accumulated during handling.
    Errors []error

    // Lazy-init bag for user data. nil until first Set().
    data map[string]any
}

func (c *Context) Param(name string) string { return c.params.Get(name) }
func (c *Context) Route() *Route            { return c.matchedRoute }
func (c *Context) MatchedPath() string      { return c.matchedPath }

// Set stores arbitrary user data. Allocates the map on first call.
func (c *Context) Set(key string, value any) {
    if c.data == nil {
        c.data = make(map[string]any, 4)
    }
    c.data[key] = value
}

func (c *Context) Get(key string) (any, bool) {
    if c.data == nil { return nil, false }
    v, ok := c.data[key]
    return v, ok
}
```

**与现状对比**：
- `currentRouteName` / `currentRoutePath` 不再走 `c.Set` 写入 map，直接类型化字段（P-15）。
- `c.Set/Get` 保留（用户 API 兼容性高），但 lazy 初始化，不用就不分配。

### 3.6 Route 与 HandlersChain

```go
// route.go
type Route struct {
    name    string
    path    string
    methods []string

    // User-supplied middleware + main handler, in registration order.
    // After Freeze(), finalChain = globalChain + this.
    chain      HandlersChain
    finalChain HandlersChain // populated by router.Freeze()

    Opts map[string]any
}

// HandlerFunc is the standard handler signature.
type HandlerFunc func(c *Context)

// HandlersChain is a slice of handlers (middlewares + final handler).
// The "final handler" is just the last element; there's no separate field.
type HandlersChain []HandlerFunc
```

**取舍**：彻底取消 v1 的 `handler HandlerFunc` + `handlers HandlersChain` 二分，统一为 `chain` 一条链。注册时 main handler 自动 append 到 chain 末尾。这消除了 P-6 中"每请求 append handler 到 handlers"的开销根源。

---

## 4. 关键算法

### 4.1 静态路由识别

```go
// utils.go

// isStaticPath reports whether path contains no dynamic segments.
// Static = no '{', '[', ':', '*' characters.
func isStaticPath(path string) bool {
    for i := 0; i < len(path); i++ {
        switch path[i] {
        case '{', '[', ':', '*':
            return false
        }
    }
    return true
}
```

### 4.2 路径规范化（P-10）

**只在注册期执行一次**：

```go
// utils.go

// normalizePath enforces:
//   - leading '/'
//   - no trailing '/' (unless root)
//   - no doubled '//'
// Idempotent. Allocates only when changes are needed.
func normalizePath(path string) string {
    if path == "" {
        return "/"
    }
    // Fast path: already canonical (no allocation).
    if path[0] == '/' && !strings.Contains(path, "//") &&
        (len(path) == 1 || path[len(path)-1] != '/') {
        return path
    }

    var b strings.Builder
    b.Grow(len(path) + 1)
    if path[0] != '/' {
        b.WriteByte('/')
    }

    prevSlash := false
    for i := 0; i < len(path); i++ {
        c := path[i]
        if c == '/' {
            if prevSlash {
                continue
            }
            prevSlash = true
        } else {
            prevSlash = false
        }
        b.WriteByte(c)
    }

    out := b.String()
    if len(out) > 1 && out[len(out)-1] == '/' {
        out = out[:len(out)-1]
    }
    return out
}
```

请求期 `formatPath` 简化为最小路径修正（仅在 `useEncodedPath=true` 或 `strictLastSlash=false` 必要时才 trim 一次尾斜杠），多数场景直接使用 `req.URL.Path`。

### 4.3 节点插入（含分裂、优先级排序）

伪代码：

```
insert(path, route):
    node = root
    for {
        cp = longestCommonPrefix(node.prefix, path)
        if cp < len(node.prefix):
            splitNode(node, cp)        // node 变成新父节点的子
        path = path[cp:]
        if len(path) == 0:
            node.chain = route.finalChain  // freeze 时填，注册时填 chain
            node.route = route
            node.bumpPriority()
            return
        switch path[0]:
        case ':':
            if node.paramChild == nil:
                node.paramChild = newParamNode(path)
                continue with paramChild as new node, path advanced past param segment
        case '*':
            node.wildcardChild = newWildcardNode(path)
            return
        default:
            // Static child lookup by first byte
            i = indexOf(node.indices, path[0])
            if i >= 0:
                node = node.children[i]
                continue
            // Create new static child
            child = newStaticNode(path)
            node.indices = append(node.indices, path[0])
            node.children = append(node.children, child)
            node.bumpPriority()
            sortChildrenByPriority(node)  // P-16
            return
    }
```

**`bumpPriority` + `sortChildrenByPriority`**：每次插入到子树时，叶子向上回溯 `priority++`，父节点保持 `indices/children` 按 priority 降序，确保查找时**首字节比较的命中顺序**与实际访问频率一致（参考 httprouter 的 incrementChildPrio）。

### 4.4 路由查找（P-2 修复优先级）

伪代码：

```
lookup(path, params *Params) (*Route, bool):
    node = root
    for {
        // Match prefix
        if !strings.HasPrefix(path, node.prefix):
            return nil, false
        path = path[len(node.prefix):]

        if len(path) == 0:
            return node.route, node.route != nil

        // Priority order: STATIC > PARAM > WILDCARD       (P-2 fix)
        // 1. Static children — first-byte index lookup
        if i := indexByte(node.indices, path[0]); i >= 0:
            node = node.children[i]
            continue

        // 2. Param child — match up to next '/'
        if node.paramChild != nil:
            end = indexByte(path, '/')
            if end < 0: end = len(path)
            params.append(node.paramChild.paramName, path[:end])
            if end == len(path):
                return node.paramChild.route, node.paramChild.route != nil
            path = path[end:]
            node = node.paramChild
            continue

        // 3. Wildcard child — match all remaining
        if node.wildcardChild != nil:
            params.append(node.wildcardChild.paramName, path)
            return node.wildcardChild.route, node.wildcardChild.route != nil

        return nil, false
    }
```

**回溯**：当走静态命中但深处无叶子时，需要回溯尝试 paramChild/wildcardChild。实现方式参考 httprouter，用栈或递归（递归更简单且 Go 编译器能尾递归优化）。

### 4.5 Freeze（P-8、P-9、P-6）

```go
// router.go
func (r *Router) Freeze() {
    if r.frozen.Load() {
        return
    }

    // 1. Merge globalChain into every route's chain → finalChain.
    for _, route := range r.routeList {
        if len(r.globalChain) == 0 {
            route.finalChain = route.chain
        } else {
            merged := make(HandlersChain, 0, len(r.globalChain)+len(route.chain))
            merged = append(merged, r.globalChain...)
            merged = append(merged, route.chain...)
            route.finalChain = merged
        }
    }

    // 2. Push finalChain into every leaf node.
    for i := range r.dynamicTrees {
        if r.dynamicTrees[i] != nil {
            r.dynamicTrees[i].propagateChains()
        }
    }
    // staticRoutes already point to *Route, so chain is reachable via route.finalChain.

    // 3. Mirror GET → HEAD (P-9). Only routes that don't already have HEAD.
    r.mirrorGetToHead()

    r.frozen.Store(true)
}

// ServeHTTP triggers lazy freeze on first call.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    if !r.frozen.Load() {
        r.Freeze()
    }
    // ... hot path
}

// Add / GET / Use after freeze panic.
func (r *Router) Add(path string, h HandlerFunc, methods ...string) *Route {
    if r.frozen.Load() {
        panic("rux: cannot add route after router is frozen")
    }
    // ...
}
```

**为什么 atomic.Bool 而非 mutex**：freeze 状态读多写一次。`atomic.Bool.Load()` 在 amd64/arm64 上是普通 mov 指令，比 RLock 便宜两个数量级。

### 4.6 静态路由命中（P-5）

```go
func (r *Router) match(method, path string) (*Route, *Params) {
    idx := methodIndex(method)
    if idx < 0 {
        return nil, nil
    }

    // 1. Static — single map lookup, no string concat. (P-5)
    if m := r.staticRoutes[idx]; m != nil {
        if route, ok := m[path]; ok {
            return route, nil
        }
    }

    // 2. Dynamic — radix tree.
    if tree := r.dynamicTrees[idx]; tree != nil {
        // Caller's Context owns the Params buffer.
        // tree.lookup writes into &ctx.params and returns the route.
        if route, ok := tree.lookup(path, &ctx.params); ok {
            return route, &ctx.params
        }
    }
    return nil, nil
}
```

> 注意：实际签名会让 lookup 直接写入 `ctx.params`（Context 内联），不传 `*Params` 参数。这里只是说明意图。

### 4.7 HEAD 镜像（P-9）

```go
// During Freeze, for every GET route that has no explicit HEAD counterpart,
// register the same Route into the HEAD tree.
func (r *Router) mirrorGetToHead() {
    getStatic := r.staticRoutes[methodIndex(GET)]
    headStatic := r.staticRoutes[methodIndex(HEAD)]
    if headStatic == nil && len(getStatic) > 0 {
        headStatic = make(map[string]*Route, len(getStatic))
        r.staticRoutes[methodIndex(HEAD)] = headStatic
    }
    for path, route := range getStatic {
        if _, exists := headStatic[path]; !exists {
            headStatic[path] = route
        }
    }
    // Same for dynamic tree: walk GET tree, insert into HEAD tree.
    // walk(fn) traverses the tree depth-first and yields the original
    // registration path for each leaf (reconstructed from prefixes).
    // lookupExact(path) is an internal helper that checks for an exact
    // path registration without parameter binding (used to skip routes
    // the user already registered explicitly for HEAD).
    if get := r.dynamicTrees[methodIndex(GET)]; get != nil {
        head := r.dynamicTrees[methodIndex(HEAD)]
        if head == nil {
            head = &radixTree{root: newRoot()}
            r.dynamicTrees[methodIndex(HEAD)] = head
        }
        get.root.walk(func(path string, leaf *node) {
            if _, ok := head.lookupExact(path); !ok {
                head.insert(path, leaf.route)
            }
        })
    }
}
```

---

## 5. 中间件与 Group（P-6）

### 5.1 Group 处理

`r.Group(prefix, fn, middlewares...)` 与 v1 行为相同：
- 进入：`currentGroupPrefix += prefix`，`currentGroupHandlers += middlewares`
- 注册期 `r.GET(...)` 时把 group prefix/middlewares 合并进 route
- 退出：恢复

不变。

### 5.2 全局中间件 `r.Use(...)`

```go
func (r *Router) Use(handlers ...HandlerFunc) {
    if r.frozen.Load() {
        panic("rux: cannot Use after frozen")
    }
    r.globalChain = append(r.globalChain, handlers...)
}
```

**关键点**：`globalChain` 在 freeze 时一次性合并到每个 `route.finalChain`，请求期不再 append。

### 5.3 链合并示例

```go
// 注册：
r.Use(logger, recovery)                       // globalChain
r.Group("/api", func() {
    r.GET("/users", listUsers, authMW)        // route.chain = [authMW, listUsers]
}, apiKeyMW)                                  // group middleware

// Route 注册完成时（注册期）：
//   route.chain = [apiKeyMW, authMW, listUsers]
// Freeze 时：
//   route.finalChain = [logger, recovery, apiKeyMW, authMW, listUsers]
// 请求期：
//   ctx.handlers = route.finalChain  // 直接引用，0 alloc
```

---

## 6. API 与行为

### 6.1 公开 API 一览（破坏性变更已标 ⚠️）

```go
// Router lifecycle
func New(opts ...Option) *Router
func (r *Router) Freeze()                         // 新增
func (r *Router) Frozen() bool                    // 新增

// Route registration (registration phase only; panic after freeze)
func (r *Router) GET(path string, h HandlerFunc, mw ...HandlerFunc) *Route
func (r *Router) POST(...) *Route
// ... (PUT, PATCH, DELETE, HEAD, OPTIONS, CONNECT, TRACE)
func (r *Router) Any(path string, h HandlerFunc, mw ...HandlerFunc) *Route
func (r *Router) Add(path string, h HandlerFunc, methods ...string) *Route
func (r *Router) AddNamed(name, path string, h HandlerFunc, methods ...string) *Route
func (r *Router) Group(prefix string, fn func(), mw ...HandlerFunc)
func (r *Router) Resource(basePath string, controller any, mw ...HandlerFunc)
func (r *Router) Controller(basePath string, c ControllerFace, mw ...HandlerFunc)
func (r *Router) Use(mw ...HandlerFunc)
func (r *Router) NotFound(h ...HandlerFunc)
func (r *Router) NotAllowed(h ...HandlerFunc)

// Static file helpers (unchanged)
func (r *Router) StaticFile(path, file string)
func (r *Router) StaticDir(prefix, dir string)
func (r *Router) StaticFS(prefix string, fs http.FileSystem)
func (r *Router) StaticFiles(prefix, root, exts string)

// Lookup (offline / debug usage; not the hot path used by ServeHTTP)
// Returns a heap-allocated Params slice snapshot to keep the API ergonomic;
// the hot path (ServeHTTP) does NOT call Match — it uses an internal
// signature that writes into Context.params directly with zero allocation.
func (r *Router) Match(method, path string) (*Route, []Param, bool) // ⚠️ 返回值变化（P-12）

// Serving
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request)
func (r *Router) Listen(addr ...string)
func (r *Router) ListenTLS(addr, cert, key string)
func (r *Router) ListenUnix(file string)
func (r *Router) WrapHTTPHandlers(h ...func(http.Handler) http.Handler) http.Handler

// URL building
func (r *Router) BuildURL(name string, args ...any) *url.URL
func (r *Router) GetRoute(name string) *Route

// Context API
func (c *Context) Param(name string) string         // 不变
func (c *Context) Params() *Params                  // ⚠️ 新增；返回内联 params 指针
func (c *Context) Route() *Route                    // 新增
func (c *Context) MatchedPath() string              // 新增
func (c *Context) Set(key string, val any)          // 不变
func (c *Context) Get(key string) (any, bool)       // 不变
func (c *Context) JSON(code int, obj any) error     // 不变
func (c *Context) Text(code int, body string) error // 不变
// ...
```

### 6.2 移除 / 替代的 v1 API

| v1 API | v2 处理 | 替代方案 |
|---|---|---|
| `MatchResult` 结构 + `ReleaseMatchResult` | ⚠️ 移除（P-12） | `Match` 返回 `(*Route, []Param, bool)` |
| `QuickMatch` | ⚠️ 移除 | 等价于 `Match` |
| `EnableCaching`, `MaxNumCaches` | 已移除 | 不需要（Radix Tree 已够快） |
| Regex 参数 `{id:\d+}` | ⚠️ 移除（P-14） | 注册中间件验证：见下文 §6.2.1 |
| `Route.handler` + `Route.handlers` 二分 | ⚠️ 合并 | 统一 `Route.chain` |
| `c.Params` 字段（map 类型） | ⚠️ 改为方法 `c.Params() *Params` | `c.Param(name)` 直接获取 |

#### 6.2.1 Regex 参数替代

```go
// v1 写法（v2 不再支持）
r.GET("/users/{id:\\d+}", showUser)

// v2 写法 1：注册时附验证中间件
r.GET("/users/{id}", showUser, validate.IntParam("id"))

// v2 写法 2：handler 内自检
r.GET("/users/{id}", func(c *rux.Context) {
    id := c.Params().Int("id")
    if id <= 0 { c.Text(400, "bad id"); return }
    showUser(c, id)
})
```

提供 `pkg/validate` helper 包装常见模式（int/uuid/slug/...）。

### 6.3 错误处理策略

- **注册期**：所有违规（路径不合法、可选段位置错、参数过多、frozen 后注册）→ `panic`，因为这是程序员错误，不是运行时错误。
- **运行期**：未知 method / 路径未命中 → 走 noRoute / noAllowed 链，不 panic。
- **handler 内 panic**：由 `OnPanic` 捕获（与 v1 一致）。

---

## 7. 性能优化清单（按 P-x 回引）

| ID | 问题 | 修复手段 | 实现位置 |
|---|---|---|---|
| P-1 | 双重 method 分发 | `[9]*radixTree` 数组 + 节点单 chain | §3.1, §3.2, §3.3 |
| P-2 | 通配符优先级 bug | 查找顺序：静态→参数→通配符 | §4.4 |
| P-3 | paramsPool 未生效 | 内联 Params 到 Context，无 pool 需求 | §3.4 |
| P-4 | Params 用 map | 改 `[16]Param + count` 数组 | §3.4 |
| P-5 | 静态路由 key 拼接 | `[9]map[string]*Route` 方法分桶 | §3.3, §4.6 |
| P-6 | 中间件链每请求 append | Freeze 时合并到 `route.finalChain` | §4.5, §5 |
| P-7 | 节点 children map | `indices []byte + children []*node` | §3.2 |
| P-8 | RWMutex 长期开销 | atomic.Bool freeze 后无锁 | §3.3, §4.5 |
| P-9 | HEAD→GET 二次匹配 | Freeze 时镜像 GET 树到 HEAD | §4.7 |
| P-10 | 路径规范化 ≥3 次 | 注册期一次；请求期最简 trim | §4.2 |
| P-11 | handlers/routes 双 map | 节点单 `chain + route` 字段 | §3.2 |
| P-12 | MatchResult 半成品 | Match 直接返回 3 值 | §6.1 |
| P-13 | fastrux 与主包重复 | 删除 fastrux 子包 | §2.1 |
| P-14 | regex 参数静默丢失 | 移除 + 中间件替代方案 | §6.2.1 |
| P-15 | Context Set/Get 用 map | 类型化字段 + lazy map | §3.5 |
| P-16 | 节点子节点无优先级 | priority + 排序 indices | §3.2, §4.3 |

### 7.1 预期收益（对比 fastrux 现状）

| 场景 | fastrux 现状 | rux v2 目标 | 主要来源 |
|---|---|---|---|
| 静态路由 ServeHTTP | 114 ns / 1 alloc | **40-60 ns / 0 alloc** | P-5（无 string concat）+ P-6（无 chain append）+ P-15（无 map init） |
| 动态单参数 ServeHTTP | 304 ns / 3 alloc | **80-120 ns / 0 alloc** | P-4（params 内联）+ P-7（数组索引）+ 上述 |
| 5 参数 ServeHTTP | 377 ns / 3 alloc | **150-200 ns / 0 alloc** | 同上 |
| 404 ServeHTTP | 108 ns / 0 alloc | **40-60 ns / 0 alloc** | P-5 + P-15 |

---

## 8. 测试策略

### 8.1 单元测试覆盖

按层组织，每层独立可测：

| 层 | 测试文件 | 关键场景 |
|---|---|---|
| `methodIndex` | `rux_test.go` | 9 个有效方法、未知方法、空字符串、性能基准 |
| `Params` | `params_test.go`（新建） | append/Get/Has/Int/Reset、容量边界、超出 panic |
| `node`（树结构） | `tree_test.go` | 静态/参数/通配符/分裂/优先级排序/边界（根、空、单字符） |
| `radixTree`（注册+查找） | `tree_test.go` | 优先级（静态>参数>通配符）、回溯、可选段展开、HEAD 镜像 |
| `Router`（端到端） | `router_test.go` | Group、Resource、Controller、Use、Freeze 后 panic、404/405 |
| `dispatch` | `dispatch_test.go` | OnPanic、OnError、interceptAll、useEncodedPath |
| `Context` | `context_test.go` | Param、Set/Get、Next/Abort、错误链 |
| `binding/render` | 已有 | 不变 |

### 8.2 兼容性验证

- 端到端 fixture：`testdata/v1-routes.json` 列举 v1 时代典型路由形态，确保 v2 注册不 panic、查找语义符合预期。
- Regex 参数路由：明确写入 `xfail` 列表，注册时 panic，提示用户迁移。

### 8.3 基准测试

`benchmark_test.go` 重写，覆盖：

```go
BenchmarkV2_StaticRoute
BenchmarkV2_StaticRoute_404
BenchmarkV2_Param1
BenchmarkV2_Param5
BenchmarkV2_Wildcard
BenchmarkV2_GithubAPI       // 模拟真实 200+ 路由
BenchmarkV2_Parallel_Static // 并发场景，验证无锁路径
BenchmarkV2_Parallel_Param
```

每个 benchmark 同时跑 v1 / fastrux / v2 三方对比，输出到 `_benchmarks/results-v2.md`。

### 8.4 竞态检测

`go test -race -count=10 ./...` 必须通过。重点：
- 多 goroutine 并发 `ServeHTTP`（应零数据竞争 —— freeze 后无写）
- 注册期单 goroutine（Router 不保证注册期并发安全）

---

## 9. 实施阶段

### Phase 0：准备（0.5 天）
- [ ] 创建 `v2-rewrite` 工作分支（基于当前 `fea_v2`）
- [ ] 把 v1 `master` 用 tag 锁住（`v1.x-final`）
- [ ] 备份 `fastrux/` 到 `_archive/fastrux-snapshot.tar.gz`

### Phase 1：核心数据结构（1.5 天）
- [ ] `rux.go`: methodIndex / 常量 / Option 类型
- [ ] `tree.go`: node 结构、insert、splitNode、bumpPriority
- [ ] `params.go`（context 内嵌即可，单独文件可选）
- [ ] 单测：tree 结构正确性

### Phase 2：路由查找（1 天）
- [ ] `tree.go`: lookup（静态/参数/通配符/回溯）
- [ ] 静态/动态优先级测试
- [ ] 单测：100% 路径覆盖

### Phase 3：Router（1.5 天）
- [ ] `router.go`: Router struct、staticRoutes、dynamicTrees
- [ ] `Add` / GET/POST/...各 verb / Any
- [ ] Group / Resource / Controller / Use
- [ ] 可选参数展开（复用 `internal/util/util.go`）

### Phase 4：Freeze 与 Dispatch（1 天）
- [ ] `Freeze()`: 链合并 + HEAD 镜像
- [ ] `ServeHTTP`: 无锁热路径
- [ ] 405 / 404 / fallback 路径
- [ ] OnPanic / OnError

### Phase 5：Context 与渲染（1 天）
- [ ] `context.go`: 内联 Params + 类型化字段
- [ ] 把 v1 `context_render.go` / `context_binding.go` 适配过来
- [ ] `response_writer.go` 适配

### Phase 6：旧代码清理（0.5 天）
- [ ] 删除 `fastrux/` 子包
- [ ] 删除 `_tmp/` 中过时文档
- [ ] `docs/design/design-refactor-core.md` / `design-implementation.md` 标记 ARCHIVED

### Phase 7：测试与基准（1.5 天）
- [ ] 跑通所有迁移过来的 v1 测试
- [ ] 新增 v2 单测达到 ≥ 90% 覆盖率
- [ ] 跑 benchmark，对比 v1 / fastrux / v2 / httprouter / gin
- [ ] `go test -race` 通过

### Phase 8：文档（1 天）
- [ ] `README.md` / `README.zh-CN.md` 重写示例
- [ ] `MIGRATION-v1-to-v2.md` 迁移指南
- [ ] `CHANGELOG.md` 标记 v2.0.0

**总估时**：约 9 天单人工作量。

---

## 10. 风险与待确认

### 10.1 已识别风险

| 风险 | 影响 | 缓解 |
|---|---|---|
| `MaxParams = 16` 上限被某些用户超出 | 注册 panic | 按现实 API 调研：GitHub/AWS/Twilio 最多 6-7；16 已 4x 富余。如有反例可调到 32（成本：每 Context 多 512B） |
| Freeze 模式打破"运行期热加载"用户场景 | 这类用户必须迁回 v1 | 文档明确说明；提供 `r.Restart(newRouter)` 模式作为软方案（atomic.Pointer 切换） |
| HEAD 镜像导致 GET 路由 panic 时双重影响 | 同一 panic 影响两个方法 | OnPanic handler 拿到 `c.Req.Method` 仍正确，无逻辑差异 |
| 失去 regex 参数后用户被迫加中间件 | 代码量增加 | 提供 `pkg/validate` 包封装常见 pattern |
| 移除 fastrux 子包破坏已使用它的下游 | 编译错误 | fastrux 当前未发布稳定 tag，影响面可控 |

### 10.2 待确认事项

| 编号 | 待确认 | 默认决策（如不反对则按此实施） |
|---|---|---|
| Q1 | `MaxParams` 取值 | **16** |
| Q2 | 是否提供 `r.MutableMode()` 软逃生口 | **不提供**（freeze 即不可逆，简单清晰） |
| Q3 | `c.Params()` 返回 `*Params` 还是 `Params`（值拷贝） | **`*Params`**（避免 16-Param 数组拷贝开销） |
| Q4 | 是否保留 `pkg/binding`/`pkg/render` 的 v1 风格 API | **保留**，仅适配新 Context |
| Q5 | 是否提供 v1 -> v2 自动迁移工具（`gofmt -r` 风格） | **暂不提供**，文档迁移指南足够 |
| Q6 | 全局中间件能否在路由注册之间插入（顺序敏感） | **不能**：v2 要求 `Use` 全部在路由注册前；之后 `Use` panic（更安全的强约束） |
| Q7 | `OnError`/`OnPanic` 是否仍为单 handler | **保持单 handler**（与 v1 一致） |

---

## 11. 附录

### 11.1 关键文件 LOC 预算

| 文件 | 预期 LOC |
|---|---|
| `rux.go` | ~120 |
| `router.go` | ~400 |
| `route.go` | ~180 |
| `tree.go` | ~500 |
| `context.go` | ~280 |
| `context_render.go` | ~250（迁移） |
| `context_binding.go` | ~120（迁移） |
| `dispatch.go` | ~180 |
| `middleware.go` | ~100 |
| `extends.go` | ~150 |
| `response_writer.go` | ~80 |
| `utils.go` | ~150 |

### 11.2 与 httprouter 设计的差异

| 维度 | httprouter | rux v2 | 理由 |
|---|---|---|---|
| 中间件 | 不支持 | 一等公民 | rux 定位是框架而非纯路由 |
| Group | 不支持 | 支持 | 同上 |
| 命名路由 / URL build | 不支持 | 支持 | 模板渲染常用 |
| 节点结构 | 类似 | 几乎相同 | httprouter 已经是黄金标准 |
| Params | `[]Param` slice | `[16]Param` 内联 | 内联换更少 alloc |
| Freeze 模式 | 隐式（不支持运行时改） | 显式 | 明确语义 |

### 11.3 名词表

| 术语 | 含义 |
|---|---|
| **Frozen Router** | 调用过 `Freeze()` 或首次 `ServeHTTP` 之后的状态，不再接受路由注册 |
| **finalChain** | Freeze 时合并好的最终 handler 链（global + group + route + main handler） |
| **method index** | HTTP 方法名到 0-8 整数的映射，用作 `[9]` 数组下标 |
| **HEAD 镜像** | Freeze 时把所有 GET 路由复制注册到 HEAD 树（除非用户已显式注册 HEAD） |
| **可选段** | 路径中 `[/...]` 包裹的段，注册期展开为有/无两条独立路由 |

---

## 12. 评审清单

请在评审本文档时确认：

- [ ] §1.2 的 P-1 ~ P-16 问题清单覆盖完整（如有遗漏请补充）
- [ ] §3 的数据结构能否落地（无 Go 语法/语义违和）
- [ ] §4 的算法描述清晰（含边界场景）
- [ ] §6 的破坏性变更列表完整
- [ ] §7 的预期收益数字可接受
- [ ] §9 的阶段划分粒度合理（任意阶段交付物可单独 review）
- [ ] §10.2 的 Q1-Q7 默认决策符合预期

评审通过后，进入 `writing-plans` 阶段，按 §9 把每个 Phase 拆成可执行的 task list。
