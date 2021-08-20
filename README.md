# Rux

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gookit/rux?style=flat-square)
[![Actions Status](https://github.com/gookit/rux/workflows/unit-tests/badge.svg)](https://github.com/gookit/rux/actions)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/gookit/rux)](https://github.com/gookit/rux)
[![GoDoc](https://godoc.org/github.com/gookit/rux?status.svg)](https://pkg.go.dev/github.com/gookit/rux?tab=doc)
[![Build Status](https://travis-ci.org/gookit/rux.svg?branch=master)](https://travis-ci.org/gookit/rux)
[![Coverage Status](https://coveralls.io/repos/github/gookit/rux/badge.svg?branch=master)](https://coveralls.io/github/gookit/rux?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/rux)](https://goreportcard.com/report/github.com/gookit/rux)

Simple and fast web framework for build golang HTTP applications.

> NOTICE: `v1.3.x` is not fully compatible with `v1.2.x` version

- Fast route match, support route group
- Support route path params and named routing
- Support cache recently accessed dynamic routes
- Support route middleware, group middleware, global middleware
- Support generic `http.Handler` interface middleware
- Support static file access handle
- Support add handlers for handle `NotFound` and `NotAllowed`

## [中文说明](README.zh-CN.md)

中文说明请看 **[README.zh-CN](README.zh-CN.md)**

## GoDoc

- [godoc for github](https://pkg.go.dev/github.com/gookit/rux?tab=doc)

## Install

```bash
go get github.com/gookit/rux
```

## Quick start

```go
package main

import (
	"github.com/gookit/rux"
)

func main() {
	r := rux.New()
	
	// Add Routes:
	r.GET("/", func(c *rux.Context) {
		c.Text(200, "hello")
	})
	r.GET("/hello/{name}", func(c *rux.Context) {
		c.Text(200, "hello " + c.Param("name"))
	})
	r.POST("/post", func(c *rux.Context) {
		c.Text(200, "hello")
	})
	// add multi method support for an route path
	r.Add("/post[/{id}]", func(c *rux.Context) {
		if c.Param("id") == "" {
			// do create post
			c.Text(200, "created")
			return
		}

		id := c.Params.Int("id")
		// do update post
		c.Text(200, "updated " + fmt.Sprint(id))
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
    r.GET(`/{id:\d+}`, func(c *rux.Context) {
        c.Text(200, "view detail, id: " + c.Param("id"))
    })
})
```

## Path Params

You can add the path params like: `{id}` Or `{id:\d+}`

```go
// can access by: "/blog/123"
r.GET(`/blog/{id:\d+}`, func(c *rux.Context) {
    c.Text(200, "view detail, id: " + c.Param("id"))
})
```

optional params, like `/about[.html]` or `/posts[/{id}]`:

```go
// can access by: "/blog/my-article" "/blog/my-article.html"
r.GET(`/blog/{title:\w+}[.html]`, func(c *rux.Context) {
    c.Text(200, "view detail, id: " + c.Param("id"))
})

r.Add("/posts[/{id}]", func(c *rux.Context) {
    if c.Param("id") == "" {
        // do create post
        c.Text(200, "created")
        return
    }

    id := c.Params.Int("id")
    // do update post
    c.Text(200, "updated " + fmt.Sprint(id))
}, rux.POST, rux.PUT)
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

	"github.com/gookit/rux"
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
	
	"github.com/gookit/rux"
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
	"github.com/gookit/rux"
)

func main() {
	r := rux.New()

	// one file
	r.StaticFile("/site.js", "testdata/site.js")

	// allow any files in the directory.
	r.StaticDir("/static", "testdata")

	// file type limit in the directory
	r.StaticFiles("/assets", "testdata", "css|js")
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

	"github.com/gookit/rux"
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

	"github.com/gookit/rux"
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
func (p *Product) Index(c *rux.Context) {
	// do something
}

// create product [optional]
func (p *Product) Create(c *rux.Context) {
	// do something
}

// save new product [optional]
func (p *Product) Store(c *rux.Context) {
	// do something
}

// show product with {id} [optional]
func (p *Product) Show(c *rux.Context) {
	// do something
}

// edit product [optional]
func (p *Product) Edit(c *rux.Context) {
	// do something
}

// save edited product [optional]
func (p *Product) Update(c *rux.Context) {
	// do something
}

// delete product [optional]
func (p *Product) Delete(c *rux.Context) {
	// do something
}

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

	"github.com/gookit/rux"
)

// News controller
type News struct {
}

func (n *News) AddRoutes(g *rux.Router) {
	g.GET("/", n.Index)
	g.POST("/", n.Create)
	g.PUT("/", n.Edit)
}

func (n *News) Index(c *rux.Context) {
	// Do something
}

func (n *News) Create(c *rux.Context) {
	// Do something
}

func (n *News) Edit(c *rux.Context) {
	// Do something
}

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

	"github.com/gookit/rux"
)

func main() {
	// Initialize a router as usual
	router := rux.New()
	router.GET(`/news/{category_id}/{new_id:\d+}/detail`, func(c *rux.Context) {
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
