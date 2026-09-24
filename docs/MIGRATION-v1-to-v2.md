# Migrating from rux v1 to v2

## TL;DR

v2 is a clean-room rewrite focused on extreme performance. The high-level
API surface (Router, Group, Resource, Controller, GET/POST/...) is largely
unchanged, but several v1 features have been removed or changed shape.

If your app uses only basic routing with optional params, migration is
essentially zero-touch.

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
        if p.Key == "id" {
            id = p.Value
            break
        }
    }
}
```

### 2. Regex params `{id:\d+}` removed

Use a validation middleware:

```go
// v1
r.GET("/users/{id:\\d+}", showUser)

// v2 — option A: handler-internal validation
r.GET("/users/{id}", func(c *rux.Context) {
    id := c.Params().Int("id")
    if id <= 0 { c.AbortWithStatus(400); return }
    showUser(c)
})

// v2 — option B: built-in regex middleware (whole-value match, 400 on failure)
r.GET("/users/{id}", showUser, handlers.ParamRegex("id", `\d+`))

// v2 — option C: write your own when you need a custom response
func validateIntParam(name string) rux.HandlerFunc {
    return func(c *rux.Context) {
        if _, err := strconv.Atoi(c.Param(name)); err != nil {
            c.AbortWithStatus(400, "invalid "+name)
            return
        }
        c.Next()
    }
}
r.GET("/users/{id}", showUser, validateIntParam("id"))
```

`{file:.+}` and `{file:.*}` are still supported and become `*file` wildcards.

### 3. `Route.handler` / `Route.handlers` unified to `Route.chain`

Most users never accessed these directly — no action needed.
`route.Handler()` and `route.Handlers()` accessors continue to work
(handler is now last element of chain).

### 4. `Use()` ordering

Global middleware is merged into every route chain when the router freezes (on the
first request), so `Use` works at any point before that, including after routes were
registered — v1's retroactive behaviour is back:

```go
r.GET("/x", h)
r.Use(mw) // fine: mw applies to /x as well, and to anything registered later
```

Earlier v2 releases (v2.0.x – v2.1.0) panicked here with
`rux: Use must be called before any route registration (Q6)`; that restriction is
gone. Two things to keep in mind:

- once the first request has been served the router is frozen and `Use` panics with
  `rux: cannot Use after router is frozen`, so configure middleware during setup
- `Use` is always global: written inside a `Group` closure it applies to every
  route, not just that group. Pass middleware to `Group` (or `Group.Use` on a group
  value) when you want group scope.

### 5. Routes become read-only after first request

After the first `ServeHTTP` call (or explicit `r.Freeze()`), any
`r.Add/GET/POST/Group/Use/NotFound/NotAllowed` panics. Hot-reload systems should build a
new Router and atomic-swap externally.

`NotFound` / `NotAllowed` moved into this group because the 404/405 handler
chains are composed once at freeze time, with the global middleware chain in
front of them (see below).

### 5.1 404/405 responses run the global middleware chain

Unmatched paths and method mismatches used to bypass `r.Use(...)` entirely, so a
global auth / security-header / logging middleware was skipped for exactly the
requests an attacker controls. The global chain now runs first for `NotFound`
(404) and `NotAllowed` (405), like it does for matched routes. If a global auth
middleware rejects the request, the client gets that status (e.g. 401) instead of
404.

If you relied on the old behavior for a specific path, register a normal route
(including a `/*path` wildcard) and branch inside the handler: route handlers
were always inside the global chain.

### 6. `MaxParams = 16` cap

Routes with more than 16 path parameters panic at registration time.

### 7. `EnableCaching` / `MaxNumCaches` removed

Radix Tree lookup is fast enough that the LRU cache adds no value.

### 8. `fastrux` subpackage deleted

If you imported `github.com/gookit/rux/fastrux`, switch to the main
`github.com/gookit/rux` package — it now ships fastrux's performance.

## Performance

See `_benchmarks/v2-results.txt`.
