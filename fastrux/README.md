# FastRux - High-Performance HTTP Router for Go

FastRux is a blazingly fast HTTP router built on **Radix Tree**, achieving **2-4x performance** improvements over regex-based routers with **zero to minimal allocations**.

## Features

✨ **High Performance**
- Static routes: ~115 ns/op, **1 allocation**
- Dynamic routes: ~304 ns/op, **3 allocations**
- 404 responses: ~108 ns/op, **0 allocations**
- Radix Tree O(m) lookup vs regex O(n)

🎯 **Zero Configuration**
- No cache configuration needed
- No LRU limits to set
- Works great out of the box

🔧 **Full Feature Set**
- All HTTP methods (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, TRACE, CONNECT)
- Named routes with URL building
- Route groups with middleware
- RESTful resource routing
- Controller-based routing
- Static file serving
- Wildcard routes
- Optional parameters
- Method not allowed handling
- Custom 404/405 handlers
- Panic recovery
- Context with full request/response utilities

## Quick Start

```go
package main

import (
    "github.com/gookit/rux/fastrux"
)

func main() {
    r := fastrux.New()

    r.GET("/", func(c *fastrux.Context) {
        c.Text(200, "Hello FastRux!")
    })

    r.GET("/users/{id}", func(c *fastrux.Context) {
        c.JSON(200, map[string]string{
            "id": c.Param("id"),
        })
    })

    r.Listen(":8080")
}
```

## Installation

```bash
go get github.com/gookit/rux/fastrux
```

## Usage Examples

### Basic Routes

```go
r := fastrux.New()

// Simple routes
r.GET("/ping", func(c *fastrux.Context) {
    c.Text(200, "pong")
})

r.POST("/users", func(c *fastrux.Context) {
    c.JSON(201, map[string]string{"status": "created"})
})

// Multiple methods
r.Add("/resource", handler, "GET", "POST")

// Any method
r.Any("/debug", debugHandler)
```

### Route Parameters

```go
// Single parameter
r.GET("/users/{id}", func(c *fastrux.Context) {
    id := c.Param("id")
    c.Text(200, "User: "+id)
})

// Multiple parameters
r.GET("/posts/{year}/{month}/{day}", func(c *fastrux.Context) {
    year := c.Param("year")
    month := c.Param("month")
    day := c.Param("day")
    // ...
})

// Wildcard (catches everything)
r.GET("/files/*filepath", func(c *fastrux.Context) {
    path := c.Param("filepath")
    c.File(path)
})
```

### Optional Parameters

```go
// Expands to two routes: /posts and /posts/{id}
r.GET("/posts[/{id}]", func(c *fastrux.Context) {
    if id := c.Param("id"); id != "" {
        c.Text(200, "Post: "+id)
    } else {
        c.Text(200, "All posts")
    }
})
```

### Route Groups

```go
r.Group("/api", func() {
    r.Group("/v1", func() {
        r.GET("/users", listUsers)
        r.POST("/users", createUser)
        r.GET("/users/{id}", getUser)
    })
}, authMiddleware, loggingMiddleware)
```

### Named Routes

```go
r.AddNamed("user.show", "/users/{id}", showUser, "GET")

// Build URL
url := r.BuildURL("user.show", fastrux.M{"id": "123"})
// => /users/123
```

### RESTful Resources

```go
type UserController struct{}

func (u *UserController) Index(c *fastrux.Context)  { /* GET /users */ }
func (u *UserController) Create(c *fastrux.Context) { /* GET /users/create */ }
func (u *UserController) Store(c *fastrux.Context)  { /* POST /users */ }
func (u *UserController) Show(c *fastrux.Context)   { /* GET /users/{id} */ }
func (u *UserController) Edit(c *fastrux.Context)   { /* GET /users/{id}/edit */ }
func (u *UserController) Update(c *fastrux.Context) { /* PUT/PATCH /users/{id} */ }
func (u *UserController) Delete(c *fastrux.Context) { /* DELETE /users/{id} */ }

// Registers all 7 RESTful routes
r.Resource("/", &UserController{})
```

### Middleware

```go
// Global middleware
r.Use(logger, recovery)

// Group middleware
r.Group("/admin", func() {
    r.GET("/dashboard", handler)
}, authMiddleware)

// Route-specific middleware
r.GET("/protected", handler, authMiddleware)
```

### Static Files

```go
// Single file
r.StaticFile("/favicon.ico", "./public/favicon.ico")

// Directory
r.StaticDir("/static", "./public")

// With file extension filter
r.StaticFiles("/assets", "./public", "css|js|png|jpg")

// Filesystem
r.StaticFS("/files", http.Dir("./uploads"))
```

### Context Utilities

```go
r.GET("/example", func(c *fastrux.Context) {
    // Request
    id := c.Param("id")
    name := c.Query("name", "default")
    token := c.Header("Authorization")

    // JSON binding
    var data struct {
        Email string `json:"email"`
    }
    c.BindJSON(&data)

    // Response
    c.JSON(200, map[string]any{
        "id":    id,
        "name":  name,
        "email": data.Email,
    })

    // Or other formats
    c.Text(200, "text")
    c.HTML(200, []byte("<h1>HTML</h1>"))
    c.XML(200, data)
    c.File("./file.pdf")
    c.Redirect("/other-path", 302)
})
```

### Error Handling

```go
r := fastrux.New()

// Custom 404
r.NotFound(func(c *fastrux.Context) {
    c.JSON(404, map[string]string{"error": "not found"})
})

// Custom 405 (requires HandleMethodNotAllowed option)
r = fastrux.New(fastrux.HandleMethodNotAllowed)
r.NotAllowed(func(c *fastrux.Context) {
    allowed := c.SafeGet(fastrux.CTXAllowedMethods).([]string)
    c.JSON(405, map[string]any{
        "error":   "method not allowed",
        "allowed": allowed,
    })
})

// Panic recovery
r.OnPanic = func(c *fastrux.Context) {
    err := c.SafeGet(fastrux.CTXRecoverResult)
    c.JSON(500, map[string]any{"error": "internal server error"})
}

// Global error handler
r.OnError = func(c *fastrux.Context) {
    for _, err := range c.Errors {
        log.Println(err)
    }
}
```

### Router Options

```go
r := fastrux.New(
    fastrux.StrictLastSlash,         // /path != /path/
    fastrux.HandleMethodNotAllowed,  // Return 405 for wrong methods
    fastrux.HandleFallbackRoute,     // Enable /* fallback
    fastrux.UseEncodedPath,          // Use URL.EscapedPath() instead of Path
    fastrux.InterceptAll("/maintenance"), // Redirect all to one path
)
```

## Performance Characteristics

### Complexity
- **Static routes**: O(1) map lookup
- **Dynamic routes**: O(m) where m = path length
- **Memory**: O(n × m) where n = routes, m = avg path length

### Benchmarks

```
BenchmarkServeHTTP_Static      114.8 ns/op    16 B/op     1 allocs/op
BenchmarkServeHTTP_Param1      303.8 ns/op   352 B/op     3 allocs/op
BenchmarkServeHTTP_Param5      377.2 ns/op   352 B/op     3 allocs/op
BenchmarkServeHTTP_404         107.6 ns/op     0 B/op     0 allocs/op

BenchmarkHighLoad_Static        36.7 ns/op    16 B/op     1 allocs/op (parallel)
BenchmarkHighLoad_Param        160.6 ns/op   352 B/op     3 allocs/op (parallel)
```

See [PERFORMANCE.md](./PERFORMANCE.md) for detailed analysis.

## Design Principles

### 1. No Regex Constraints
**Before (rux)**: `{id:\d+}` with regex validation
**After (fastrux)**: `{id}` - validation in middleware

```go
// Validation middleware
func validateID(c *fastrux.Context) {
    if id := c.Param("id"); !isNumeric(id) {
        c.AbortWithStatus(400, "invalid id")
        return
    }
    c.Next()
}

r.GET("/users/{id}", handler, validateID)
```

**Benefit**: 40-70% faster matching, cleaner separation of concerns

### 2. Optional Segments Expanded
**Before**: `[/{id}]` stored as one route with complex logic
**After**: Expanded to two routes at registration time

```go
r.GET("/posts[/{id}]", handler)
// Internally becomes:
// 1. /posts
// 2. /posts/:id
```

**Benefit**: Simpler tree, faster matching, no runtime overhead

### 3. No LRU Cache
**Before**: LRU cache to speed up regex matching
**After**: Radix Tree is fast enough, no cache needed

**Benefit**: Zero cache configuration, no memory overhead, simpler code

### 4. Lazy Allocation
**Before**: Always allocate Params map
**After**: Allocate only when route has parameters

**Benefit**: 0 allocations for 404s, 1 allocation for static routes

## Migration from rux

FastRux maintains API compatibility with rux v1.x:

```go
// Old (rux)
import "github.com/gookit/rux"
r := rux.New()

// New (fastrux)
import "github.com/gookit/rux/fastrux"
r := fastrux.New()
```

### Breaking Changes

1. **Regex params stripped**: `{id:\d+}` → `:id` (move validation to middleware)
2. **MatchResult API**: `Match()` returns `*MatchResult` instead of `(*Route, Params, []string)`
3. **No Renderer field**: Removed deprecated `router.Renderer`

### API Compatibility

✅ All HTTP method registration (GET, POST, etc.)
✅ Route groups
✅ Named routes
✅ RESTful resources
✅ Controllers
✅ Static file serving
✅ Middleware
✅ Context API
✅ NotFound/NotAllowed handlers

## Contributing

Contributions are welcome! Please:
1. Run tests: `go test ./fastrux/`
2. Run benchmarks: `go test ./fastrux/ -bench=. -benchmem`
3. Ensure no allocation regressions

## License

MIT License - same as rux

## Credits

Built on top of [gookit/rux](https://github.com/gookit/rux) with Radix Tree implementation inspired by best practices from httprouter, gin, and echo.
