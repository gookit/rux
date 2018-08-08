# sux router

[![GoDoc](https://godoc.org/github.com/gookit/sux?status.svg)](https://godoc.org/github.com/gookit/sux)
[![Build Status](https://travis-ci.org/gookit/sux.svg?branch=master)](https://travis-ci.org/gookit/sux)
[![Coverage Status](https://coveralls.io/repos/github/gookit/sux/badge.svg?branch=master)](https://coveralls.io/github/gookit/sux?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/sux)](https://goreportcard.com/report/github.com/gookit/sux)

Simple and fast request router for golang HTTP applications.

- support route group
- support route path params
- support cache recently accessed dynamic routes
- support route middleware, group middleware, global middleware
- support generic `http.Handler` interface middleware
- support add handlers for handle `NotFound` and `NotAllowed`

> **[中文说明](README_cn.md)**

## Godoc

- [godoc for gopkg](https://godoc.org/gopkg.in/gookit/ini.v1)
- [godoc for github](https://godoc.org/github.com/gookit/ini)

## Quick start

```go
package main

import (
	"github.com/gookit/sux"
)

func main() {
	r := sux.New()
	r.GET("/", func(c *sux.Context) {
		c.Text(200, "hello")
	})
	r.GET("/hello/{name}", func(c *sux.Context) {
		c.Text(200, "hello " + c.Param("name"))
	})
	r.POST("/post", func(c *sux.Context) {
		c.Text(200, "hello")
	})
	r.Group("/articles", func() {
		r.GET("", func(c *sux.Context) {
			c.Text(200, "view list")
		})
		r.POST("", func(c *sux.Context) {
			c.Text(200, "create ok")
		})
		r.GET(`/{id:\d+}`, func(c *sux.Context) {
			c.Text(200, "view detail, id: " + c.Param("id"))
		})
	})

	// quick start
	r.Listen(":8080")
	// can also
	// http.ListenAndServe(":8080", r)
}
```

## Use Middleware

sux support use middleware, allow:

- global middleware
- group middleware
- route middleware

**Call priority**: `global middleware -> group middleware -> route middleware`

### Example

```go
package main

import (
	"fmt"
	"github.com/gookit/sux"
)

func main() {
	s := ""
	r := sux.New()
	
	// add global middleware
	r.Use(func(c *sux.Context) {
	    // do something ...
	})
	
	// use middleware for the route
	route := r.GET("/middle", func(c *sux.Context) { // main handler
		s += "-O-"
	}, func(c *sux.Context) { // middle 1
     		s += "a"
     		c.Next() // Notice: call Next()
     		s += "A"
     		// c.Abort() // if call Abort(), will abort at the end of this middleware run
     	})
	// add by Use()
	route.Use(func(c *sux.Context) { // middle 2
		s += "b"
		c.Next()
		s += "B"
	})

	// now, send a GET request to /middle
	fmt.Print(s)
	// OUT: ab-O-BA
}
```

- **Call sequence**: `middle 1 -> middle 2 -> main handler -> middle 2 -> middle 1`
- **Flow chart**:

```text
        +----------------------------+
        | middle 1                   |
        |  +---------------------+   |
        |  | middle 2            |   |
 start  |  |  +---------------+  |   | end
------->|  |  |     main      |  |   |--->----
        |  |  |    handler    |  |   |
        |  |  |_______________|  |   |    
        |  |_____________________|   |  
        |____________________________|
```

> more please see [middleware_test.go](middleware_test.go) middleware tests

## Use http.Handler

sux is support generic `http.Handler` interface middleware

> You can use `sux.WarpHttpHandler()` convert `http.Handler` as `sux.HandlerFunc`

```go
package main

import (
	"net/http"
	"github.com/gookit/sux"
	// here we use gorilla/handlers, it provides some generic handlers.
	"github.com/gorilla/handlers"
)

func main() {
	r := sux.New()
	
	// create a simple generic http.Handler
	h0 := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("new-key", "val")
	})
	
	r.Use(sux.WarpHttpHandler(h0), sux.WarpHttpHandler(handlers.ProxyHeaders()))
	
	r.GET("/", func(c *sux.Context) {
		c.Text(200, "hello")
	})
	// add routes ...
	
    // Wrap our server with our gzip handler to gzip compress all responses.
    http.ListenAndServe(":8000", handlers.CompressHandler(r))
}
```

## Multi domains

> code is ref from `julienschmidt/httprouter`

```go
package main

import (
	"log"
	"net/http"
	"github.com/gookit/sux"
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
	router := sux.New()
	router.GET("/", Index)
	router.GET("/hello/{name}", func(c *sux.Context) {})

	// Make a new HostSwitch and insert the router (our http handler)
	// for example.com and port 12345
	hs := make(HostSwitch)
	hs["example.com:12345"] = router

	// Use the HostSwitch to listen and serve on port 12345
	log.Fatal(http.ListenAndServe(":12345", hs))
}
```

## Other

- run tests

```bash
go test -cover
go test -bench .
```

- code format

```bash
go fmt ./...
```

- GoLint

```bash
golint
```

## Ref

- https://github.com/gin-gonic/gin
- https://github.com/gorilla/mux
- https://github.com/julienschmidt/httprouter
- https://github.com/xialeistudio/go-dispatcher

## License

**MIT**
