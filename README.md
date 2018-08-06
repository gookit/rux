# Simple Router

[![GoDoc](https://godoc.org/github.com/gookit/sux?status.svg)](https://godoc.org/github.com/gookit/sux)
[![Build Status](https://travis-ci.org/gookit/sux.svg?branch=master)](https://travis-ci.org/gookit/sux)
[![Coverage Status](https://coveralls.io/repos/github/gookit/sux/badge.svg?branch=master)](https://coveralls.io/github/gookit/sux?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/sux)](https://goreportcard.com/report/github.com/gookit/sux)

A simple and fast router for golang http application

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
	r.GET("/users/{id}", func(c *sux.Context) {
		c.Text(200, "hello")
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

MIT
