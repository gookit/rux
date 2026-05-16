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

// v2 — option B: validation middleware (write your own)
r.GET("/users/{id}", showUser, validateIntParam("id"))
```

`{file:.+}` and `{file:.*}` are still supported and become `*file` wildcards.

### 3. `Route.handler` / `Route.handlers` unified to `Route.chain`

Most users never accessed these directly — no action needed.
`route.Handler()` and `route.Handlers()` accessors continue to work
(handler is now last element of chain).

### 4. `Use()` must precede route registration

```go
// v1 — worked retroactively
r.GET("/x", h)
r.Use(mw)

// v2 — panics
r.GET("/x", h)
r.Use(mw) // panic: rux: Use must be called before any route registration
```

Move all `Use()` calls to the top of your setup.

### 5. Routes become read-only after first request

After the first `ServeHTTP` call (or explicit `r.Freeze()`), any
`r.Add/GET/POST/Group/Use` panics. Hot-reload systems should build a
new Router and atomic-swap externally.

### 6. `MaxParams = 16` cap

Routes with more than 16 path parameters panic at registration time.

### 7. `EnableCaching` / `MaxNumCaches` removed

Radix Tree lookup is fast enough that the LRU cache adds no value.

### 8. `fastrux` subpackage deleted

If you imported `github.com/gookit/rux/fastrux`, switch to the main
`github.com/gookit/rux` package — it now ships fastrux's performance.

## Performance

See `_benchmarks/v2-results.txt`.
