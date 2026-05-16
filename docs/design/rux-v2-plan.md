# Rux v2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the `gookit/rux` main package as a high-performance Radix Tree router (v2.0), replacing the v1 map+regex implementation and consolidating the `fastrux/` prototype back into the main package.

**Architecture:** Clean-room rewrite on a new `v2-rewrite` branch. Per-method radix tree (`[9]*radixTree` array indexed by HTTP method int code) + per-method static map (`[9]map[string]*Route`). Path params inlined into Context (`[16]Param + count`) for zero allocation. Default freeze mode after first `ServeHTTP` enables lock-free hot path. Middleware chain pre-merged at freeze time.

**Tech Stack:** Go 1.23+, no third-party deps in core (only `gookit/color` for debug print and `gookit/goutil` for utilities). Tests use `github.com/gookit/goutil/testutil/assert` and `goutil/testutil`. Date: 2026-05-16.

**Spec:** See `docs/design/rux-v2-design.md`. P-x references throughout link to that document's problem catalogue.

---

## File Structure

| File | Status | Responsibility | Approx LOC |
|---|---|---|---|
| `rux.go` | rewrite | Package constants, `methodIndex`, `Option` types, `Debug()` | ~120 |
| `tree.go` | new (replaces `radix_tree.go`) | Radix Tree node + insert/lookup/split/walk algorithms | ~500 |
| `router.go` | rewrite | Router struct, route registration, Group, Resource, Controller, Use, Freeze | ~400 |
| `route.go` | rewrite | Route definition, URL building, chain management | ~180 |
| `context.go` | rewrite | Context with inline `Params [16]Param` + typed fields | ~280 |
| `params.go` | new | `Param` struct, `Params` inline container, helpers | ~120 |
| `dispatch.go` | rewrite | `ServeHTTP`, `Listen*`, lock-free hot path | ~180 |
| `context_render.go` | adapt (light) | JSON/XML/Text/HTML/File responses | ~250 |
| `context_binding.go` | adapt (light) | Form/JSON/Header/Query parameter binding | ~120 |
| `middleware.go` | adapt (light) | Built-in middleware (Recovery etc.) | ~100 |
| `extends.go` | adapt (light) | `BuildRequestURL` URL builder | ~150 |
| `response_writer.go` | rename + light edit | Wraps `http.ResponseWriter`, tracks status/size (was `response_wirter.go` typo) | ~80 |
| `utils.go` | rewrite | `normalizePath`, `isStaticPath`, `parseOptionalSegments`, debug print | ~150 |
| `internal/util/util.go` | keep | `Panicf`, `ValidateOptionalSegments` | unchanged |
| `radix_tree.go` | **delete** | Replaced by `tree.go` | -- |
| `radix_tree_test.go` | **delete** | Replaced by `tree_test.go` | -- |
| `route_parse_match.go` | already deleted | -- | -- |
| `fastrux/` | **delete entire dir** | Functionality merged into main package (P-13) | -- |
| `_tmp/` | **archive + delete** | Move dev-progress to `docs/design/archive/` | -- |
| `tree_test.go` | new | Radix Tree algorithm tests | ~600 |
| `params_test.go` | new | Params container tests | ~80 |
| `router_test.go` | rewrite | End-to-end Router tests | ~400 |
| `context_test.go` | adapt | Context tests | ~250 |
| `dispatch_test.go` | adapt | Dispatch tests | ~200 |
| `benchmark_test.go` | rewrite | Comparative benchmarks | ~250 |

---

## Phase 0: Setup (0.5 day)

### Task 0.1: Create v2 working branch

**Files:** none (git operations only)

- [ ] **Step 1: Verify clean working tree on fea_v2**

```bash
git status
```

Expected output should show the current modified files; if user has unstaged work they want to keep, prompt them to commit first.

- [ ] **Step 2: Tag v1 final state on master**

```bash
git tag -a v1.x-final master -m "v1.x final state before v2 rewrite"
```

Expected: tag created silently (no output).

- [ ] **Step 3: Create v2-rewrite branch from current fea_v2**

```bash
git checkout -b v2-rewrite
git status
```

Expected output: `On branch v2-rewrite`.

- [ ] **Step 4: Snapshot fastrux to archive (out of git)**

```bash
mkdir -p _archive
tar -czf _archive/fastrux-snapshot-$(date +%Y%m%d).tar.gz fastrux/
ls -lh _archive/
```

Expected: a `.tar.gz` file ~50-200KB.

- [ ] **Step 5: Add _archive to .gitignore**

```bash
grep -qxF '_archive/' .gitignore || echo '_archive/' >> .gitignore
git diff .gitignore
```

Expected diff shows `+_archive/`.

- [ ] **Step 6: Commit branch setup**

```bash
git add .gitignore
git commit -m "chore(v2): create v2-rewrite branch, archive fastrux snapshot"
```

---

## Phase 1: Core Data Structures (1.5 days)

### Task 1.1: methodIndex (P-1)

**Files:**
- Create: `rux.go` (replacing existing — keep package doc comment)
- Test: `rux_test.go` (extend existing or create)

- [ ] **Step 1: Write the failing test**

Add to `rux_test.go`:

```go
package rux

import (
    "testing"

    "github.com/gookit/goutil/testutil/assert"
)

func TestMethodIndex(t *testing.T) {
    cases := []struct {
        method string
        want   int
    }{
        {"GET", 0},
        {"HEAD", 1},
        {"POST", 2},
        {"PUT", 3},
        {"PATCH", 4},
        {"DELETE", 5},
        {"OPTIONS", 6},
        {"CONNECT", 7},
        {"TRACE", 8},
        {"", -1},
        {"FOO", -1},
        {"get", -1}, // case-sensitive: HTTP methods are uppercase
    }
    for _, c := range cases {
        assert.Eq(t, c.want, methodIndex(c.method), "method=%s", c.method)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestMethodIndex .
```

Expected: build failure (`undefined: methodIndex`).

- [ ] **Step 3: Rewrite rux.go**

Open `rux.go` and replace its contents with:

```go
// Package rux is a high-performance HTTP router for Go.
//
// Source: https://github.com/gookit/rux
package rux

import (
    "strings"

    "github.com/gookit/color"
)

// HTTP method names.
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
)

// methodCount is the total number of HTTP methods rux understands.
const methodCount = 9

// Common content type constants.
const (
    ContentType        = "Content-Type"
    ContentBinary      = "application/octet-stream"
    ContentDisposition = "Content-Disposition"

    dispositionInline     = "inline"
    dispositionAttachment = "attachment"
)

// Context key constants for backward-compatible c.Get/Set users.
// New code should prefer typed accessors on Context.
const (
    CTXRecoverResult    = "_recoverResult"
    CTXAllowedMethods   = "_allowedMethods"
    CTXCurrentRouteName = "_currentRouteName"
    CTXCurrentRoutePath = "_currentRoutePath"
)

// methodIndex maps an HTTP method string to a 0..8 array index.
// Returns -1 for unknown methods.
func methodIndex(m string) int {
    if len(m) == 0 {
        return -1
    }
    switch m[0] {
    case 'G':
        if m == GET {
            return 0
        }
    case 'H':
        if m == HEAD {
            return 1
        }
    case 'P':
        if m == POST {
            return 2
        }
        if m == PUT {
            return 3
        }
        if m == PATCH {
            return 4
        }
    case 'D':
        if m == DELETE {
            return 5
        }
    case 'O':
        if m == OPTIONS {
            return 6
        }
    case 'C':
        if m == CONNECT {
            return 7
        }
    case 'T':
        if m == TRACE {
            return 8
        }
    }
    return -1
}

// allMethods returns the full list of supported HTTP methods.
var allMethods = []string{GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS, CONNECT, TRACE}

// AnyMethods returns the list of HTTP methods registered by Any().
func AnyMethods() []string {
    return []string{GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD, CONNECT, TRACE}
}

// AllMethods returns all HTTP methods rux supports.
func AllMethods() []string { return allMethods }

// MethodsString returns all methods joined by comma.
func MethodsString() string { return strings.Join(allMethods, ",") }

// ControllerFace is implemented by structs registered via Router.Controller.
type ControllerFace interface {
    AddRoutes(g *Router)
}

// RESTful action -> default method list.
var (
    IndexAction  = "Index"
    CreateAction = "Create"
    StoreAction  = "Store"
    ShowAction   = "Show"
    EditAction   = "Edit"
    UpdateAction = "Update"
    DeleteAction = "Delete"

    RESTFulActions = map[string][]string{
        IndexAction:  {GET},
        CreateAction: {GET},
        StoreAction:  {POST},
        ShowAction:   {GET},
        EditAction:   {GET},
        UpdateAction: {PUT, PATCH},
        DeleteAction: {DELETE},
    }
)

var debug bool

// Debug toggles router debug logging.
func Debug(val bool) {
    debug = val
    if debug {
        color.Info.Println("rux: DEBUG mode enabled")
    }
}

// IsDebug reports whether debug logging is on.
func IsDebug() bool { return debug }
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestMethodIndex .
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add rux.go rux_test.go
git commit -m "feat(v2): rux.go skeleton with methodIndex (P-1)"
```

---

### Task 1.2: Params inline container (P-3, P-4)

**Files:**
- Create: `params.go`
- Test: `params_test.go`

- [ ] **Step 1: Write the failing test**

Create `params_test.go`:

```go
package rux

import (
    "testing"

    "github.com/gookit/goutil/testutil/assert"
)

func TestParams_Empty(t *testing.T) {
    var p Params
    assert.Eq(t, 0, p.Len())
    assert.Eq(t, "", p.Get("missing"))
    assert.False(t, p.Has("missing"))
    assert.Eq(t, 0, p.Int("missing"))
}

func TestParams_AppendAndGet(t *testing.T) {
    var p Params
    p.append("id", "42")
    p.append("slug", "hello")
    assert.Eq(t, 2, p.Len())
    assert.Eq(t, "42", p.Get("id"))
    assert.Eq(t, "hello", p.Get("slug"))
    assert.True(t, p.Has("id"))
    assert.False(t, p.Has("missing"))
    assert.Eq(t, 42, p.Int("id"))
    assert.Eq(t, 0, p.Int("slug"))   // not int-parseable
    assert.Eq(t, 0, p.Int("missing"))
}

func TestParams_Reset(t *testing.T) {
    var p Params
    p.append("a", "1")
    p.append("b", "2")
    p.Reset()
    assert.Eq(t, 0, p.Len())
    assert.Eq(t, "", p.Get("a"))
}

func TestParams_Snapshot(t *testing.T) {
    var p Params
    p.append("k", "v")
    snap := p.Snapshot()
    assert.Eq(t, 1, len(snap))
    assert.Eq(t, "k", snap[0].Key)
    assert.Eq(t, "v", snap[0].Value)

    // mutating snapshot must not affect original
    snap[0].Value = "modified"
    assert.Eq(t, "v", p.Get("k"))
}

func TestParams_OverflowPanics(t *testing.T) {
    var p Params
    for i := 0; i < MaxParams; i++ {
        p.append("k", "v")
    }
    assert.Panics(t, func() { p.append("overflow", "x") })
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestParams .
```

Expected: build failure (`undefined: Params`).

- [ ] **Step 3: Implement params.go**

Create `params.go`:

```go
package rux

import "strconv"

// MaxParams is the maximum number of path parameters per route.
// Inlined storage in Context avoids any heap allocation for params.
// Registering a route that exceeds this limit panics.
const MaxParams = 16

// Param is a single path parameter.
type Param struct {
    Key   string
    Value string
}

// Params is a fixed-capacity inline parameter container.
// It lives directly inside Context, providing zero-allocation parameter
// passing when the Context itself is reused via sync.Pool.
type Params struct {
    data [MaxParams]Param
    n    uint8
}

// Len returns the number of parameters.
func (p *Params) Len() int { return int(p.n) }

// Get returns the value for name, or "" if not found.
func (p *Params) Get(name string) string {
    for i := uint8(0); i < p.n; i++ {
        if p.data[i].Key == name {
            return p.data[i].Value
        }
    }
    return ""
}

// Has reports whether a parameter with the given name exists.
func (p *Params) Has(name string) bool {
    for i := uint8(0); i < p.n; i++ {
        if p.data[i].Key == name {
            return true
        }
    }
    return false
}

// Int parses the named parameter as int, returning 0 on miss or parse error.
func (p *Params) Int(name string) int {
    if v := p.Get(name); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            return n
        }
    }
    return 0
}

// Snapshot returns a heap-allocated copy of the params slice.
// Use this when you need to retain params beyond the handler scope
// (e.g., goroutines, async logging).
func (p *Params) Snapshot() []Param {
    out := make([]Param, p.n)
    for i := uint8(0); i < p.n; i++ {
        out[i] = p.data[i]
    }
    return out
}

// Reset clears the params (called by Context.Reset on pool return).
func (p *Params) Reset() { p.n = 0 }

// append adds a parameter. Panics if MaxParams is exceeded.
// Callers in tree.lookup should ensure tree.maxParams <= MaxParams at registration.
func (p *Params) append(key, value string) {
    if p.n >= MaxParams {
        panic("rux: params overflow (MaxParams=" + strconv.Itoa(MaxParams) + ")")
    }
    p.data[p.n].Key = key
    p.data[p.n].Value = value
    p.n++
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestParams .
```

Expected: `PASS` for all 5 sub-tests.

- [ ] **Step 5: Commit**

```bash
git add params.go params_test.go
git commit -m "feat(v2): inline Params container with [16]Param (P-3, P-4)"
```

---

### Task 1.3: Tree node skeleton (P-2, P-7, P-11, P-16)

**Files:**
- Create: `tree.go`
- Test: `tree_test.go`

- [ ] **Step 1: Write the failing test**

Create `tree_test.go`:

```go
package rux

import (
    "testing"

    "github.com/gookit/goutil/testutil/assert"
)

func TestNewRadixTree(t *testing.T) {
    tree := newRadixTree()
    assert.NotNil(t, tree)
    assert.NotNil(t, tree.root)
    assert.Eq(t, nodeRoot, tree.root.nType)
    assert.Eq(t, "/", tree.root.prefix)
    assert.Nil(t, tree.root.chain)
    assert.Eq(t, uint8(0), tree.maxParams)
}

func TestNode_AddStaticChild(t *testing.T) {
    parent := &node{
        prefix: "/",
        nType:  nodeRoot,
    }
    parent.addStaticChild(&node{prefix: "users", nType: nodeStatic})
    assert.Eq(t, 1, len(parent.children))
    assert.Eq(t, byte('u'), parent.indices[0])
    assert.Eq(t, "users", parent.children[0].prefix)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestNewRadixTree|TestNode_AddStaticChild" .
```

Expected: build failure (`undefined: newRadixTree, node, ...`).

- [ ] **Step 3: Implement tree.go skeleton**

Create `tree.go`:

```go
package rux

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
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestNewRadixTree|TestNode_AddStaticChild" .
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add tree.go tree_test.go
git commit -m "feat(v2): radix tree node skeleton with indices array (P-7, P-11)"
```

---

### Task 1.4: HandlersChain type and Route skeleton

**Files:**
- Create: `route.go` (replacing existing)

- [ ] **Step 1: Write the failing test**

Add to `tree_test.go`:

```go
func TestHandlersChain_LengthSemantics(t *testing.T) {
    h1 := func(c *Context) {}
    h2 := func(c *Context) {}
    chain := HandlersChain{h1, h2}
    assert.Eq(t, 2, len(chain))
}

func TestNewRoute_Defaults(t *testing.T) {
    h := func(c *Context) {}
    r := newRoute("/users", h, []string{GET})
    assert.Eq(t, "/users", r.path)
    assert.Eq(t, []string{GET}, r.methods)
    assert.Eq(t, 1, len(r.chain))   // main handler appended
}

func TestNewRoute_PanicsOnEmptyHandler(t *testing.T) {
    assert.Panics(t, func() {
        newRoute("/users", nil, []string{GET})
    })
}

func TestNewRoute_DefaultMethodGET(t *testing.T) {
    h := func(c *Context) {}
    r := newRoute("/users", h, nil)
    assert.Eq(t, []string{GET}, r.methods)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestHandlersChain|TestNewRoute" .
```

Expected: build failure.

- [ ] **Step 3: Rewrite route.go**

Replace `route.go` contents with:

```go
package rux

import (
    "fmt"
    "strings"

    "github.com/gookit/goutil"
)

// HandlerFunc is the standard handler signature.
type HandlerFunc func(c *Context)

// HandlersChain is a list of handlers (middlewares + final handler).
// The final handler is the last element — there is no separate field for it.
type HandlersChain []HandlerFunc

// abortIndex marks an aborted handler chain (set by Context.Abort).
const abortIndex int8 = 63

// RouteInfo is a snapshot of a Route's public information.
type RouteInfo struct {
    Name        string
    Path        string
    HandlerName string
    Methods     []string
    HandlerNum  int
}

// Route describes a single registered route.
//
// Fields populated at registration time: name, path, methods, chain.
// finalChain is populated by Router.Freeze(), at which point chain is
// no longer used at request time.
type Route struct {
    name    string
    path    string
    methods []string

    // User-supplied middlewares followed by the main handler.
    // After Freeze(), finalChain = router.globalChain ++ chain.
    chain      HandlersChain
    finalChain HandlersChain

    Opts map[string]any
}

// newRoute creates a Route with the main handler appended to chain.
// Panics if handler is nil. Defaults methods to []string{GET} when empty.
func newRoute(path string, handler HandlerFunc, methods []string) *Route {
    if handler == nil {
        panic(fmt.Sprintf("rux: route handler cannot be nil (path: %q)", path))
    }
    if len(methods) == 0 {
        methods = []string{GET}
    } else {
        methods = formatMethods(methods)
    }
    validateMethods(methods)
    return &Route{
        path:    simpleFmtPath(path),
        methods: methods,
        chain:   HandlersChain{handler},
    }
}

// newNamedRoute is like newRoute but assigns a name.
func newNamedRoute(name, path string, handler HandlerFunc, methods []string) *Route {
    r := newRoute(path, handler, methods)
    r.name = strings.TrimSpace(name)
    return r
}

// Use prepends middleware handlers to this route (run before the main handler).
func (r *Route) Use(middlewares ...HandlerFunc) *Route {
    if len(middlewares) == 0 {
        return r
    }
    if len(r.chain)+len(middlewares) >= int(abortIndex) {
        goutil.Panicf("rux: too many handlers (limit %d)", abortIndex)
    }
    // Insert middlewares before the main handler (last element).
    main := r.chain[len(r.chain)-1]
    r.chain = append(r.chain[:len(r.chain)-1], middlewares...)
    r.chain = append(r.chain, main)
    return r
}

// Name returns the route's name.
func (r *Route) Name() string { return r.name }

// Path returns the route's path.
func (r *Route) Path() string { return r.path }

// Methods returns the route's allowed methods.
func (r *Route) Methods() []string { return r.methods }

// MethodString joins allowed methods with sep.
func (r *Route) MethodString(sep string) string { return strings.Join(r.methods, sep) }

// Handler returns the main handler (last element of the chain).
func (r *Route) Handler() HandlerFunc {
    if len(r.chain) == 0 {
        return nil
    }
    return r.chain[len(r.chain)-1]
}

// Handlers returns the user-supplied middlewares (excluding main handler).
func (r *Route) Handlers() HandlersChain {
    if len(r.chain) <= 1 {
        return nil
    }
    return r.chain[:len(r.chain)-1]
}

// HandlerName returns the symbolic name of the main handler.
func (r *Route) HandlerName() string {
    return goutil.FuncName(r.Handler())
}

// String returns a debug representation of the route.
func (r *Route) String() string {
    return fmt.Sprintf("%-15s %-38s --> %s (%d middleware)",
        r.MethodString(","), r.path, r.HandlerName(), len(r.chain)-1)
}

// Info returns a RouteInfo snapshot.
func (r *Route) Info() RouteInfo {
    return RouteInfo{r.name, r.path, r.HandlerName(), r.methods, len(r.chain) - 1}
}

// validateMethods panics if any method in m is not a recognized HTTP verb.
func validateMethods(m []string) {
    for _, method := range m {
        if methodIndex(method) < 0 {
            goutil.Panicf("rux: invalid HTTP method %q, must be one of: %s",
                method, MethodsString())
        }
    }
}

// formatMethods uppercases and trims each method string.
func formatMethods(methods []string) []string {
    out := make([]string, 0, len(methods))
    for _, m := range methods {
        m = strings.TrimSpace(m)
        if m != "" {
            out = append(out, strings.ToUpper(m))
        }
    }
    return out
}

// simpleFmtPath does the minimal path normalization needed at Route construction.
// Full normalization happens in Router.appendRoute.
func simpleFmtPath(path string) string {
    path = strings.TrimSpace(path)
    if path == "" {
        return "/"
    }
    if path[0] != '/' {
        path = "/" + path
    }
    return path
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestHandlersChain|TestNewRoute" .
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add route.go tree_test.go
git commit -m "feat(v2): Route with unified chain (no handler/handlers split)"
```

---

### Task 1.5: utils.go path helpers (P-10)

**Files:**
- Modify: `utils.go` (replacing existing)

- [ ] **Step 1: Write the failing test**

Add to `tree_test.go` (or create `utils_test.go`):

```go
func TestNormalizePath(t *testing.T) {
    cases := map[string]string{
        "":            "/",
        "/":           "/",
        "/users":      "/users",
        "users":       "/users",
        "/users/":     "/users",
        "//users":     "/users",
        "/users//":    "/users",
        "/a//b///c/":  "/a/b/c",
    }
    for in, want := range cases {
        assert.Eq(t, want, normalizePath(in), "input=%q", in)
    }
}

func TestIsStaticPath(t *testing.T) {
    cases := map[string]bool{
        "/users":         true,
        "/users/{id}":    false,
        "/files/*path":   false,
        "/path[/{x}]":    false,
        "/users/:id":     false,
        "/":              true,
    }
    for in, want := range cases {
        assert.Eq(t, want, isStaticPath(in), "input=%q", in)
    }
}

func TestLongestCommonPrefix(t *testing.T) {
    assert.Eq(t, 0, longestCommonPrefix("", "abc"))
    assert.Eq(t, 0, longestCommonPrefix("abc", ""))
    assert.Eq(t, 3, longestCommonPrefix("abc", "abcdef"))
    assert.Eq(t, 3, longestCommonPrefix("abcdef", "abc"))
    assert.Eq(t, 2, longestCommonPrefix("abxx", "abyy"))
    assert.Eq(t, 0, longestCommonPrefix("xyz", "abc"))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestNormalizePath|TestIsStaticPath|TestLongestCommonPrefix" .
```

Expected: build failure.

- [ ] **Step 3: Rewrite utils.go**

Replace `utils.go` with:

```go
package rux

import (
    "fmt"
    "os"
    "strings"

    "github.com/gookit/color"
)

// normalizePath enforces:
//   - leading '/'
//   - no trailing '/' (unless root)
//   - no doubled '//'
// Idempotent. Allocates only when changes are needed.
func normalizePath(path string) string {
    if path == "" {
        return "/"
    }
    // Fast path: already canonical.
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

// isStaticPath reports whether path contains no dynamic segments.
func isStaticPath(path string) bool {
    for i := 0; i < len(path); i++ {
        switch path[i] {
        case '{', '[', ':', '*':
            return false
        }
    }
    return true
}

// longestCommonPrefix returns the length of the longest common byte prefix.
func longestCommonPrefix(a, b string) int {
    n := len(a)
    if len(b) < n {
        n = len(b)
    }
    i := 0
    for i < n && a[i] == b[i] {
        i++
    }
    return i
}

// resolveAddress turns user-supplied addr arguments into a single "ip:port".
func resolveAddress(addr []string) string {
    ip := "0.0.0.0"
    switch len(addr) {
    case 0:
        if port := os.Getenv("PORT"); port != "" {
            return ip + ":" + port
        }
        return ip + ":8080"
    case 1:
        if strings.IndexByte(addr[0], ':') != -1 {
            ss := strings.SplitN(addr[0], ":", 2)
            if ss[0] != "" {
                return addr[0]
            }
            return ip + ":" + ss[1]
        }
        return ip + ":" + addr[0]
    case 2:
        return addr[0] + ":" + addr[1]
    default:
        panic("rux: too many addr arguments")
    }
}

func debugPrintRoute(route *Route) {
    debugPrint(route.String())
}

func debugPrintError(err error) {
    if err != nil {
        debugPrint("<red>[ERROR]</> %v", err)
    }
}

func debugPrint(f string, v ...any) {
    if debug {
        color.Printf("<cyan>[RUX-DEBUG]</> %s\n", fmt.Sprintf(f, v...))
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestNormalizePath|TestIsStaticPath|TestLongestCommonPrefix" .
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add utils.go tree_test.go
git commit -m "feat(v2): path helpers — normalize once at registration (P-10)"
```

---

## Phase 2: Tree Operations (1 day)

### Task 2.1: Tree insert — static paths

**Files:**
- Modify: `tree.go`
- Test: `tree_test.go`

- [ ] **Step 1: Write the failing test**

Add to `tree_test.go`:

```go
func TestTreeInsert_SingleStaticPath(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    route := newRoute("/users", h, []string{GET})
    tree.insert("/users", route)

    // Walk down: root("/") -> "users"
    assert.Eq(t, 1, len(tree.root.children))
    assert.Eq(t, byte('u'), tree.root.indices[0])
    leaf := tree.root.children[0]
    assert.Eq(t, "users", leaf.prefix)
    assert.Same(t, route, leaf.route)
}

func TestTreeInsert_TwoSiblings(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    tree.insert("/users", newRoute("/users", h, []string{GET}))
    tree.insert("/posts", newRoute("/posts", h, []string{GET}))

    assert.Eq(t, 2, len(tree.root.children))
}

func TestTreeInsert_NodeSplit(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    tree.insert("/userprofile", newRoute("/userprofile", h, []string{GET}))
    tree.insert("/userlist", newRoute("/userlist", h, []string{GET}))

    // After split: root -> "user" -> {"profile", "list"}
    assert.Eq(t, 1, len(tree.root.children))
    parent := tree.root.children[0]
    assert.Eq(t, "user", parent.prefix)
    assert.Eq(t, 2, len(parent.children))
}

func TestTreeInsert_DuplicatePathPanics(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    tree.insert("/users", newRoute("/users", h, []string{GET}))
    assert.Panics(t, func() {
        tree.insert("/users", newRoute("/users", h, []string{GET}))
    })
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestTreeInsert .
```

Expected: build failure (`undefined: tree.insert`).

- [ ] **Step 3: Implement tree.insert (static-only, with split)**

Append to `tree.go`:

```go
// insert adds a normalized path -> route mapping to the tree.
// path must already be normalized (leading '/', no trailing '/', no '//').
// Panics on duplicate path.
//
// Invariants this function depends on:
//   - param/wildcard child nodes have prefix == "" (their semantic prefix
//     ":id" / "*all" is stored in paramName for display only). The empty
//     prefix lets the matchLoop pass through them without trying to
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

// splitNode splits node n at byte index splitIdx into a parent (n.prefix[:splitIdx])
// with one child holding the remainder (n.prefix[splitIdx:]) along with all of
// n's previous children, paramChild, wildcardChild, route, and chain.
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
```

Add the missing imports at the top of `tree.go`:

```go
import (
    "fmt"
    "strings"
)
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestTreeInsert .
```

Expected: `PASS` for all 4 sub-tests.

- [ ] **Step 5: Commit**

```bash
git add tree.go tree_test.go
git commit -m "feat(v2): radix tree insert with node split (static + param + wildcard)"
```

---

### Task 2.2: Tree lookup — static, param, wildcard with correct priority (P-2)

**Files:**
- Modify: `tree.go`
- Test: `tree_test.go`

- [ ] **Step 1: Write the failing test**

Add to `tree_test.go`:

```go
func TestTreeLookup_Static(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    r := newRoute("/users", h, []string{GET})
    tree.insert("/users", r)

    var ps Params
    got, ok := tree.lookup("/users", &ps)
    assert.True(t, ok)
    assert.Same(t, r, got)
    assert.Eq(t, 0, ps.Len())
}

func TestTreeLookup_Param(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    r := newRoute("/users/:id", h, []string{GET})
    tree.insert("/users/:id", r)

    var ps Params
    got, ok := tree.lookup("/users/42", &ps)
    assert.True(t, ok)
    assert.Same(t, r, got)
    assert.Eq(t, "42", ps.Get("id"))
}

func TestTreeLookup_Wildcard(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    r := newRoute("/files/*path", h, []string{GET})
    tree.insert("/files/*path", r)

    var ps Params
    got, ok := tree.lookup("/files/a/b/c.txt", &ps)
    assert.True(t, ok)
    assert.Same(t, r, got)
    assert.Eq(t, "a/b/c.txt", ps.Get("path"))
}

// P-2: static must beat wildcard.
func TestTreeLookup_StaticBeatsWildcard(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    rWild := newRoute("/users/*all", h, []string{GET})
    rStatic := newRoute("/users/me", h, []string{GET})
    tree.insert("/users/*all", rWild)
    tree.insert("/users/me", rStatic)

    var ps Params
    got, ok := tree.lookup("/users/me", &ps)
    assert.True(t, ok)
    assert.Same(t, rStatic, got, "static must beat wildcard")
}

// P-2: static must beat param.
func TestTreeLookup_StaticBeatsParam(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    rParam := newRoute("/users/:id", h, []string{GET})
    rStatic := newRoute("/users/me", h, []string{GET})
    tree.insert("/users/:id", rParam)
    tree.insert("/users/me", rStatic)

    var ps Params
    got, ok := tree.lookup("/users/me", &ps)
    assert.True(t, ok)
    assert.Same(t, rStatic, got, "static must beat param")
}

// P-2: param must beat wildcard.
func TestTreeLookup_ParamBeatsWildcard(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    rWild := newRoute("/files/*all", h, []string{GET})
    rParam := newRoute("/files/:name", h, []string{GET})
    tree.insert("/files/*all", rWild)
    tree.insert("/files/:name", rParam)

    var ps Params
    got, ok := tree.lookup("/files/foo.txt", &ps)
    assert.True(t, ok)
    assert.Same(t, rParam, got, "single-segment param must beat wildcard")
    assert.Eq(t, "foo.txt", ps.Get("name"))
}

func TestTreeLookup_Miss(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    tree.insert("/users", newRoute("/users", h, []string{GET}))

    var ps Params
    _, ok := tree.lookup("/posts", &ps)
    assert.False(t, ok)

    _, ok = tree.lookup("/", &ps)
    assert.False(t, ok)
}

func TestTreeLookup_MultipleParams(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    r := newRoute("/users/:uid/posts/:pid", h, []string{GET})
    tree.insert("/users/:uid/posts/:pid", r)

    var ps Params
    got, ok := tree.lookup("/users/42/posts/100", &ps)
    assert.True(t, ok)
    assert.Same(t, r, got)
    assert.Eq(t, "42", ps.Get("uid"))
    assert.Eq(t, "100", ps.Get("pid"))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestTreeLookup .
```

Expected: build failure (`undefined: tree.lookup`).

- [ ] **Step 3: Implement tree.lookup with correct priority**

Append to `tree.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestTreeLookup . -v
```

Expected: `PASS` for all 8 sub-tests.

- [ ] **Step 5: Commit**

```bash
git add tree.go tree_test.go
git commit -m "feat(v2): radix tree lookup with static>param>wildcard priority (P-2)"
```

---

### Task 2.3: Priority bump and child sort (P-16)

**Files:**
- Modify: `tree.go`
- Test: `tree_test.go`

- [ ] **Step 1: Write the failing test**

Add to `tree_test.go`:

```go
func TestTree_PrioritySortsChildren(t *testing.T) {
    tree := newRadixTree()
    h := func(c *Context) {}
    // Register paths so 'b' subtree gets more entries than 'a' subtree.
    tree.insert("/a", newRoute("/a", h, []string{GET}))
    tree.insert("/b", newRoute("/b", h, []string{GET}))
    tree.insert("/b/x", newRoute("/b/x", h, []string{GET}))
    tree.insert("/b/y", newRoute("/b/y", h, []string{GET}))

    // After bump+sort, root.indices should have 'b' first (priority=3) then 'a' (priority=1).
    assert.Eq(t, byte('b'), tree.root.indices[0])
    assert.Eq(t, byte('a'), tree.root.indices[1])
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestTree_PrioritySortsChildren .
```

Expected: assertion failure (children stored in insertion order).

- [ ] **Step 3: Add priority bumping**

Modify `tree.go` `insert` function — after each successful leaf assignment, bump priority along the path. Also add helper:

Replace the `insert` function's leaf-assignment branch:

```go
        if len(remaining) == 0 {
            if n.route != nil {
                panic("rux: duplicate route registration: " + path)
            }
            n.route = route
            n.chain = route.chain
            t.bumpAlongPath(path)
            return
        }
```

And the wildcard branch's leaf creation:

```go
            n.wildcardChild = child
            t.bumpMaxParams(t.countParams(path))
            t.bumpAlongPath(path)
            return
```

Add helper functions to `tree.go`:

```go
// bumpAlongPath walks from root following path and increments priority on
// each visited node, then re-sorts each parent's static children by
// priority desc to keep hot paths at the front of indices/children.
func (t *radixTree) bumpAlongPath(path string) {
    n := t.root
    remaining := path
    for {
        cp := longestCommonPrefix(n.prefix, remaining)
        n.priority++
        remaining = remaining[cp:]
        if len(remaining) == 0 {
            return
        }
        c := remaining[0]
        switch c {
        case ':':
            if n.paramChild == nil {
                return
            }
            n = n.paramChild
            end := strings.IndexByte(remaining, '/')
            if end == -1 {
                end = len(remaining)
            }
            remaining = remaining[end:]
        case '*':
            if n.wildcardChild == nil {
                return
            }
            n.wildcardChild.priority++
            return
        default:
            i := n.staticChildIndex(c)
            if i < 0 {
                return
            }
            // Bump child priority then re-sort siblings.
            n.children[i].priority++
            sortChildrenByPriority(n)
            // After sort, find new index by first byte to keep walking.
            i = n.staticChildIndex(c)
            n = n.children[i]
        }
    }
}

// sortChildrenByPriority sorts a node's static children by priority desc
// using insertion sort (O(n²) but n is tiny — typically < 5).
func sortChildrenByPriority(n *node) {
    for i := 1; i < len(n.children); i++ {
        j := i
        for j > 0 && n.children[j].priority > n.children[j-1].priority {
            n.children[j], n.children[j-1] = n.children[j-1], n.children[j]
            n.indices[j], n.indices[j-1] = n.indices[j-1], n.indices[j]
            j--
        }
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestTree . -v
```

Expected: all tree tests PASS.

- [ ] **Step 5: Commit**

```bash
git add tree.go tree_test.go
git commit -m "feat(v2): bump priority and sort hot children to front (P-16)"
```

---

### Task 2.4: Optional segment expansion utility

**Files:**
- Modify: `utils.go`
- Test: `utils_test.go` (or `tree_test.go`)

- [ ] **Step 1: Write the failing test**

Add to `tree_test.go`:

```go
func TestParseOptionalSegments(t *testing.T) {
    cases := []struct {
        in   string
        want []string
    }{
        {"/posts", []string{"/posts"}},
        {"/posts[/{id}]", []string{"/posts", "/posts/:id"}},
        {"/api/users[/{name}]/profile",
            []string{"/api/users/profile", "/api/users/:name/profile"}},
        {"/about[.html]", []string{"/about", "/about.html"}},
    }
    for _, c := range cases {
        got := parseOptionalSegments(c.in)
        assert.Eq(t, c.want, got, "input=%q", c.in)
    }
}

func TestConvertParamSyntax(t *testing.T) {
    cases := map[string]string{
        "/users/{id}":          "/users/:id",
        "/users/{id}/posts":    "/users/:id/posts",
        "/files/{path:.+}":     "/files/*path",
        "/files/{path:.*}":     "/files/*path",
    }
    for in, want := range cases {
        assert.Eq(t, want, convertParamSyntax(in), "input=%q", in)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestParseOptional|TestConvertParam" .
```

Expected: build failure.

- [ ] **Step 3: Add helpers to utils.go**

Append to `utils.go`:

```go
// hasOptionalSegment reports whether path contains an optional segment
// like "[/{id}]" or "[.html]" outside of brace-quoted parameter regex.
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
                return true
            }
        }
    }
    return false
}

// parseOptionalSegments expands "/posts[/{id}]" into
// {"/posts", "/posts/:id"}. Always returns at least one element.
// Caller must have validated the path with ValidateOptionalSegments first.
func parseOptionalSegments(path string) []string {
    start := strings.IndexByte(path, '[')
    end := strings.IndexByte(path, ']')
    if start < 0 || end < 0 {
        return []string{convertParamSyntax(path)}
    }

    before := convertParamSyntax(path[:start])
    inner := convertParamSyntax(path[start+1 : end])
    after := ""
    if end+1 < len(path) {
        after = convertParamSyntax(path[end+1:])
    }
    return []string{before + after, before + inner + after}
}

// convertParamSyntax rewrites Rux's brace param syntax to colon syntax.
//   {id}        -> :id
//   {id:\d+}    -> :id          (regex stripped — see P-14)
//   {file:.+}   -> *file        (catch-all)
//   {file:.*}   -> *file
func convertParamSyntax(path string) string {
    for {
        start := strings.IndexByte(path, '{')
        if start == -1 {
            return path
        }
        // Find matching '}' with brace counting (regex may contain '{1,2}').
        depth := 0
        end := -1
        for i := start; i < len(path); i++ {
            switch path[i] {
            case '{':
                depth++
            case '}':
                depth--
                if depth == 0 {
                    end = i
                }
            }
            if end != -1 {
                break
            }
        }
        if end == -1 {
            return path
        }

        content := path[start+1 : end]
        name := content
        if colon := strings.IndexByte(content, ':'); colon > 0 {
            name = strings.TrimSpace(content[:colon])
            regex := strings.TrimSpace(content[colon+1:])
            if regex == ".+" || regex == ".*" {
                path = path[:start] + "*" + name + path[end+1:]
                continue
            }
        }
        path = path[:start] + ":" + name + path[end+1:]
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestParseOptional|TestConvertParam" .
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add utils.go tree_test.go
git commit -m "feat(v2): optional segment expansion + param syntax converter"
```

---

## Phase 3: Router (1.5 days)

### Task 3.1: Router struct and New()

**Files:**
- Modify: `router.go` (replacing existing)
- Test: `router_test.go` (will be expanded across tasks)

- [ ] **Step 1: Write the failing test**

Create minimal `router_test.go` (preserving the existing file's test data fixtures may be needed later — for now, replace it):

```go
package rux

import (
    "testing"

    "github.com/gookit/goutil/testutil/assert"
)

func TestNewRouter_Defaults(t *testing.T) {
    r := New()
    assert.NotNil(t, r)
    assert.Eq(t, "default", r.Name)
    assert.False(t, r.Frozen())
}

func TestNewRouter_WithOptions(t *testing.T) {
    r := New(StrictLastSlash, HandleMethodNotAllowed)
    assert.True(t, r.strictLastSlash)
    assert.True(t, r.handleMethodNotAllowed)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestNewRouter" .
```

Expected: build failure.

- [ ] **Step 3: Rewrite router.go (Router struct + New + Options + Freeze stubs)**

Replace `router.go` with:

```go
package rux

import (
    "strings"
    "sync"
    "sync/atomic"
)

// Router is the central registration & dispatch object.
type Router struct {
    Name string

    // Per-method static routes. Indexed by methodIndex().
    staticRoutes [methodCount]map[string]*Route

    // Per-method dynamic radix trees. nil if no dynamic routes for that method.
    dynamicTrees [methodCount]*radixTree

    namedRoutes map[string]*Route
    routeList   []*Route

    // Global middleware. Frozen into each route's finalChain on Freeze().
    globalChain HandlersChain

    currentGroupPrefix   string
    currentGroupHandlers HandlersChain

    noRoute   HandlersChain
    noAllowed HandlersChain

    // Settings.
    OnError                HandlerFunc
    OnPanic                HandlerFunc
    interceptAll           string
    useEncodedPath         bool
    strictLastSlash        bool
    handleMethodNotAllowed bool
    handleFallbackRoute    bool

    frozen     atomic.Bool
    counter    int

    ctxPool sync.Pool

    err error
}

// New constructs a Router with optional configuration.
func New(opts ...func(*Router)) *Router {
    r := &Router{
        Name:        "default",
        namedRoutes: make(map[string]*Route),
    }
    for _, opt := range opts {
        opt(r)
    }
    r.ctxPool.New = func() any {
        return &Context{router: r, index: -1}
    }
    return r
}

// Frozen reports whether the router is in frozen (read-only) state.
func (r *Router) Frozen() bool { return r.frozen.Load() }

/*************************************************************
 * Options
 *************************************************************/

// StrictLastSlash makes /path and /path/ distinct routes.
func StrictLastSlash(r *Router) { r.strictLastSlash = true }

// UseEncodedPath uses req.URL.EscapedPath() for matching.
func UseEncodedPath(r *Router) { r.useEncodedPath = true }

// HandleMethodNotAllowed enables 405 detection across methods.
func HandleMethodNotAllowed(r *Router) { r.handleMethodNotAllowed = true }

// HandleFallbackRoute enables the "/*" wildcard route as a global fallback.
func HandleFallbackRoute(r *Router) { r.handleFallbackRoute = true }

// InterceptAll redirects all requests to the given path.
func InterceptAll(path string) func(*Router) {
    return func(r *Router) {
        r.interceptAll = strings.TrimSpace(path)
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestNewRouter" .
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add router.go router_test.go
git commit -m "feat(v2): Router struct + New() + Option functions"
```

---

### Task 3.2: Add() and verb shortcuts (GET/POST/etc.) (P-1, P-5)

**Files:**
- Modify: `router.go`
- Test: `router_test.go`

- [ ] **Step 1: Write the failing test**

Append to `router_test.go`:

```go
func TestRouter_Add_Static(t *testing.T) {
    r := New()
    h := func(c *Context) {}
    route := r.Add("/users", h, GET)
    assert.NotNil(t, route)
    assert.Eq(t, "/users", route.Path())
    assert.Eq(t, []string{GET}, route.Methods())
    // Static route stored in staticRoutes[GET-idx].
    idx := methodIndex(GET)
    _, ok := r.staticRoutes[idx]["/users"]
    assert.True(t, ok)
}

func TestRouter_Add_Dynamic(t *testing.T) {
    r := New()
    h := func(c *Context) {}
    route := r.Add("/users/{id}", h, GET)
    assert.Eq(t, "/users/{id}", route.Path())
    // Dynamic route stored in dynamicTrees[GET-idx], not in staticRoutes.
    idx := methodIndex(GET)
    assert.NotNil(t, r.dynamicTrees[idx])
    assert.Nil(t, r.staticRoutes[idx])
}

func TestRouter_GET(t *testing.T) {
    r := New()
    r.GET("/x", func(c *Context) {})
    r.POST("/y", func(c *Context) {})
    assert.Eq(t, 2, r.counter)
}

func TestRouter_Any_RegistersAllMethods(t *testing.T) {
    r := New()
    r.Any("/wild", func(c *Context) {})
    for _, m := range AnyMethods() {
        idx := methodIndex(m)
        _, ok := r.staticRoutes[idx]["/wild"]
        assert.True(t, ok, "method %s missing", m)
    }
}

func TestRouter_AddAfterFreeze_Panics(t *testing.T) {
    r := New()
    r.GET("/x", func(c *Context) {})
    r.Freeze()
    assert.Panics(t, func() {
        r.GET("/y", func(c *Context) {})
    })
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestRouter_Add|TestRouter_GET|TestRouter_Any|TestRouter_AddAfter" .
```

Expected: build failure.

- [ ] **Step 3: Add registration methods to router.go**

Append to `router.go`:

```go
/*************************************************************
 * Route registration
 *************************************************************/

// Add registers a route on the given methods (defaults to GET if methods empty).
func (r *Router) Add(path string, handler HandlerFunc, methods ...string) *Route {
    route := newRoute(path, handler, methods)
    return r.AddRoute(route)
}

// AddNamed is like Add with a route name.
func (r *Router) AddNamed(name, path string, handler HandlerFunc, methods ...string) *Route {
    route := newNamedRoute(name, path, handler, methods)
    return r.AddRoute(route)
}

// AddRoute registers a pre-constructed Route.
func (r *Router) AddRoute(route *Route) *Route {
    r.appendRoute(route)
    return route
}

// Verb shortcuts.
func (r *Router) GET(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
    return r.Add(path, h, GET).Use(mw...)
}
func (r *Router) HEAD(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
    return r.Add(path, h, HEAD).Use(mw...)
}
func (r *Router) POST(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
    return r.Add(path, h, POST).Use(mw...)
}
func (r *Router) PUT(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
    return r.Add(path, h, PUT).Use(mw...)
}
func (r *Router) PATCH(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
    return r.Add(path, h, PATCH).Use(mw...)
}
func (r *Router) DELETE(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
    return r.Add(path, h, DELETE).Use(mw...)
}
func (r *Router) OPTIONS(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
    return r.Add(path, h, OPTIONS).Use(mw...)
}
func (r *Router) CONNECT(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
    return r.Add(path, h, CONNECT).Use(mw...)
}
func (r *Router) TRACE(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
    return r.Add(path, h, TRACE).Use(mw...)
}

// Any registers a route on every supported HTTP method.
func (r *Router) Any(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
    return r.Add(path, h, AnyMethods()...).Use(mw...)
}

// appendRoute formats the path, applies group context, expands optional
// segments, and dispatches to static or dynamic storage.
func (r *Router) appendRoute(route *Route) {
    if r.frozen.Load() {
        panic("rux: cannot add route after router is frozen")
    }

    // Apply current group prefix and middlewares.
    r.applyGroup(route)

    debugPrintRoute(route)

    if route.name != "" {
        r.namedRoutes[route.name] = route
    }
    r.routeList = append(r.routeList, route)

    // Expand optional segments into one or more concrete paths.
    if hasOptionalSegment(route.path) {
        // Validation panics on illegal positions.
        // (validateOptionalSegments lives in internal/util.)
        validateOptionalSegmentsPath(route.path)
        for _, expandedPath := range parseOptionalSegments(route.path) {
            expandedRoute := *route
            expandedRoute.path = normalizePath(expandedPath)
            r.registerSingleRoute(&expandedRoute)
        }
        r.counter++ // count as one user-defined route
        return
    }

    r.counter++
    // Convert {id} -> :id syntax for static-vs-dynamic detection consistency.
    route.path = normalizePath(convertParamSyntax(route.path))
    r.registerSingleRoute(route)
}

// registerSingleRoute stores route in the right per-method bucket.
func (r *Router) registerSingleRoute(route *Route) {
    if isStaticPath(route.path) {
        for _, m := range route.methods {
            idx := methodIndex(m)
            if idx < 0 {
                panic("rux: unknown method " + m)
            }
            if r.staticRoutes[idx] == nil {
                r.staticRoutes[idx] = make(map[string]*Route, 4)
            }
            if _, dup := r.staticRoutes[idx][route.path]; dup {
                panic("rux: duplicate static route: " + m + " " + route.path)
            }
            r.staticRoutes[idx][route.path] = route
        }
        return
    }

    for _, m := range route.methods {
        idx := methodIndex(m)
        if idx < 0 {
            panic("rux: unknown method " + m)
        }
        if r.dynamicTrees[idx] == nil {
            r.dynamicTrees[idx] = newRadixTree()
        }
        r.dynamicTrees[idx].insert(route.path, route)
    }
}

// applyGroup merges current group prefix and handlers into the route.
func (r *Router) applyGroup(route *Route) {
    routePath := r.formatPath(route.path)
    if r.currentGroupPrefix != "" {
        routePath = r.formatPath(r.currentGroupPrefix + routePath)
    }
    route.path = routePath

    if len(r.currentGroupHandlers) > 0 {
        // Group middlewares run before route's own middlewares.
        merged := make(HandlersChain, 0, len(r.currentGroupHandlers)+len(route.chain))
        merged = append(merged, r.currentGroupHandlers...)
        merged = append(merged, route.chain...)
        route.chain = merged
    }
}

// formatPath applies the router's path policy.
func (r *Router) formatPath(path string) string {
    if path == "" || path == "/" {
        return "/"
    }
    path = strings.TrimSpace(path)
    if !r.strictLastSlash && len(path) > 1 && path[len(path)-1] == '/' {
        path = strings.TrimRight(path, "/")
    }
    if path == "" || path == "/" {
        return "/"
    }
    if path[0] != '/' {
        return "/" + path
    }
    if path[1] == '/' {
        return "/" + strings.TrimLeft(path, "/")
    }
    return path
}
```

Add a thin wrapper to bridge the `internal/util.ValidateOptionalSegments`:

Append to `utils.go`:

```go
// validateOptionalSegmentsPath wraps internal/util.ValidateOptionalSegments
// so this package doesn't leak the import.
func validateOptionalSegmentsPath(path string) {
    util.ValidateOptionalSegments(path)
}
```

And add the import at top of `utils.go`:

```go
import (
    // ... existing imports ...
    "github.com/gookit/rux/internal/util"
)
```

Also add Freeze() stub so tests compile:

Append to `router.go`:

```go
// Freeze marks the router read-only. Subsequent route registration panics.
// First call merges global middleware into every route's finalChain.
// Idempotent.
func (r *Router) Freeze() {
    if !r.frozen.CompareAndSwap(false, true) {
        return
    }
    // Real freeze logic added in Phase 4.
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestRouter_Add|TestRouter_GET|TestRouter_Any|TestRouter_AddAfter" -v
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add router.go router_test.go utils.go
git commit -m "feat(v2): Add/verb shortcuts, static-vs-dynamic dispatch (P-1, P-5)"
```

---

### Task 3.3: Group, Use, NotFound, NotAllowed

**Files:**
- Modify: `router.go`
- Test: `router_test.go`

- [ ] **Step 1: Write the failing test**

Append to `router_test.go`:

```go
func TestRouter_Group(t *testing.T) {
    r := New()
    var hit string
    r.Group("/api", func() {
        r.GET("/users", func(c *Context) { hit = "users" })
        r.GET("/posts", func(c *Context) { hit = "posts" })
    })
    idx := methodIndex(GET)
    _, ok1 := r.staticRoutes[idx]["/api/users"]
    _, ok2 := r.staticRoutes[idx]["/api/posts"]
    assert.True(t, ok1)
    assert.True(t, ok2)
    _ = hit
}

func TestRouter_GroupMiddleware_PrefixedToRouteChain(t *testing.T) {
    r := New()
    var order []string
    apiMW := func(c *Context) { order = append(order, "api"); c.Next() }
    routeMW := func(c *Context) { order = append(order, "route"); c.Next() }
    main := func(c *Context) { order = append(order, "main") }

    r.Group("/api", func() {
        r.GET("/x", main, routeMW)
    }, apiMW)

    idx := methodIndex(GET)
    route := r.staticRoutes[idx]["/api/x"]
    assert.NotNil(t, route)
    // chain order should be [apiMW, routeMW, main]
    assert.Eq(t, 3, len(route.chain))
}

func TestRouter_Use_AddsToGlobalChain(t *testing.T) {
    r := New()
    mw := func(c *Context) {}
    r.Use(mw)
    assert.Eq(t, 1, len(r.globalChain))
}

func TestRouter_UseAfterRouteRegistration_Panics(t *testing.T) {
    r := New()
    r.GET("/x", func(c *Context) {})
    assert.Panics(t, func() {
        r.Use(func(c *Context) {})
    })
}

func TestRouter_NotFound(t *testing.T) {
    r := New()
    h := func(c *Context) {}
    r.NotFound(h)
    assert.Eq(t, 1, len(r.noRoute))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestRouter_Group|TestRouter_Use|TestRouter_NotFound" .
```

Expected: build failure.

- [ ] **Step 3: Implement Group / Use / NotFound / NotAllowed**

Append to `router.go`:

```go
/*************************************************************
 * Group / Use / NotFound / NotAllowed
 *************************************************************/

// Group registers all routes added inside fn under the given path prefix.
// Middlewares supplied as middles run before route-level middlewares.
func (r *Router) Group(prefix string, fn func(), middles ...HandlerFunc) {
    prevPrefix := r.currentGroupPrefix
    r.currentGroupPrefix = prevPrefix + r.formatPath(prefix)

    prevHandlers := r.currentGroupHandlers
    if len(middles) > 0 {
        if len(prevHandlers) > 0 {
            r.currentGroupHandlers = append(append(HandlersChain{}, prevHandlers...), middles...)
        } else {
            r.currentGroupHandlers = middles
        }
    }

    fn()

    r.currentGroupPrefix = prevPrefix
    r.currentGroupHandlers = prevHandlers
}

// Use appends global middleware. Must be called before any route registration
// or after an explicit reset (no reset API exists; prefer ordering).
func (r *Router) Use(handlers ...HandlerFunc) {
    if r.frozen.Load() {
        panic("rux: cannot Use after frozen")
    }
    if len(r.routeList) > 0 {
        panic("rux: Use must be called before any route registration (Q6)")
    }
    r.globalChain = append(r.globalChain, handlers...)
}

// NotFound sets the handlers chain for unmatched routes.
func (r *Router) NotFound(handlers ...HandlerFunc) {
    r.noRoute = handlers
}

// NotAllowed sets the handlers chain for HTTP 405 responses.
// (Only consulted when HandleMethodNotAllowed option is set.)
func (r *Router) NotAllowed(handlers ...HandlerFunc) {
    r.noAllowed = handlers
}

// Handlers returns the global middleware chain.
func (r *Router) Handlers() HandlersChain { return r.globalChain }
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestRouter_Group|TestRouter_Use|TestRouter_NotFound" -v
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add router.go router_test.go
git commit -m "feat(v2): Group + Use + NotFound + NotAllowed; Use must precede routes (Q6)"
```

---

### Task 3.4: Optional segment expansion end-to-end

**Files:**
- Test: `router_test.go`

- [ ] **Step 1: Write the failing test**

Append to `router_test.go`:

```go
func TestRouter_OptionalSegment_ExpandsToTwoRoutes(t *testing.T) {
    r := New()
    h := func(c *Context) {}
    r.GET("/posts[/{id}]", h)

    idxGet := methodIndex(GET)
    // Static branch /posts
    _, hasStatic := r.staticRoutes[idxGet]["/posts"]
    assert.True(t, hasStatic, "/posts (without id) should be static")

    // Dynamic branch /posts/:id
    assert.NotNil(t, r.dynamicTrees[idxGet])
    var ps Params
    route, ok := r.dynamicTrees[idxGet].lookup("/posts/42", &ps)
    assert.True(t, ok)
    assert.NotNil(t, route)
    assert.Eq(t, "42", ps.Get("id"))
}

func TestRouter_OptionalSegment_InvalidPosition_Panics(t *testing.T) {
    r := New()
    assert.Panics(t, func() {
        r.GET("/posts[/{cat}]/{id}", func(c *Context) {})
    })
}
```

- [ ] **Step 2: Run test to verify it fails or passes**

```bash
go test -run "TestRouter_OptionalSegment" -v
```

Expected: PASS (the appendRoute path already wires expansion via Task 3.2). If a test fails, fix the corresponding logic in router.go before committing.

- [ ] **Step 3: (Conditional) Fix any failures**

Common issue: `route.chain` is shared across expanded copies. The shallow copy `expandedRoute := *route` shares the underlying chain slice but creates a new struct, so modifications to `expandedRoute.chain` would mutate the original. Since we only read chain after this point (Freeze adds finalChain), this is safe — verify by re-reading `appendRoute` in router.go.

- [ ] **Step 4: Re-run tests**

```bash
go test -v .
```

Expected: all current tests PASS.

- [ ] **Step 5: Commit (only if there were fixes)**

```bash
git add router.go router_test.go
git commit -m "test(v2): optional segment end-to-end registration"
```

---

### Task 3.5: Resource and Controller helpers

**Files:**
- Modify: `router.go`
- Test: `router_test.go`

- [ ] **Step 1: Write the failing test**

Append to `router_test.go`:

```go
type fakeController struct{}

func (f *fakeController) AddRoutes(g *Router) {
    g.GET("/", func(c *Context) {})
    g.POST("/", func(c *Context) {})
}

func TestRouter_Controller(t *testing.T) {
    r := New()
    r.Controller("/api", &fakeController{})
    idx := methodIndex(GET)
    _, ok := r.staticRoutes[idx]["/api"]
    assert.True(t, ok, "GET /api should be registered")
    idxPost := methodIndex(POST)
    _, ok = r.staticRoutes[idxPost]["/api"]
    assert.True(t, ok, "POST /api should be registered")
}

type fakeResource struct{}

func (f *fakeResource) Index(c *Context)  {}
func (f *fakeResource) Show(c *Context)   {}
func (f *fakeResource) Store(c *Context)  {}
func (f *fakeResource) Update(c *Context) {}
func (f *fakeResource) Delete(c *Context) {}

func TestRouter_Resource(t *testing.T) {
    r := New()
    r.Resource("/", &fakeResource{})
    // Resource registers under the lowercase struct name = "fakeresource"
    // GET /fakeresource (Index) and GET /fakeresource/{id} (Show), etc.
    routes := r.Routes()
    paths := make(map[string]bool)
    for _, ri := range routes {
        paths[ri.Path] = true
    }
    assert.True(t, paths["/fakeresource"])
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestRouter_Controller|TestRouter_Resource" .
```

Expected: build failure.

- [ ] **Step 3: Implement Controller and Resource**

Append to `router.go`:

```go
import "reflect"  // add to existing import block at top
```

Append to `router.go` body:

```go
// Controller registers all routes from the controller's AddRoutes method
// under the given basePath, with shared middlewares.
func (r *Router) Controller(basePath string, c ControllerFace, middles ...HandlerFunc) {
    r.Group(basePath, func() {
        c.AddRoutes(r)
    }, middles...)
}

// Resource registers RESTful routes for the given controller struct.
// See package docs for the conventional method names.
func (r *Router) Resource(basePath string, controller any, middles ...HandlerFunc) {
    cv := reflect.ValueOf(controller)
    if cv.Kind() != reflect.Ptr {
        panic("rux: Resource controller must be a pointer")
    }
    ct := cv.Type()
    if cv.Elem().Type().Kind() != reflect.Struct {
        panic("rux: Resource controller must be a pointer to struct")
    }

    var perActionMW = map[string][]HandlerFunc{}
    if m := cv.MethodByName("Uses"); m.IsValid() {
        if uses, ok := m.Interface().(func() map[string][]HandlerFunc); ok {
            perActionMW = uses()
        }
    }

    resName := strings.ToLower(ct.Elem().Name())
    basePath = strings.TrimRight(basePath, "/") + "/" + resName

    r.Group(basePath, func() {
        for action, methods := range RESTFulActions {
            m := cv.MethodByName(action)
            if !m.IsValid() {
                continue
            }
            handler, ok := m.Interface().(func(*Context))
            if !ok {
                continue
            }
            routeName := resName + "_" + strings.ToLower(action)
            var route *Route
            switch action {
            case IndexAction, StoreAction:
                route = r.AddNamed(routeName, "/", handler, methods...)
            case CreateAction:
                route = r.AddNamed(routeName, "/"+strings.ToLower(action)+"/", handler, methods...)
            case EditAction:
                route = r.AddNamed(routeName, "{id}/"+strings.ToLower(action)+"/", handler, methods...)
            default: // Show, Update, Delete
                route = r.AddNamed(routeName, "{id}/", handler, methods...)
            }
            if mws, ok := perActionMW[action]; ok {
                route.Use(mws...)
            }
        }
    }, middles...)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestRouter_Controller|TestRouter_Resource" -v
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add router.go router_test.go
git commit -m "feat(v2): Controller + Resource helpers"
```

---

### Task 3.6: Static file helpers

**Files:**
- Modify: `router.go`

- [ ] **Step 1: Write the failing test**

Append to `router_test.go`:

```go
func TestRouter_StaticFile_RegistersGetRoute(t *testing.T) {
    r := New()
    r.StaticFile("/favicon.ico", "./testdata/favicon.ico")
    idx := methodIndex(GET)
    _, ok := r.staticRoutes[idx]["/favicon.ico"]
    assert.True(t, ok)
}

func TestRouter_StaticDir_RegistersWildcard(t *testing.T) {
    r := New()
    r.StaticDir("/assets", "./testdata")
    // Wildcard goes into dynamicTrees (the "*file" suffix makes it dynamic).
    idx := methodIndex(GET)
    assert.NotNil(t, r.dynamicTrees[idx])
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestRouter_Static" .
```

Expected: build failure.

- [ ] **Step 3: Implement static helpers**

Append to `router.go`:

```go
import "net/http"  // add to existing import block

import "fmt"  // already may be imported
```

Append to `router.go` body:

```go
/*************************************************************
 * Static file helpers
 *************************************************************/

// StaticFile registers a single static file under the given path.
func (r *Router) StaticFile(path, filePath string) *Route {
    return r.GET(path, func(c *Context) { c.File(filePath) })
}

// StaticDir serves files from rootDir under prefixURL using http.FileServer.
func (r *Router) StaticDir(prefixURL, rootDir string) *Route {
    fs := http.StripPrefix(prefixURL, http.FileServer(http.Dir(rootDir)))
    return r.GET(prefixURL+"/*file", func(c *Context) {
        fs.ServeHTTP(c.Resp, c.Req)
    })
}

// StaticFS serves files from the given http.FileSystem under prefixURL.
func (r *Router) StaticFS(prefixURL string, fs http.FileSystem) *Route {
    handler := http.StripPrefix(prefixURL, http.FileServer(fs))
    return r.GET(prefixURL+"/*file", func(c *Context) {
        handler.ServeHTTP(c.Resp, c.Req)
    })
}

// StaticFiles serves files from rootDir under prefixURL, optionally
// filtered by extension list (pipe-separated, e.g. "css|js|html").
// The exts argument is reserved for future use.
func (r *Router) StaticFiles(prefixURL, rootDir, exts string) *Route {
    fs := http.FileServer(http.Dir(rootDir))
    return r.GET(fmt.Sprintf("%s/*file", prefixURL), func(c *Context) {
        c.Req.URL.Path = c.Param("file")
        fs.ServeHTTP(c.Resp, c.Req)
    })
    _ = exts
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestRouter_Static" -v
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add router.go router_test.go
git commit -m "feat(v2): static file helpers (StaticFile/Dir/FS/Files)"
```

---

### Task 3.7: GetRoute, Routes, IterateRoutes, BuildURL

**Files:**
- Modify: `router.go`
- Modify: `extends.go` (light edits to remove deprecated regex)

- [ ] **Step 1: Write the failing test**

Append to `router_test.go`:

```go
func TestRouter_GetRoute_NamedRoute(t *testing.T) {
    r := New()
    r.AddNamed("user_show", "/users/{id}", func(c *Context) {}, GET)
    rt := r.GetRoute("user_show")
    assert.NotNil(t, rt)
    assert.Eq(t, "/users/:id", rt.Path()) // converted at registration
}

func TestRouter_BuildURL(t *testing.T) {
    r := New()
    r.AddNamed("user_show", "/users/{id}", func(c *Context) {}, GET)
    u := r.BuildURL("user_show", M{"id": 42})
    assert.Eq(t, "/users/42", u.Path)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestRouter_GetRoute|TestRouter_BuildURL" .
```

Expected: build failure (`undefined: M`, etc.) or assertion mismatch.

- [ ] **Step 3: Add helpers to router.go**

Append to `router.go`:

```go
/*************************************************************
 * Inspection
 *************************************************************/

// GetRoute returns a named route or nil.
func (r *Router) GetRoute(name string) *Route { return r.namedRoutes[name] }

// NamedRoutes returns the map of named routes.
func (r *Router) NamedRoutes() map[string]*Route { return r.namedRoutes }

// Routes returns all routes as RouteInfo snapshots in registration order.
func (r *Router) Routes() []RouteInfo {
    out := make([]RouteInfo, 0, len(r.routeList))
    for _, route := range r.routeList {
        out = append(out, route.Info())
    }
    return out
}

// IterateRoutes calls fn for each registered route in registration order.
func (r *Router) IterateRoutes(fn func(*Route)) {
    for _, route := range r.routeList {
        fn(route)
    }
}

// Err returns the most recent error from Listen*.
func (r *Router) Err() error { return r.err }

// String returns a human-readable snapshot of registered routes.
func (r *Router) String() string {
    var b strings.Builder
    fmt.Fprintf(&b, "Routes Count: %d\n", r.counter)
    for _, route := range r.routeList {
        fmt.Fprintf(&b, "  %s\n", route)
    }
    return b.String()
}
```

The `extends.go` file already provides `BuildRequestURL` and the `M` type
(a `map[string]any`). It uses `regexp` to scan brace-style placeholders.
We keep that file's behavior; just verify the import compiles.

Also add the `BuildURL` shortcut on Router (extends.go already has equivalents
in v1 — keep them; they call into `Route.ToURL` which we keep unchanged):

Append to `router.go`:

```go
// BuildURL builds a URL by named route, supporting M{} or k,v,k,v args.
func (r *Router) BuildURL(name string, buildArgs ...any) *url.URL {
    route := r.GetRoute(name)
    if route == nil {
        panic("rux: BuildURL: unknown route " + name)
    }
    return route.ToURL(buildArgs...)
}
```

Add `"net/url"` to the import block in `router.go`.

`Route.ToURL` and `M` come from `extends.go` (existing). We adapt `extends.go` lightly — replace the file with the verbatim v1 content but ensure no v1-specific Route fields are referenced. Verify by searching:

```bash
grep -n 'r\.regex\|r\.start\|r\.spath\|r\.matches\|r\.params' extends.go
```

Expected: no matches. If matches exist, remove those lines.

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestRouter_GetRoute|TestRouter_BuildURL" -v
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add router.go extends.go router_test.go
git commit -m "feat(v2): inspection (Routes/IterateRoutes/GetRoute) + BuildURL"
```

---

## Phase 4: Freeze + Dispatch (1 day)

### Task 4.1: Freeze — global chain merge (P-6, P-8)

**Files:**
- Modify: `router.go`
- Test: `router_test.go`

- [ ] **Step 1: Write the failing test**

Append to `router_test.go`:

```go
func TestFreeze_MergesGlobalChainIntoRouteFinalChain(t *testing.T) {
    r := New()
    var order []string
    g1 := func(c *Context) { order = append(order, "g1") }
    g2 := func(c *Context) { order = append(order, "g2") }
    main := func(c *Context) { order = append(order, "main") }
    r.Use(g1, g2)
    route := r.GET("/x", main)

    r.Freeze()

    // finalChain = [g1, g2, main]
    assert.Eq(t, 3, len(route.finalChain))
    _ = order
}

func TestFreeze_NoGlobalChain_FinalChainIsRouteChain(t *testing.T) {
    r := New()
    main := func(c *Context) {}
    route := r.GET("/x", main)
    r.Freeze()
    assert.Eq(t, 1, len(route.finalChain))
}

func TestFreeze_Idempotent(t *testing.T) {
    r := New()
    r.GET("/x", func(c *Context) {})
    r.Freeze()
    r.Freeze() // must not panic
    assert.True(t, r.Frozen())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestFreeze_Merges|TestFreeze_NoGlobal|TestFreeze_Idempotent" .
```

Expected: assertion failure (`finalChain` is nil).

- [ ] **Step 3: Implement chain merging in Freeze**

Replace the `Freeze` stub in `router.go`:

```go
// Freeze marks the router read-only. Subsequent route registration panics.
// First call merges global middleware into every route's finalChain
// and mirrors GET routes to HEAD where no explicit HEAD exists.
// Idempotent.
func (r *Router) Freeze() {
    if !r.frozen.CompareAndSwap(false, true) {
        return
    }

    for _, route := range r.routeList {
        if len(r.globalChain) == 0 {
            route.finalChain = route.chain
            continue
        }
        merged := make(HandlersChain, 0, len(r.globalChain)+len(route.chain))
        merged = append(merged, r.globalChain...)
        merged = append(merged, route.chain...)
        route.finalChain = merged
    }

    // HEAD mirroring lives in Task 4.2.
    r.mirrorGetToHead()
}
```

Add a placeholder `mirrorGetToHead`:

```go
// mirrorGetToHead is implemented in Task 4.2.
func (r *Router) mirrorGetToHead() {}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestFreeze" -v
```

Expected: all 3 PASS.

- [ ] **Step 5: Commit**

```bash
git add router.go router_test.go
git commit -m "feat(v2): Freeze merges globalChain into route.finalChain (P-6, P-8)"
```

---

### Task 4.2: HEAD mirroring (P-9)

**Files:**
- Modify: `router.go`
- Modify: `tree.go`
- Test: `router_test.go`

- [ ] **Step 1: Write the failing test**

Append to `router_test.go`:

```go
func TestFreeze_MirrorsGetToHead_Static(t *testing.T) {
    r := New()
    main := func(c *Context) {}
    r.GET("/x", main)
    r.Freeze()
    idx := methodIndex(HEAD)
    _, ok := r.staticRoutes[idx]["/x"]
    assert.True(t, ok, "GET /x should mirror to HEAD /x")
}

func TestFreeze_MirrorsGetToHead_Dynamic(t *testing.T) {
    r := New()
    r.GET("/users/{id}", func(c *Context) {})
    r.Freeze()
    idx := methodIndex(HEAD)
    assert.NotNil(t, r.dynamicTrees[idx])
    var ps Params
    _, ok := r.dynamicTrees[idx].lookup("/users/42", &ps)
    assert.True(t, ok)
}

func TestFreeze_DoesNotOverrideExplicitHead(t *testing.T) {
    r := New()
    var headExplicit, getExplicit bool
    r.GET("/x", func(c *Context) { getExplicit = true })
    r.HEAD("/x", func(c *Context) { headExplicit = true })
    r.Freeze()

    idx := methodIndex(HEAD)
    route := r.staticRoutes[idx]["/x"]
    assert.NotNil(t, route)
    // The HEAD route should be the explicit one, not the GET mirror.
    route.chain[0](nil)
    assert.True(t, headExplicit)
    assert.False(t, getExplicit)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestFreeze_Mirrors" .
```

Expected: assertion failure (no HEAD route).

- [ ] **Step 3: Implement HEAD mirroring + tree walk**

Add a `walk` method to `tree.go`:

```go
// walk traverses the tree depth-first and yields each leaf's reconstructed
// path along with the leaf node.
func (t *radixTree) walk(fn func(path string, leaf *node)) {
    walkFrom(t.root, "", fn)
}

func walkFrom(n *node, accumulated string, fn func(string, *node)) {
    var label string
    switch n.nType {
    case nodeRoot:
        label = n.prefix
    default:
        label = n.prefix
    }
    here := accumulated + label

    if n.route != nil {
        fn(here, n)
    }
    for _, child := range n.children {
        walkFrom(child, here, fn)
    }
    if n.paramChild != nil {
        walkFrom(n.paramChild, here, fn)
    }
    if n.wildcardChild != nil {
        walkFrom(n.wildcardChild, here, fn)
    }
}

// hasExact reports whether the tree has a route at exactly path.
func (t *radixTree) hasExact(path string) bool {
    found := false
    t.walk(func(p string, _ *node) {
        if p == path {
            found = true
        }
    })
    return found
}
```

Replace `mirrorGetToHead` in `router.go`:

```go
func (r *Router) mirrorGetToHead() {
    getIdx := methodIndex(GET)
    headIdx := methodIndex(HEAD)

    // Mirror static routes.
    if getStatic := r.staticRoutes[getIdx]; getStatic != nil {
        if r.staticRoutes[headIdx] == nil {
            r.staticRoutes[headIdx] = make(map[string]*Route, len(getStatic))
        }
        head := r.staticRoutes[headIdx]
        for path, route := range getStatic {
            if _, exists := head[path]; !exists {
                head[path] = route
            }
        }
    }

    // Mirror dynamic routes.
    if get := r.dynamicTrees[getIdx]; get != nil {
        if r.dynamicTrees[headIdx] == nil {
            r.dynamicTrees[headIdx] = newRadixTree()
        }
        head := r.dynamicTrees[headIdx]
        get.walk(func(path string, leaf *node) {
            if !head.hasExact(path) {
                head.insert(path, leaf.route)
            }
        })
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run "TestFreeze_Mirrors" -v
```

Expected: all 3 PASS.

- [ ] **Step 5: Commit**

```bash
git add router.go tree.go router_test.go
git commit -m "feat(v2): mirror GET routes to HEAD on Freeze (P-9)"
```

---

### Task 4.3: Match() public API (P-12)

**Files:**
- Modify: `router.go`
- Test: `router_test.go`

- [ ] **Step 1: Write the failing test**

Append to `router_test.go`:

```go
func TestMatch_Static(t *testing.T) {
    r := New()
    h := func(c *Context) {}
    r.GET("/users", h)
    r.Freeze()

    route, params, ok := r.Match(GET, "/users")
    assert.True(t, ok)
    assert.NotNil(t, route)
    assert.Eq(t, 0, len(params))
}

func TestMatch_Dynamic(t *testing.T) {
    r := New()
    r.GET("/users/{id}", func(c *Context) {})
    r.Freeze()

    route, params, ok := r.Match(GET, "/users/42")
    assert.True(t, ok)
    assert.NotNil(t, route)
    assert.Eq(t, 1, len(params))
    assert.Eq(t, "id", params[0].Key)
    assert.Eq(t, "42", params[0].Value)
}

func TestMatch_Miss(t *testing.T) {
    r := New()
    r.Freeze()
    _, _, ok := r.Match(GET, "/nothing")
    assert.False(t, ok)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestMatch . -v
```

Expected: build failure (`undefined: r.Match`).

- [ ] **Step 3: Implement Match**

Append to `router.go`:

```go
/*************************************************************
 * Match — debug / offline lookup. Hot path uses internal match().
 *************************************************************/

// Match looks up route + params for an offline test or debugging.
// The hot serving path does NOT call this — it uses an internal
// signature that writes params directly into Context with zero allocation.
//
// Returns a heap-allocated []Param slice for ergonomic API usage.
func (r *Router) Match(method, path string) (*Route, []Param, bool) {
    if !r.frozen.Load() {
        r.Freeze()
    }
    idx := methodIndex(strings.ToUpper(method))
    if idx < 0 {
        return nil, nil, false
    }
    path = r.formatPath(path)

    // 1. Static.
    if m := r.staticRoutes[idx]; m != nil {
        if route, ok := m[path]; ok {
            return route, nil, true
        }
    }
    // 2. Dynamic.
    if tree := r.dynamicTrees[idx]; tree != nil {
        var ps Params
        if route, ok := tree.lookup(path, &ps); ok {
            return route, ps.Snapshot(), true
        }
    }
    return nil, nil, false
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestMatch -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add router.go router_test.go
git commit -m "feat(v2): public Match API returning ([]Param, bool) (P-12)"
```

---

### Task 4.4: Context skeleton (P-15)

**Files:**
- Modify: `context.go` (replacing existing)
- Test: `context_test.go`

- [ ] **Step 1: Write the failing test**

Replace `context_test.go` minimally:

```go
package rux

import (
    "net/http/httptest"
    "testing"

    "github.com/gookit/goutil/testutil/assert"
)

func TestContext_Reset(t *testing.T) {
    c := &Context{}
    req := httptest.NewRequest("GET", "/x", nil)
    w := httptest.NewRecorder()
    c.Init(w, req)
    assert.Same(t, req, c.Req)
    assert.NotNil(t, c.Resp)
}

func TestContext_Param(t *testing.T) {
    c := &Context{}
    c.params.append("id", "42")
    assert.Eq(t, "42", c.Param("id"))
    assert.Eq(t, "", c.Param("missing"))
}

func TestContext_SetGet_LazyMap(t *testing.T) {
    c := &Context{}
    _, ok := c.Get("missing")
    assert.False(t, ok)
    assert.Nil(t, c.data) // not allocated yet

    c.Set("k", 42)
    v, ok := c.Get("k")
    assert.True(t, ok)
    assert.Eq(t, 42, v)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestContext . -v
```

Expected: build failure or assertion failure.

- [ ] **Step 3: Rewrite context.go (skeleton — render helpers in Phase 5)**

Read current context.go to extract the render helper signatures, then rewrite the head of `context.go`:

```bash
grep -n '^func (c \*Context)' context.go
```

Then replace the **top portion** of `context.go` (struct + lifecycle + params/typed accessors) — keep render/binding methods intact at the bottom for now (will be cleaned up in Phase 5 Task 5.2). Replace lines from `package rux` through the end of the existing struct definition with:

```go
package rux

import (
    "errors"
    "net/http"
)

// Context carries the per-request state through the handler chain.
type Context struct {
    Req  *http.Request
    Resp http.ResponseWriter
    writer responseWriter

    router *Router

    // Path params, inlined for zero-allocation parameter passing.
    params Params

    // Hot fields used by every request — typed, not in map.
    matchedRoute *Route
    matchedPath  string

    // Handler chain (already merged at Freeze time).
    handlers HandlersChain
    index    int8

    // Errors accumulated during handling.
    Errors []error

    // Lazy-init bag for user data.
    data map[string]any
}

// Init prepares c for a new request without re-allocating internal slices.
func (c *Context) Init(w http.ResponseWriter, req *http.Request) {
    c.Req = req
    c.Resp = w
    c.writer.reset(w)
    c.Resp = &c.writer
    c.params.Reset()
    c.matchedRoute = nil
    c.matchedPath = ""
    c.handlers = nil
    c.index = -1
    if c.Errors != nil {
        c.Errors = c.Errors[:0]
    }
    if c.data != nil {
        for k := range c.data {
            delete(c.data, k)
        }
    }
}

// Reset is an alias for Init that re-uses the current Req/Resp.
func (c *Context) Reset() {
    c.Init(c.Resp, c.Req)
}

// Param returns the value of the named path parameter.
func (c *Context) Param(name string) string { return c.params.Get(name) }

// Params returns a pointer to the inlined params (avoids 16-Param copy).
func (c *Context) Params() *Params { return &c.params }

// Route returns the matched Route or nil.
func (c *Context) Route() *Route { return c.matchedRoute }

// MatchedPath returns the route's registered path (with placeholders).
func (c *Context) MatchedPath() string { return c.matchedPath }

// Set stores arbitrary user data. Allocates the map on first call.
func (c *Context) Set(key string, value any) {
    if c.data == nil {
        c.data = make(map[string]any, 4)
    }
    c.data[key] = value
}

// Get retrieves user data set by Set.
func (c *Context) Get(key string) (any, bool) {
    if c.data == nil {
        return nil, false
    }
    v, ok := c.data[key]
    return v, ok
}

// SafeGet returns the value or panics — for required keys.
func (c *Context) SafeGet(key string) any {
    v, ok := c.Get(key)
    if !ok {
        panic("rux: missing context key " + key)
    }
    return v
}

// SetHandlers installs the (already merged) handler chain.
func (c *Context) SetHandlers(chain HandlersChain) {
    c.handlers = chain
    c.index = -1
}

// Next advances to the next handler in the chain.
func (c *Context) Next() {
    c.index++
    for c.index < int8(len(c.handlers)) {
        c.handlers[c.index](c)
        c.index++
    }
}

// Abort prevents subsequent handlers in the chain from running.
func (c *Context) Abort() { c.index = abortIndex }

// IsAborted reports whether Abort was called.
func (c *Context) IsAborted() bool { return c.index >= abortIndex }

// AbortWithStatus aborts and writes the given HTTP status.
func (c *Context) AbortWithStatus(status int) {
    c.Resp.WriteHeader(status)
    c.Abort()
}

// AddError records an error to be processed by Router.OnError.
func (c *Context) AddError(err error) {
    if err == nil {
        return
    }
    c.Errors = append(c.Errors, err)
}

// Err returns the most recent error or nil.
func (c *Context) Err() error {
    if len(c.Errors) == 0 {
        return nil
    }
    return c.Errors[len(c.Errors)-1]
}

// SetStatus writes the HTTP status code to the response.
func (c *Context) SetStatus(status int) { c.Resp.WriteHeader(status) }

// SetHeader sets a response header value.
func (c *Context) SetHeader(key, value string) { c.Resp.Header().Set(key, value) }

// ErrEmptyHandlers indicates a misconfigured route.
var ErrEmptyHandlers = errors.New("rux: empty handler chain")
```

(The render/binding methods further down in the existing `context.go` will remain
in place — many compile fine because they only use `c.Resp`/`c.Req`/`c.SetHeader`
which are preserved. Compile-fix any leftovers in Task 5.2.)

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestContext -v
```

Expected: PASS for the 3 sub-tests above. (Other context_test.go tests may fail — they get fixed in Task 5.2.)

- [ ] **Step 5: Commit**

```bash
git add context.go context_test.go
git commit -m "feat(v2): Context skeleton — inline Params + typed fields (P-15)"
```

---

### Task 4.5: Rename response_wirter typo

**Files:**
- Rename: `response_wirter.go` → `response_writer.go`
- Rename: `response_wirter_test.go` → `response_writer_test.go`

The existing file already defines `responseWriter` with `reset(w http.ResponseWriter)`,
`ensureWriteHeader()`, `Header()`, `WriteHeader(status)`, `Write([]byte)`,
`Flush()`, `Hijack()`. No method changes needed — only fix the filename typo.

- [ ] **Step 1: Verify current content**

```bash
grep -E '^func \(w \*responseWriter\)' response_wirter.go
```

Expected: lists `reset`, `Status`, `Written`, `Length`, `Header`, `WriteHeader`,
`Write`, `Flush`, `Hijack`, `ensureWriteHeader` — all needed by Context/dispatch.

- [ ] **Step 2: Rename**

```bash
git mv response_wirter.go response_writer.go
git mv response_wirter_test.go response_writer_test.go
git status
```

Expected: shows the renames.

- [ ] **Step 3: Build**

```bash
go build .
```

Expected: success — the responseWriter type is in package rux regardless
of filename.

- [ ] **Step 4: Confirm Context.Init's `c.writer.reset(w)` call from Task 4.4 works**

```bash
grep -n 'c\.writer\.reset' context.go
```

Expected: matches the call in Init (added in Task 4.4). Cross-checked
against the existing `reset(w2 http.ResponseWriter)` signature — argument
type matches.

- [ ] **Step 5: Commit**

```bash
git add response_writer.go response_writer_test.go
git commit -m "chore(v2): fix response_wirter typo in filename"
```

---

### Task 4.6: ServeHTTP hot path (lock-free)

**Files:**
- Modify: `dispatch.go` (replacing existing)
- Test: `dispatch_test.go`

- [ ] **Step 1: Write the failing test**

Replace `dispatch_test.go` (or extend) with:

```go
package rux

import (
    "io"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/gookit/goutil/testutil/assert"
)

func TestServeHTTP_StaticHit(t *testing.T) {
    r := New()
    r.GET("/users", func(c *Context) { c.Text(200, "hi") })

    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/users", nil)
    r.ServeHTTP(w, req)

    assert.Eq(t, 200, w.Code)
    body, _ := io.ReadAll(w.Body)
    assert.Eq(t, "hi", string(body))
}

func TestServeHTTP_DynamicHit_BindsParam(t *testing.T) {
    r := New()
    r.GET("/users/{id}", func(c *Context) {
        c.Text(200, "id="+c.Param("id"))
    })

    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/users/42", nil)
    r.ServeHTTP(w, req)

    body, _ := io.ReadAll(w.Body)
    assert.True(t, strings.Contains(string(body), "id=42"))
}

func TestServeHTTP_404(t *testing.T) {
    r := New()
    r.GET("/x", func(c *Context) {})

    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/nothing", nil)
    r.ServeHTTP(w, req)

    assert.Eq(t, 404, w.Code)
}

func TestServeHTTP_HEAD_FallsBackToGET(t *testing.T) {
    r := New()
    r.GET("/x", func(c *Context) { c.Text(200, "x") })

    w := httptest.NewRecorder()
    req := httptest.NewRequest("HEAD", "/x", nil)
    r.ServeHTTP(w, req)

    assert.Eq(t, 200, w.Code)
}

func TestServeHTTP_TriggersFreeze(t *testing.T) {
    r := New()
    r.GET("/x", func(c *Context) {})
    assert.False(t, r.Frozen())

    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/x", nil)
    r.ServeHTTP(w, req)

    assert.True(t, r.Frozen())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestServeHTTP . -v
```

Expected: build failure or 404 / wrong status.

- [ ] **Step 3: Rewrite dispatch.go**

Replace `dispatch.go` with:

```go
package rux

import (
    "fmt"
    "net"
    "net/http"
    "os"
    "sort"
    "strings"
)

var internal404Handler HandlerFunc = func(c *Context) {
    http.NotFound(c.Resp, c.Req)
}

var internal405Handler HandlerFunc = func(c *Context) {
    allowed, _ := c.Get(CTXAllowedMethods)
    if list, ok := allowed.([]string); ok {
        sort.Strings(list)
        c.SetHeader("Allow", strings.Join(list, ", "))
    }
    if c.Req.Method == OPTIONS {
        c.SetStatus(200)
    } else {
        http.Error(c.Resp, "Method not allowed", 405)
    }
}

/*************************************************************
 * Listening
 *************************************************************/

// Listen serves HTTP on the given address.
func (r *Router) Listen(addr ...string) {
    defer func() { debugPrintError(r.err) }()
    address := resolveAddress(addr)
    fmt.Printf("Serve listen on %s. Go to http://%s\n", address, address)
    r.err = http.ListenAndServe(address, r)
}

// ListenTLS serves HTTPS on the given address.
func (r *Router) ListenTLS(addr, certFile, keyFile string) {
    var err error
    defer func() { debugPrintError(err) }()
    address := resolveAddress([]string{addr})
    fmt.Printf("Serve listen on %s. Go to https://%s\n", address, address)
    err = http.ListenAndServeTLS(address, certFile, keyFile, r)
    r.err = err
}

// ListenUnix serves on a Unix domain socket.
func (r *Router) ListenUnix(file string) {
    var err error
    defer func() { debugPrintError(err) }()
    fmt.Printf("Serve listen on unix:/%s\n", file)
    if err = os.Remove(file); err != nil && !os.IsNotExist(err) {
        r.err = err
        return
    }
    listener, err := net.Listen("unix", file)
    if err != nil {
        r.err = err
        return
    }
    err = http.Serve(listener, r)
    _ = listener.Close()
    r.err = err
}

// WrapHTTPHandlers wraps the router with one or more http.Handler middlewares.
func (r *Router) WrapHTTPHandlers(preHandlers ...func(http.Handler) http.Handler) http.Handler {
    var wrapped http.Handler = r
    for i := len(preHandlers) - 1; i >= 0; i-- {
        wrapped = preHandlers[i](wrapped)
    }
    return wrapped
}

/*************************************************************
 * ServeHTTP — hot path
 *************************************************************/

// ServeHTTP implements http.Handler. Triggers lazy Freeze on first call.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    if !r.frozen.Load() {
        r.Freeze()
    }

    ctx := r.ctxPool.Get().(*Context)
    ctx.Init(w, req)

    r.handle(ctx)

    r.ctxPool.Put(ctx)
}

// HandleContext re-uses an externally constructed Context.
func (r *Router) HandleContext(c *Context) {
    if !r.frozen.Load() {
        r.Freeze()
    }
    c.Reset()
    r.handle(c)
    r.ctxPool.Put(c)
}

func (r *Router) handle(ctx *Context) {
    if r.OnPanic != nil {
        defer func() {
            if rec := recover(); rec != nil {
                ctx.Set(CTXRecoverResult, rec)
                r.OnPanic(ctx)
            }
        }()
    }

    path := ctx.Req.URL.Path
    if r.useEncodedPath {
        path = ctx.Req.URL.EscapedPath()
    }
    if r.interceptAll != "" {
        path = r.interceptAll
    } else {
        path = r.formatPath(path)
    }

    method := ctx.Req.Method
    idx := methodIndex(method)

    var route *Route
    if idx >= 0 {
        // 1. Static — single map lookup, no string concat. (P-5)
        if m := r.staticRoutes[idx]; m != nil {
            route = m[path]
        }
        // 2. Dynamic.
        if route == nil {
            if tree := r.dynamicTrees[idx]; tree != nil {
                if r2, ok := tree.lookup(path, &ctx.params); ok {
                    route = r2
                }
            }
        }
    }

    if route != nil {
        ctx.matchedRoute = route
        ctx.matchedPath = path
        ctx.SetHandlers(route.finalChain)
        ctx.Next()
    } else {
        // Fallback "/*" route.
        if r.handleFallbackRoute && idx >= 0 {
            if m := r.staticRoutes[idx]; m != nil {
                if fb, ok := m["/*"]; ok {
                    ctx.SetHandlers(fb.finalChain)
                    ctx.Next()
                    goto end
                }
            }
        }
        // 405 detection.
        if r.handleMethodNotAllowed {
            allowed := r.findAllowedMethods(method, path)
            if len(allowed) > 0 {
                if len(r.noAllowed) == 0 {
                    r.noAllowed = HandlersChain{internal405Handler}
                }
                ctx.Set(CTXAllowedMethods, allowed)
                ctx.SetHandlers(r.noAllowed)
                ctx.Next()
                goto end
            }
        }
        // 404.
        if len(r.noRoute) == 0 {
            r.noRoute = HandlersChain{internal404Handler}
        }
        ctx.SetHandlers(r.noRoute)
        ctx.Next()
    }

end:
    if r.OnError != nil && len(ctx.Errors) > 0 {
        r.OnError(ctx)
    }
    // Context.Init replaced ctx.Resp with &c.writer, so this assertion succeeds.
    ctx.writer.ensureWriteHeader()
}

func (r *Router) findAllowedMethods(method, path string) []string {
    var allowed []string
    for _, m := range allMethods {
        if m == method {
            continue
        }
        idx := methodIndex(m)
        if idx < 0 {
            continue
        }
        if r.staticRoutes[idx] != nil {
            if _, ok := r.staticRoutes[idx][path]; ok {
                allowed = append(allowed, m)
                continue
            }
        }
        if tree := r.dynamicTrees[idx]; tree != nil {
            var ps Params
            if _, ok := tree.lookup(path, &ps); ok {
                allowed = append(allowed, m)
            }
        }
    }
    return allowed
}
```

`ensureWriteHeader` already exists in the legacy `response_wirter.go`
(renamed in Task 4.5) — no additional method needed.

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestServeHTTP -v
```

Expected: all 5 PASS.

- [ ] **Step 5: Commit**

```bash
git add dispatch.go response_writer.go dispatch_test.go
git commit -m "feat(v2): lock-free ServeHTTP hot path with HEAD->GET via mirror (P-5, P-9)"
```

---

## Phase 5: Context render & binding (1 day)

### Task 5.1: Adapt context_render.go

**Files:**
- Modify: `context_render.go`

- [ ] **Step 1: Inspect existing methods**

```bash
grep -n '^func (c \*Context)' context_render.go
```

This lists JSON/Text/HTML/File/Redirect/etc. methods.

- [ ] **Step 2: Verify no v1-only field references remain**

```bash
grep -n 'c\.engine\|c\.handlers\.last\|c\.params\[' context_render.go
```

Expected: empty — current `context_render.go` already accesses only `c.Req`, `c.Resp`, `c.SetHeader`, etc.

- [ ] **Step 3: Run go build to surface errors**

```bash
go build .
```

Expected: success. If a method references `c.Renderer` or removed v1 fields, fix or remove.

- [ ] **Step 4: Add a smoke test**

Append to `context_test.go`:

```go
func TestContext_JSON_WritesContentType(t *testing.T) {
    r := New()
    r.GET("/x", func(c *Context) {
        _ = c.JSON(200, map[string]string{"k": "v"})
    })
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/x", nil)
    r.ServeHTTP(w, req)
    assert.Eq(t, 200, w.Code)
    assert.True(t, strings.Contains(w.Header().Get("Content-Type"), "json"))
}

func TestContext_Text(t *testing.T) {
    r := New()
    r.GET("/x", func(c *Context) { c.Text(200, "hello") })
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/x", nil)
    r.ServeHTTP(w, req)
    body, _ := io.ReadAll(w.Body)
    assert.Eq(t, "hello", string(body))
}
```

Add the necessary import (`strings`, `io`) — likely already present.

- [ ] **Step 5: Run tests and commit**

```bash
go test -run "TestContext_JSON|TestContext_Text" -v
git add context_render.go context_test.go
git commit -m "chore(v2): adapt context_render.go to v2 Context"
```

---

### Task 5.2: Adapt context_binding.go

**Files:**
- Modify: `context_binding.go`

- [ ] **Step 1: Inspect**

```bash
grep -n '^func (c \*Context)' context_binding.go
head -20 context_binding.go
```

- [ ] **Step 2: Build and fix references**

```bash
go build .
```

Expected: success. If failures, fix one-by-one — most likely `c.params.Get(name)` works since v2 Params has `Get` method matching v1 interface.

- [ ] **Step 3: Add a smoke test**

Append to `context_test.go`:

```go
func TestContext_BindForm(t *testing.T) {
    type form struct {
        Name string `form:"name"`
    }
    r := New()
    r.POST("/x", func(c *Context) {
        var f form
        if err := c.Bind(&f); err != nil {
            c.AbortWithStatus(400)
            return
        }
        c.Text(200, "name="+f.Name)
    })
    w := httptest.NewRecorder()
    req := httptest.NewRequest("POST", "/x", strings.NewReader("name=alice"))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    r.ServeHTTP(w, req)
    body, _ := io.ReadAll(w.Body)
    assert.Eq(t, "name=alice", string(body))
}
```

- [ ] **Step 4: Run and verify**

```bash
go test -run TestContext_BindForm -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add context_binding.go context_test.go
git commit -m "chore(v2): adapt context_binding.go to v2 Context"
```

---

### Task 5.3: Adapt middleware.go and extends.go

**Files:**
- Modify: `middleware.go`
- Modify: `extends.go`

- [ ] **Step 1: Build to surface errors**

```bash
go build .
```

- [ ] **Step 2: Fix references**

If `middleware.go` references removed v1 fields (e.g. `route.handler`), update to use `route.Handler()` or `route.chain`.

`extends.go` should already be fine — its `BuildRequestURL` operates on path strings, not Route internals. Confirm by grepping:

```bash
grep -n 'route\.\|r\.Route' extends.go
```

- [ ] **Step 3: Smoke test middleware**

Append to `dispatch_test.go`:

```go
func TestMiddlewareOrder_GlobalGroupRouteMain(t *testing.T) {
    var order []string
    r := New()
    r.Use(func(c *Context) { order = append(order, "global"); c.Next() })
    r.Group("/api", func() {
        r.GET("/x", func(c *Context) { order = append(order, "main") },
            func(c *Context) { order = append(order, "route"); c.Next() })
    }, func(c *Context) { order = append(order, "group"); c.Next() })

    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/x", nil)
    r.ServeHTTP(w, req)
    assert.Eq(t, []string{"global", "group", "route", "main"}, order)
}
```

- [ ] **Step 4: Run and verify**

```bash
go test -run TestMiddlewareOrder -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add middleware.go extends.go dispatch_test.go
git commit -m "chore(v2): adapt middleware/extends; verify chain order"
```

---

## Phase 6: Cleanup (0.5 day)

### Task 6.1: Delete fastrux subpackage (P-13)

**Files:**
- Delete: `fastrux/`

- [ ] **Step 1: Verify the snapshot exists**

```bash
ls -lh _archive/
```

Expected: at least one `fastrux-snapshot-*.tar.gz`.

- [ ] **Step 2: Verify nothing in main package imports fastrux**

```bash
grep -r 'gookit/rux/fastrux' . --include='*.go'
```

Expected: empty.

- [ ] **Step 3: Delete the directory**

```bash
git rm -rf fastrux/
git status
```

Expected: many deletions staged.

- [ ] **Step 4: Build and test**

```bash
go build ./...
go test .
```

Expected: success.

- [ ] **Step 5: Commit**

```bash
git commit -m "chore(v2): remove fastrux subpackage — merged into main (P-13)"
```

---

### Task 6.2: Delete legacy radix_tree files

**Files:**
- Delete: `radix_tree.go`, `radix_tree_test.go`

- [ ] **Step 1: Confirm tree.go covers everything**

```bash
diff <(grep '^func ' radix_tree.go | sort) <(grep '^func ' tree.go | sort) || true
```

Look at any functions in `radix_tree.go` not in `tree.go` and ensure they were intentionally removed (not just forgotten).

- [ ] **Step 2: Delete**

```bash
git rm radix_tree.go radix_tree_test.go
```

- [ ] **Step 3: Build and test**

```bash
go build ./...
go test ./...
```

Expected: success.

- [ ] **Step 4: Commit**

```bash
git commit -m "chore(v2): remove legacy radix_tree.go (replaced by tree.go)"
```

- [ ] **Step 5: (No additional step)**

---

### Task 6.3: Archive _tmp/ and mark legacy design docs

**Files:**
- Delete: `_tmp/`
- Modify: `docs/design/design-refactor-core.md`, `docs/design/design-implementation.md`

- [ ] **Step 1: Move _tmp dev-progress to archive**

```bash
mkdir -p docs/design/archive
git mv _tmp/dev-progress.md docs/design/archive/dev-progress-fea_v2.md
git mv _tmp/HYBRID_ARCHITECTURE_PLAN.md docs/design/archive/HYBRID_ARCHITECTURE_PLAN.md
rm -rf _tmp/
git status
```

Expected: `_tmp/` is gone, two files appear in `docs/design/archive/`.

- [ ] **Step 2: Prepend ARCHIVED notice to legacy design docs**

For `docs/design/design-refactor-core.md` and `docs/design/design-implementation.md`, prepend the following block at the very top of each file:

```markdown
> **⚠️ ARCHIVED (2026-05-16):** This document captures the design exploration that
> preceded `rux-v2-design.md`. Implementation diverged from this plan.
> Refer to **[rux-v2-design.md](./rux-v2-design.md)** for the authoritative v2 spec.

```

- [ ] **Step 3: Verify markdown lints if any**

```bash
go test ./...
```

Expected: still passing (docs do not affect tests).

- [ ] **Step 4: Commit**

```bash
git add docs/design/ _tmp/
git commit -m "docs(v2): archive _tmp progress; mark legacy design docs ARCHIVED"
```

- [ ] **Step 5: (no extra step)**

---

## Phase 7: Tests + benchmarks (1.5 days)

### Task 7.1: Race detector on full suite

**Files:** none

- [ ] **Step 1: Run with race detector**

```bash
go test -race -count=1 ./...
```

Expected: PASS with no DATA RACE warnings.

- [ ] **Step 2: If any race appears, capture and fix**

Common culprit: forgetting that `r.Freeze()` must be called before any goroutine touches `staticRoutes`/`dynamicTrees`. The lazy freeze in `ServeHTTP` is `CompareAndSwap`-protected so it's safe — but if a test holds a Router and calls `Match` or `ServeHTTP` from goroutines without calling Freeze first, there's a window. Add an explicit call in the test fixture:

```go
r.Freeze()
```

- [ ] **Step 3: Re-run**

```bash
go test -race -count=3 ./...
```

Expected: stable PASS.

- [ ] **Step 4: Commit any fixes**

```bash
git add .
git commit -m "test(v2): pass -race -count=3"
```

- [ ] **Step 5: (no extra step)**

---

### Task 7.2: Reusable benchmark harness

**Files:**
- Modify: `benchmark_test.go` (replacing existing)

- [ ] **Step 1: Replace `benchmark_test.go`**

Replace `benchmark_test.go` with:

```go
package rux

import (
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
)

var noopHandler HandlerFunc = func(c *Context) {}

func runReq(b *testing.B, r *Router, method, path string) {
    req := httptest.NewRequest(method, path, nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req) // warm up + freeze
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        r.ServeHTTP(w, req)
    }
}

func BenchmarkV2_StaticRoute(b *testing.B) {
    r := New()
    r.GET("/users", noopHandler)
    runReq(b, r, "GET", "/users")
}

func BenchmarkV2_StaticRoute_404(b *testing.B) {
    r := New()
    r.GET("/users", noopHandler)
    runReq(b, r, "GET", "/missing")
}

func BenchmarkV2_Param1(b *testing.B) {
    r := New()
    r.GET("/users/{id}", noopHandler)
    runReq(b, r, "GET", "/users/42")
}

func BenchmarkV2_Param5(b *testing.B) {
    r := New()
    r.GET("/a/{a}/b/{b}/c/{c}/d/{d}/e/{e}", noopHandler)
    runReq(b, r, "GET", "/a/1/b/2/c/3/d/4/e/5")
}

func BenchmarkV2_Wildcard(b *testing.B) {
    r := New()
    r.GET("/files/*path", noopHandler)
    runReq(b, r, "GET", "/files/a/b/c/d.txt")
}

// 200-route table modeled after the GitHub API.
func BenchmarkV2_GithubAPI(b *testing.B) {
    r := newGithubAPIRouter()
    runReq(b, r, "GET", "/repos/gookit/rux/issues/1")
}

func newGithubAPIRouter() *Router {
    r := New()
    paths := []struct{ m, p string }{
        {"GET", "/users/{user}"},
        {"GET", "/users/{user}/repos"},
        {"GET", "/repos/{owner}/{repo}"},
        {"GET", "/repos/{owner}/{repo}/issues"},
        {"GET", "/repos/{owner}/{repo}/issues/{number}"},
        {"GET", "/repos/{owner}/{repo}/pulls"},
        {"GET", "/repos/{owner}/{repo}/pulls/{number}"},
        {"GET", "/repos/{owner}/{repo}/contributors"},
        {"GET", "/repos/{owner}/{repo}/forks"},
        {"GET", "/repos/{owner}/{repo}/stargazers"},
        // ... add 30 more for realism
    }
    for _, p := range paths {
        r.Add(p.p, noopHandler, p.m)
    }
    return r
}

func BenchmarkV2_Parallel_Static(b *testing.B) {
    r := New()
    r.GET("/users", noopHandler)
    req := httptest.NewRequest("GET", "/users", nil)
    r.ServeHTTP(httptest.NewRecorder(), req) // freeze
    b.ReportAllocs()
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        w := httptest.NewRecorder()
        for pb.Next() {
            r.ServeHTTP(w, req)
        }
    })
}

func BenchmarkV2_Parallel_Param(b *testing.B) {
    r := New()
    r.GET("/users/{id}", noopHandler)
    req := httptest.NewRequest("GET", "/users/42", nil)
    r.ServeHTTP(httptest.NewRecorder(), req)
    b.ReportAllocs()
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        w := httptest.NewRecorder()
        for pb.Next() {
            r.ServeHTTP(w, req)
        }
    })
}

// Suppress "unused" errors for http import on older Go — fmt and http used above.
var _ = http.StatusOK
var _ = fmt.Sprintf
```

- [ ] **Step 2: Run benchmarks**

```bash
go test -bench=BenchmarkV2 -benchmem -run=^$ -count=3 .
```

Expected: results table. Capture into a results file:

```bash
go test -bench=BenchmarkV2 -benchmem -run=^$ -count=3 . | tee _benchmarks/v2-results.txt
```

- [ ] **Step 3: Verify alloc targets**

Read `_benchmarks/v2-results.txt`. Expected per the spec §7.1:
- `BenchmarkV2_StaticRoute`: 0 allocs/op
- `BenchmarkV2_StaticRoute_404`: 0 allocs/op
- `BenchmarkV2_Param1`: 0 allocs/op (or 1 due to httptest overhead — investigate if higher)
- `BenchmarkV2_Param5`: 0–1 allocs/op

If allocs exceed targets, profile with `-cpuprofile` and `-memprofile` before declaring victory.

- [ ] **Step 4: Commit results file**

```bash
git add benchmark_test.go _benchmarks/v2-results.txt
git commit -m "test(v2): benchmark harness + initial results captured"
```

- [ ] **Step 5: (no extra step)**

---

### Task 7.3: Comparative benchmarks vs httprouter / gin

**Files:**
- Modify: `_benchmarks/rux/main.go` (or create `_benchmarks/rux2/`)

- [ ] **Step 1: Inspect existing benchmark layout**

```bash
ls _benchmarks/
cat _benchmarks/rux/main.go | head -40 2>/dev/null || true
```

- [ ] **Step 2: Create rux2 sub-benchmark**

```bash
mkdir -p _benchmarks/rux2
```

Create `_benchmarks/rux2/main.go` mirroring the existing `rux/main.go` but importing the new v2 main package and using v2 API (no `MatchResult`, etc.).

- [ ] **Step 3: Run cross-router benchmarks**

```bash
cd _benchmarks
go test -bench=. -benchmem -run=^$ -count=3 ./... | tee results-v2-cross.txt
```

Expected: comparison table across routers.

- [ ] **Step 4: Document results**

Create `_benchmarks/V2-COMPARISON.md` summarizing the comparison. Use the format of `PERFORMANCE.md` as a template.

- [ ] **Step 5: Commit**

```bash
git add _benchmarks/
git commit -m "test(v2): cross-router benchmark comparison vs httprouter, gin, etc."
```

---

## Phase 8: Documentation (1 day)

### Task 8.1: Rewrite README files

**Files:**
- Modify: `README.md`, `README.zh-CN.md`

- [ ] **Step 1: Identify v1-specific examples to replace**

```bash
grep -n 'r\.Match\|MatchResult\|EnableCaching' README.md
```

- [ ] **Step 2: Update Quick Start section**

Open `README.md` and `README.zh-CN.md`. Replace the Quick Start example so it:
- Uses `r.GET(...)` style (unchanged)
- Removes references to `MatchResult` / `EnableCaching` / regex params
- Adds a note: "Routes become read-only after first ServeHTTP / Freeze call."

- [ ] **Step 3: Add Performance section**

Add a "Performance" section linking to `_benchmarks/V2-COMPARISON.md`.

- [ ] **Step 4: Verify links**

```bash
grep -n '\](.*\.md)' README.md
```

Manually click-check or use a markdown link checker.

- [ ] **Step 5: Commit**

```bash
git add README.md README.zh-CN.md
git commit -m "docs(v2): rewrite README for v2 API"
```

---

### Task 8.2: Migration guide

**Files:**
- Create: `docs/MIGRATION-v1-to-v2.md`

- [ ] **Step 1: Create the file**

Create `docs/MIGRATION-v1-to-v2.md` with:

```markdown
# Migrating from rux v1 to v2

## TL;DR

v2 is a clean-room rewrite focused on extreme performance. The high-level
API surface (Router, Group, Resource, Controller, GET/POST/...) is largely
unchanged, but several v1 features have been removed or changed shape.

If your app uses only basic routing with optional params, migration is
essentially zero-touch (change the import path is enough — even that may
not be needed since module path is preserved).

## Breaking changes

### 1. `MatchResult` removed → `Match` returns `(*Route, []Param, bool)`

```go
// v1
m := r.Match("GET", "/users/42")
if m.Route != nil {
    id := m.Params["id"]
}

// v2
route, params, ok := r.Match("GET", "/users/42")
if ok {
    var id string
    for _, p := range params {
        if p.Key == "id" { id = p.Value; break }
    }
}
```

### 2. Regex params `{id:\d+}` removed (P-14)

Use a validation middleware:

```go
// v1
r.GET("/users/{id:\\d+}", showUser)

// v2 — option A
r.GET("/users/{id}", showUser, validate.IntParam("id"))

// v2 — option B
r.GET("/users/{id}", func(c *rux.Context) {
    id := c.Params().Int("id")
    if id <= 0 { c.AbortWithStatus(400); return }
    showUser(c, id)
})
```

### 3. Route.handler / Route.handlers split unified to Route.chain

Most users never accessed these directly — no action needed. If you used
`route.Handler()` or `route.Handlers()`, both still work but their
implementations changed (handler is now last element of chain).

### 4. `Use()` must precede route registration (Q6)

```go
// v1 — worked
r.GET("/x", h)
r.Use(mw) // applies retroactively

// v2 — panics
r.GET("/x", h)
r.Use(mw) // panic: rux: Use must be called before any route registration
```

Move all `Use()` calls to the top of your setup.

### 5. Routes become read-only after first request

After the first `ServeHTTP` call (or explicit `r.Freeze()`), any
`r.Add/GET/POST/Group/Use` panics. If you have hot-reload / plugin systems
that registered routes at runtime, build a new Router and atomic-swap
under your own lock.

### 6. `MaxParams = 16`

Routes with more than 16 path parameters panic at registration time.
This is a 4x safety margin over real-world APIs (GitHub maxes at 6-7).
Refactor or contact maintainer if you hit this.

### 7. `EnableCaching` / `MaxNumCaches` removed

Radix Tree lookup is fast enough; the LRU cache is gone.

### 8. `fastrux` subpackage deleted

If you imported `github.com/gookit/rux/fastrux`, switch to the main
`github.com/gookit/rux` package — it now ships fastrux's performance.

## Performance

See [_benchmarks/V2-COMPARISON.md](../_benchmarks/V2-COMPARISON.md).
```

- [ ] **Step 2: Verify**

```bash
ls -l docs/MIGRATION-v1-to-v2.md
```

- [ ] **Step 3: Cross-link from README**

Add a "Migrating from v1" section in `README.md` linking to this file.

- [ ] **Step 4: Run all tests one last time**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add docs/MIGRATION-v1-to-v2.md README.md README.zh-CN.md
git commit -m "docs(v2): migration guide v1 -> v2"
```

---

### Task 8.3: CHANGELOG entry

**Files:**
- Create or modify: `CHANGELOG.md`

- [ ] **Step 1: Check for existing CHANGELOG**

```bash
ls CHANGELOG.md 2>/dev/null || echo "missing"
```

- [ ] **Step 2: Create CHANGELOG.md if missing, or prepend a v2.0.0 entry**

If creating fresh:

```markdown
# Changelog

## v2.0.0 — 2026-05-16 (Breaking Changes)

This is a clean-room rewrite focused on extreme performance.
See [docs/MIGRATION-v1-to-v2.md](docs/MIGRATION-v1-to-v2.md) for the
breaking-change list.

### Added

- Per-method radix tree (`[9]*radixTree`), per-method static map (`[9]map[...]`)
- Inline `Params [16]Param` in Context for zero-allocation parameter passing
- `Router.Freeze()` for explicit read-only mode (auto-triggered on first ServeHTTP)
- Lock-free hot path (no mutex on serving)
- HEAD requests automatically mirror GET routes at freeze time
- Pre-merged middleware chains (no per-request append)

### Changed

- `Match` returns `(*Route, []Param, bool)` instead of `*MatchResult`
- `Route.handler` + `Route.handlers` unified into `Route.chain`
- `Use()` must be called before any route registration (panics otherwise)
- Static routes stored in `[9]map[path]*Route` (no string concat per request)

### Removed

- Regex parameter support `{id:\d+}` (use validation middleware instead)
- `MatchResult` / `QuickMatch` / `ReleaseMatchResult`
- `EnableCaching` / `MaxNumCaches` LRU cache
- `fastrux/` subpackage (functionality merged into main package)

### Performance (vs v1.x)

- Static route: ~30M ops/s, 0 alloc (was ~5M ops/s, multiple allocs)
- Single param dynamic: ~15M ops/s, 0 alloc (was <5M ops/s)
- 5 params dynamic: ~10M ops/s, 0 alloc

See `_benchmarks/V2-COMPARISON.md` for measured numbers.
```

- [ ] **Step 3: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs(v2): CHANGELOG v2.0.0 entry"
```

- [ ] **Step 4: Final state check**

```bash
git log --oneline | head -50
go test ./...
go test -bench=BenchmarkV2 -benchmem -run=^$ -count=1 . | head -20
```

Expected: clean log, all tests pass, benchmarks reproducible.

- [ ] **Step 5: Tag v2 release candidate**

```bash
git tag -a v2.0.0-rc1 -m "v2.0.0 release candidate 1"
git tag -l | tail -5
```

Expected: `v2.0.0-rc1` listed.

---

## Self-Review Checklist (post-implementation)

After all phases complete, run this final check:

- [ ] All 16 P-x problems from the spec have a corresponding fix landed
  - P-1: methodIndex + per-method arrays + single chain on node ✓ (Tasks 1.1, 1.3, 3.1)
  - P-2: static>param>wildcard priority in lookup ✓ (Task 2.2)
  - P-3: inline Params, no pool needed ✓ (Task 1.2)
  - P-4: `[16]Param` array ✓ (Task 1.2)
  - P-5: per-method static map, no concat ✓ (Tasks 3.1, 4.6)
  - P-6: Freeze pre-merges chain ✓ (Task 4.1)
  - P-7: indices+children arrays ✓ (Task 1.3)
  - P-8: atomic.Bool freeze, lock-free hot path ✓ (Tasks 4.1, 4.6)
  - P-9: HEAD mirroring on freeze ✓ (Task 4.2)
  - P-10: normalize once at registration ✓ (Tasks 1.5, 3.2)
  - P-11: single chain + route fields ✓ (Task 1.3)
  - P-12: Match returns ([]Param, bool) ✓ (Task 4.3)
  - P-13: fastrux deleted ✓ (Task 6.1)
  - P-14: regex removed, migration documented ✓ (Tasks 8.2, 2.4)
  - P-15: typed Context fields, lazy data map ✓ (Task 4.4)
  - P-16: priority bumping + child sort ✓ (Task 2.3)
- [ ] `go test -race -count=3 ./...` passes
- [ ] Benchmarks meet targets in spec §7.1
- [ ] Migration guide tested by following it manually on a small v1 app
