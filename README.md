# Simple Router

[![GoDoc](https://godoc.org/github.com/gookit/sux?status.svg)](https://godoc.org/github.com/gookit/sux)
[![Build Status](https://travis-ci.org/gookit/sux.svg?branch=master)](https://travis-ci.org/gookit/sux)
[![Coverage Status](https://coveralls.io/repos/github/gookit/sux/badge.svg?branch=master)](https://coveralls.io/github/gookit/sux?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/sux)](https://goreportcard.com/report/github.com/gookit/sux)

A simple and fast router for golang http application

- support route group
- support route path params
- support cache recently accessed dynamic routes

## Usage

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
	r.Group("/articles", func(g *sux.Router) {
		g.GET("", func(c *sux.Context) {
			c.Text(200, "view list")
		})
		g.POST("", func(c *sux.Context) {
			c.Text(200, "create ok")
		})
		g.GET(`/{id:\d+}`, func(c *sux.Context) {
			c.Text(200, "view detail, id: " + c.Param("id"))
		})
	})

	r.Listen(":8080")
}
```

## Middleware handlers

```go
package main

import (
	"fmt"
	"github.com/gookit/sux"
)

func main() {
	s := ""
	r := sux.New()
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

- Call sequence: `middle 1 -> middle 2 -> main handler -> middle 1 -> middle 2`

> more please see [dispatch_test.go](dispatch_test.go) middleware tests

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

- tests

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

- https://github.com/xialeistudio/go-dispatcher
- https://github.com/julienschmidt/httprouter
- https://github.com/gorilla/mux

## License

**MIT**
