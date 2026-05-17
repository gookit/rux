# Rux

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gookit/rux?style=flat-square)
[![Actions Status](https://github.com/gookit/rux/workflows/Unit-Tests/badge.svg)](https://github.com/gookit/rux/actions)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/gookit/rux)](https://github.com/gookit/rux)
[![GoDoc](https://pkg.go.dev/badge/github.com/gookit/rux.svg)](https://pkg.go.dev/github.com/gookit/rux?tab=doc)
[![Coverage Status](https://coveralls.io/repos/github/gookit/rux/badge.svg?branch=master)](https://coveralls.io/github/gookit/rux?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/rux)](https://goreportcard.com/report/github.com/gookit/rux)

Simple and fast web framework for build golang HTTP applications.

> [中文说明](README.zh-CN.md)

## v2 Highlights

`rux` v2 is a clean-room rewrite focused on extreme performance:

- High-performance Radix Tree routing — per-method tree, lock-free hot path
- Zero-allocation static routes (one map lookup per request)
- Inline `Params [16]Param` in `Context` for low-allocation dynamic routing
- Auto-freeze on first `ServeHTTP` — the routing tables become read-only at runtime
- Auto HEAD → GET mirror at freeze time (no manual `r.HEAD` boilerplate)
- Pre-merged middleware chains (no per-request `append`)
- Same high-level API as v1 — `Router`, `Group`, `Resource`, `Controller`,
  `GET/POST/...` all unchanged.

See `_benchmarks/v2-results.txt` for measured numbers, and
[docs/MIGRATION-v1-to-v2.md](docs/MIGRATION-v1-to-v2.md) for breaking changes.

## Features

- Fast route match, support route group
- Support route path params and named routing
- Support route middleware, group middleware, global middleware
- Support quickly add a `RESETFul` or `Controller` style structs
- Support generic `http.Handler` interface middleware
- Support static file access handle
- Support add handlers for handle `NotFound` and `NotAllowed`

## GoDoc

- [godoc for github](https://pkg.go.dev/github.com/gookit/rux?tab=doc)

## Install

```bash
go get github.com/gookit/rux/v2
```

## Quick start

```go
package main

import (
	"fmt"

	"github.com/gookit/rux/v2"
)

func main() {
	r := rux.New()

	// Add Routes:
	r.GET("/", func(c *rux.Context) {
		c.Text(200, "hello")
	})
	r.GET("/hello/{name}", func(c *rux.Context) {
		c.Text(200, "hello "+c.Param("name"))
	})
	r.POST("/post", func(c *rux.Context) {
		c.Text(200, "hello")
	})
	// add multi method support for a route path
	r.Add("/post[/{id}]", func(c *rux.Context) {
		if c.Param("id") == "" {
			// do create post
			c.Text(200, "created")
			return
		}

		id := c.Params().Int("id")
		// do update post
		c.Text(200, "updated "+fmt.Sprint(id))
	}, rux.POST, rux.PUT)

	// Start server
	r.Listen(":8080")
	// can also
	// http.ListenAndServe(":8080", r)
}
```

## Route Group

```go
r.Group("/articles", func() {
    r.GET("", func(c *rux.Context) {
        c.Text(200, "view list")
    })
    r.POST("", func(c *rux.Context) {
        c.Text(200, "create ok")
    })
    r.GET(`/{id}`, func(c *rux.Context) {
        c.Text(200, "view detail, id: "+c.Param("id"))
    })
})
```

## Path Params

In v2 the path-param syntax is `{name}` (named) or `*name` (wildcard).
Regex constraints such as `{id:\d+}` are no longer supported — validate
inside the handler or with a small middleware (see the migration guide).

```go
// can access by: "/blog/123"
r.GET(`/blog/{id}`, func(c *rux.Context) {
    id := c.Params().Int("id")
    if id <= 0 {
        c.AbortWithStatus(400)
        return
    }
    c.Text(200, fmt.Sprintf("view detail, id: %d", id))
})
```

optional params, like `/about[.html]` or `/posts[/{id}]`:

```go
// can access by: "/blog/my-article" or "/blog/my-article.html"
r.GET(`/blog/{title}[.html]`, func(c *rux.Context) {
    c.Text(200, "view detail, title: "+c.Param("title"))
})

r.Add("/posts[/{id}]", func(c *rux.Context) {
    if c.Param("id") == "" {
        // do create post
        c.Text(200, "created")
        return
    }

    id := c.Params().Int("id")
    // do update post
    c.Text(200, "updated "+fmt.Sprint(id))
}, rux.POST, rux.PUT)
```

### Wildcards

Catch-all wildcards capture everything past the prefix:

```go
r.GET("/files/*path", func(c *rux.Context) {
    c.Text(200, "serve: "+c.Param("path"))
})
```

## Use Middleware

rux support use middleware, allow:

- global middleware
- group middleware
- route middleware

**Call priority**: `global middleware -> group middleware -> route middleware`

Examples:

```go
package main

import (
	"fmt"

	"github.com/gookit/rux/v2"
)

func main() {
	r := rux.New()

	// add global middleware
	r.Use(func(c *rux.Context) {
	    // do something ...
	})

	// add middleware for the route
	route := r.GET("/middle", func(c *rux.Context) { // main handler
		c.WriteString("-O-")
	}, func(c *rux.Context) { // middle 1
        c.WriteString("a")
        c.Next() // Notice: call Next()
        c.WriteString("A")
        // if call Abort(), will abort at the end of this middleware run
        // c.Abort()
    })

	// add more by Use()
	route.Use(func(c *rux.Context) { // middle 2
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})

	// now, access the URI /middle
	// will output: ab-O-BA
}
```

- **Call sequence**: `middle 1 -> middle 2 -> main handler -> middle 2 -> middle 1`
- **Flow chart**:

```text
        +-----------------------------+
        | middle 1                    |
        |  +----------------------+   |
        |  | middle 2             |   |
 start  |  |  +----------------+  |   | end
------->|  |  |  main handler  |  |   |--->----
        |  |  |________________|  |   |
        |  |______________________|   |
        |_____________________________|
```

> more please see [middleware_test.go](middleware_test.go) middleware tests

## Use http.Handler

rux is support generic `http.Handler` interface middleware

> You can use `rux.WrapHTTPHandler()` convert `http.Handler` as `rux.HandlerFunc`

```go
package main

import (
	"net/http"

	"github.com/gookit/rux/v2"
	// here we use gorilla/handlers, it provides some generic handlers.
	"github.com/gorilla/handlers"
)

func main() {
	r := rux.New()

	// create a simple generic http.Handler
	h0 := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("new-key", "val")
	})

	r.Use(rux.WrapHTTPHandler(h0), rux.WrapHTTPHandler(handlers.ProxyHeaders()))

	r.GET("/", func(c *rux.Context) {
		c.Text(200, "hello")
	})
	// add routes ...

    // Wrap our server with our gzip handler to gzip compress all responses.
    http.ListenAndServe(":8000", handlers.CompressHandler(r))
}
```

## More Usage

### Static Assets

```go
package main

import (
	"embed"
	"net/http"

	"github.com/gookit/rux/v2"
)

//go:embed static
var embAssets embed.FS

func main() {
	r := rux.New()

	// one file
	r.StaticFile("/site.js", "testdata/site.js")

	// allow any files in the directory.
	r.StaticDir("/static", "testdata")

	// file type limit in the directory
	r.StaticFiles("/assets", "testdata", "css|js")

	// go 1.16+: use embed assets. access: /embed/static/some.html
	r.StaticFS("/embed", http.FS(embAssets))
}
```

### Name Route

In `rux`, you can add a named route, and you can get the corresponding route instance(`rux.Route`) from the router according to the name.

Examples：

```go
	r := rux.New()

	// Method 1
	myRoute := rux.NewNamedRoute("name1", "/path4/some/{id}", emptyHandler, "GET")
	r.AddRoute(myRoute)

	// Method 2
	rux.AddNamed("name2", "/", func(c *rux.Context) {
		c.Text(200, "hello")
	})

	// Method 3
	r.GET("/hi", func(c *rux.Context) {
		c.Text(200, "hello")
	}).NamedTo("name3", r)

	// get route by name
	myRoute = r.GetRoute("name1")
```

### Redirect

redirect to other page

```go
r.GET("/", func(c *rux.Context) {
    c.AbortThen().Redirect("/login", 302)
})

// Or
r.GET("/", func(c *rux.Context) {
    c.Redirect("/login", 302)
    c.Abort()
})

r.GET("/", func(c *rux.Context) {
    c.Back()
    c.Abort()
})
```

### Cookies

you can quick operate cookies by `FastSetCookie()` `DelCookie()`

> Note: You must set or delete cookies before writing BODY content

```go
r.GET("/setcookie", func(c *rux.Context) {
    c.FastSetCookie("rux_cookie2", "test-value2", 3600)
    c.SetCookie("rux_cookie", "test-value1", 3600, "/", c.Req.URL.Host, false, true)
	c.WriteString("hello, in " + c.URL().Path)
})

// FastSetCookie accepts optional func(*http.Cookie) callbacks to override
// the developer-friendly defaults — handy for HTTPS / SameSite.
r.GET("/setsecure", func(c *rux.Context) {
    c.FastSetCookie("session", "v", 3600, func(ck *http.Cookie) {
        ck.Secure = true
        ck.SameSite = http.SameSiteStrictMode
    })
})

r.GET("/delcookie", func(c *rux.Context) {
	val := ctx.Cookie("rux_cookie") // "test-value1"
	c.DelCookie("rux_cookie", "rux_cookie2")
})
```

### Multi Domains

> code is refer from `julienschmidt/httprouter`

```go
package main

import (
	"log"
	"net/http"

	"github.com/gookit/rux/v2"
)

type HostSwitch map[string]http.Handler

// Implement the ServeHTTP method on our new type
func (hs HostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if a http.Handler is registered for the given host.
	// If yes, use it to handle the request.
	if router := hs[r.Host]; router != nil {
		router.ServeHTTP(w, r)
	} else {
		// Handle host names for which no handler is registered
		http.Error(w, "Forbidden", 403) // Or Redirect?
	}
}

func main() {
	// Initialize a router as usual
	router := rux.New()
	router.GET("/", Index)
	router.GET("/hello/{name}", func(c *rux.Context) {})

	// Make a new HostSwitch and insert the router (our http handler)
	// for example.com and port 12345
	hs := make(HostSwitch)
	hs["example.com:12345"] = router

	// Use the HostSwitch to listen and serve on port 12345
	log.Fatal(http.ListenAndServe(":12345", hs))
}
```

### RESETFul Style

```go
package main

import (
	"log"
	"net/http"

	"github.com/gookit/rux/v2"
)

type Product struct {
}

// Uses middlewares [optional]
func (Product) Uses() map[string][]rux.HandlerFunc {
	return map[string][]rux.HandlerFunc{
		// function name: handlers
		"Delete": []rux.HandlerFunc{
			handlers.HTTPBasicAuth(map[string]string{"test": "123"}),
			handlers.GenRequestID(),
		},
	}
}

// all products [optional]
func (p *Product) Index(c *rux.Context) { }

// create product [optional]
func (p *Product) Create(c *rux.Context) { }

// save new product [optional]
func (p *Product) Store(c *rux.Context) { }

// show product with {id} [optional]
func (p *Product) Show(c *rux.Context) { }

// edit product [optional]
func (p *Product) Edit(c *rux.Context) { }

// save edited product [optional]
func (p *Product) Update(c *rux.Context) { }

// delete product [optional]
func (p *Product) Delete(c *rux.Context) { }

func main() {
	router := rux.New()

	// methods	Path	Action	Route Name
    // GET	/product	index	product_index
    // GET	/product/create	create	product_create
    // POST	/product	store	product_store
    // GET	/product/{id}	show	product_show
    // GET	/product/{id}/edit	edit	product_edit
    // PUT/PATCH	/product/{id}	update	product_update
    // DELETE	/product/{id}	delete	product_delete
    // resetful style
	router.Resource("/", new(Product))

	log.Fatal(http.ListenAndServe(":12345", router))
}
```

### Controller Style

```go
package main

import (
	"log"
	"net/http"

	"github.com/gookit/rux/v2"
)

// News controller
type News struct {
}

func (n *News) AddRoutes(g *rux.Router) {
	g.GET("/", n.Index)
	g.POST("/", n.Create)
	g.PUT("/", n.Edit)
}

func (n *News) Index(c *rux.Context) { }

func (n *News) Create(c *rux.Context) { }

func (n *News) Edit(c *rux.Context) { }

func main() {
	router := rux.New()

	// controller style
	router.Controller("/news", new(News))

	log.Fatal(http.ListenAndServe(":12345", router))
}
```

### Build URL

```go
package main

import (
	"log"
	"net/http"

	"github.com/gookit/rux/v2"
)

func main() {
	// Initialize a router as usual
	router := rux.New()
	router.GET(`/news/{category_id}/{new_id}/detail`, func(c *rux.Context) {
		var u = make(url.Values)
        u.Add("username", "admin")
        u.Add("password", "12345")

		b := rux.NewBuildRequestURL()
        // b.Scheme("https")
        // b.Host("www.mytest.com")
        b.Queries(u)
        b.Params(rux.M{"{category_id}": "100", "{new_id}": "20"})
		// b.Path("/dev")
        // println(b.Build().String())

        println(c.Router().BuildRequestURL("new_detail", b).String())
		// result:  /news/100/20/detail?username=admin&password=12345
		// get current route name
		if c.MustGet(rux.CTXCurrentRouteName) == "new_detail" {
            // post data etc....
        }
	}).NamedTo("new_detail", router)

	// Use the HostSwitch to listen and serve on port 12345
	log.Fatal(http.ListenAndServe(":12345", router))
}
```

## Production-Ready Server

Package `server` wraps a `rux.Router` with sensible HTTP timeouts,
graceful shutdown, lifecycle hooks, and built-in `/healthz` / `/readyz`
endpoints. It is the recommended way to run rux in containers / k8s.

```go
package main

import (
	"context"
	"log"

	"github.com/gookit/rux/v2"
	"github.com/gookit/rux/v2/server"
)

func main() {
	s := server.New(false) // false = no debug logging
	s.Addr = ":8080"

	s.GET("/", func(c *rux.Context) {
		c.Text(200, "hello")
	})

	// Optional liveness/readiness endpoints under /healthz and /readyz.
	s.MountHealthChecks()

	// Optional lifecycle hooks (warm caches, validate config, etc.).
	s.PreStart = append(s.PreStart, func(ctx context.Context) error {
		return nil
	})

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
```

What `Run()` does for you:

- `ListenAndServe` (or TLS variant when `TLSCertFile`/`TLSKeyFile` are set)
- Wait for `SIGINT` / `SIGTERM` (configurable via `StopSignals`)
- On signal: flip `/readyz` to 503 → wait `DrainDelay` so the upstream LB
  can drain → call `http.Server.Shutdown` bounded by `ShutdownTimeout`
- Run `PreShutdown` / `PostShutdown` hooks in order

Defaults tuned for container deployments:

| Field               | Default | Purpose                              |
| ------------------- | ------- | ------------------------------------ |
| `ReadHeaderTimeout` | 2s      | slowloris defense                    |
| `ReadTimeout`       | 10s     | full request read budget             |
| `WriteTimeout`      | 30s     | response write budget                |
| `IdleTimeout`       | 120s    | keep-alive idle close                |
| `DrainDelay`        | 5s      | LB drain window after stop signal    |
| `ShutdownTimeout`   | 25s     | bound on graceful shutdown           |

### Echo Server (httpbin-style)

`server.NewEchoServer()` builds a Server with httpbin-style debug
endpoints pre-mounted: `/anything`, `/get|post|put|patch|delete`,
`/status/{code}`, `/delay/{n}`, `/redirect/{n}`, `/cookies`,
`/basic-auth/{u}/{p}`, `/bytes/{n}`, `/uuid`, `/download/{filename}`,
`POST /upload`, and a `/*path` catch-all. Useful for local debugging,
integration tests, and as a `/debug` subtree inside larger apps via
`server.MountEchoRoutes(r)`.

```bash
go run ./_examples/echo-server
# then:
curl http://127.0.0.1:18080/anything
curl -F "file=@./README.md" http://127.0.0.1:18080/upload
```

See [docs/echo-server.md](docs/echo-server.md) for the full endpoint
table and usage recipes.

### Server-Sent Events

`pkg/sse` wraps the SSE wire format and lifecycle so handlers only
have to drive the producer. The Hooks struct exposes
`OnConnect` / `OnDisconnect` / `OnSend` / `OnError` callbacks for
auth, logging, filtering, and metrics — any field may be nil.

```go
import "github.com/gookit/rux/v2/pkg/sse"

s.GET("/events", func(c *rux.Context) {
    _ = sse.Stream(c, &sse.Hooks{
        OnConnect:    func(c *rux.Context) error { /* auth check */ return nil },
        OnDisconnect: func(c *rux.Context, reason error) { /* audit */ },
    }, func(send sse.SendFunc, done <-chan struct{}) error {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-done:
                return nil
            case t := <-ticker.C:
                if err := send(sse.Event{Data: t.Format(time.RFC3339)}); err != nil {
                    return err
                }
            }
        }
    })
})
```

`OnConnect` runs **before** the SSE headers are written, so a
rejecting hook can issue any 4xx via `c.Resp` (e.g.
`http.Error(c.Resp, "no token", 401)`).

`Stream` emits a leading `: connected\n\n` comment frame by default
(suppress with `StreamWith` and `SendConnected: false`). For
keepalives use `StreamWith` and set `KeepaliveInterval`:

```go
sse.StreamWith(c, &sse.Options{
    Hooks: myHooks,
    SendConnected: true,
    KeepaliveInterval: 30 * time.Second, // ": keepalive\n\n" every 30s
}, producer)
```

**Two different timeouts — both matter:**

| Timer                                    | Defeated by             |
| ---------------------------------------- | ----------------------- |
| `server.Server.WriteTimeout` (default 30s)        | Must set `= 0`. Heartbeats do NOT save you — this bounds the whole response lifetime. |
| Proxy / NAT idle timeout (nginx 60s, ALB 60s, …)  | `KeepaliveInterval` ≤ that value. |

**Keyed push with Hub.** For business-driven pushes (notify user X,
broadcast to all) use `sse.NewHub` — an in-memory registry keyed by
ID (e.g. user ID), multi-connection-per-id (multi-tab fan-out), with
non-blocking per-client buffer + dropped-event counter + `OnDrop`
hook:

```go
hub := sse.NewHub(64) // per-client buffer size

s.GET("/events", func(c *rux.Context) {
    uid := authUserID(c)
    _ = sse.Stream(c, nil, sse.HubProducer(hub, uid))
})

// elsewhere, business code:
delivered, dropped := hub.Send("user-42", sse.Event{Name: "notify", Data: "..."})
hub.Broadcast(sse.Event{Name: "announce", Data: "..."})
hub.SetOnDrop(func(c *sse.Client, _ sse.Event) {
    log.Printf("slow client %s, dropped=%d", c.ID, c.Dropped())
})
```

See `_examples/sse-server` for the full setup (subscribe + push +
broadcast + stats endpoints with a tiny HTML demo client).

## Migrating from v1

If you are upgrading from rux v1.x, please read
[docs/MIGRATION-v1-to-v2.md](docs/MIGRATION-v1-to-v2.md) for a complete
list of breaking changes. The high-level API surface is largely unchanged
and most basic applications need no source edits.

## Performance

rux v2 targets sub-200 ns/op for typical dynamic routes and 0 alloc/op
for static and most parametrized routes. See
[`_benchmarks/v2-results.txt`](_benchmarks/v2-results.txt) for the
benchmark numbers measured on the current branch.

## Help

- lint

```bash
golint ./...
```

- format check

```bash
# list error files
gofmt -s -l ./
# fix format and write to file
gofmt -s -w some.go
```

- unit test

```bash
go test -cover ./...
```

## Gookit Packages

- [gookit/ini](https://github.com/gookit/ini) Go config management, use INI files
- [gookit/rux](https://github.com/gookit/rux) Simple and fast request router for golang HTTP
- [gookit/gcli](https://github.com/gookit/gcli) build CLI application, tool library, running CLI commands
- [gookit/slog](https://github.com/gookit/slog) Concise and extensible go log library
- [gookit/event](https://github.com/gookit/event) Lightweight event manager and dispatcher implements by Go
- [gookit/cache](https://github.com/gookit/cache) Generic cache use and cache manager for golang. support File, Memory, Redis, Memcached.
- [gookit/config](https://github.com/gookit/config) Go config management. support JSON, YAML, TOML, INI, HCL, ENV and Flags
- [gookit/color](https://github.com/gookit/color) A command-line color library with true color support, universal API methods and Windows support
- [gookit/filter](https://github.com/gookit/filter) Provide filtering, sanitizing, and conversion of golang data
- [gookit/validate](https://github.com/gookit/validate) Use for data validation and filtering. support Map, Struct, Form data
- [gookit/goutil](https://github.com/gookit/goutil) Some utils for the Go: string, array/slice, map, format, cli, env, filesystem, test and more
- More please see https://github.com/gookit

## See also

- https://github.com/gin-gonic/gin
- https://github.com/gorilla/mux
- https://github.com/julienschmidt/httprouter
- https://github.com/xialeistudio/go-dispatcher

## License

**[MIT](LICENSE)**
