package sux

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func ExampleRouter_ServeHTTP() {
	r := New()
	r.GET("/", func(c *Context) {
		c.Text(200, "hello")
	})
	r.GET("/users/{id}", func(c *Context) {
		c.Text(200, "hello")
	})
	r.POST("/post", func(c *Context) {
		c.Text(200, "hello")
	})

	r.Listen(":8080")
}

type aStr struct {
	str string
}

func (a *aStr) reset() {
	a.str = ""
}

func (a *aStr) set(s ...interface{}) {
	a.str = fmt.Sprint(s...)
}

func (a *aStr) append(s string) {
	a.str += s
}

func TestRouter_ServeHTTP(t *testing.T) {
	art := assert.New(t)

	r := New()
	s := &aStr{}

	// simple
	r.GET("/", func(c *Context) {
		s.set("ok")

		art.Equal(c.URL().Path, "/")
	})
	mockRequest(r, GET, "/", "")
	art.Equal("ok", s.str)

	// use Params
	r.GET("/users/{id}", func(c *Context) {
		s.set("id:" + c.Param("id"))
	})
	mockRequest(r, GET, "/users/23", "")
	art.Equal("id:23", s.str)
	mockRequest(r, GET, "/users/tom", "")
	art.Equal("id:tom", s.str)

	// not exist
	s.reset()
	mockRequest(r, GET, "/users", "")
	art.Equal("", s.str)

	// receive input data
	r.POST("/users", func(c *Context) {
		bd, _ := c.RawData()
		s.set("body:", string(bd))

		p := c.Query("page")
		if p != "" {
			s.append(",page=" + p)
		}
	})
	s.reset()
	mockRequest(r, POST, "/users", "data")
	art.Equal("body:data", s.str)
	s.reset()
	mockRequest(r, POST, "/users?page=2", "data")
	art.Equal("body:data,page=2", s.str)

	// add not found handler
	r.NotFound(func(c *Context) {
		s.set("not-found")
	})
	mockRequest(r, GET, "/not-exist", "")
	art.Equal("not-found", s.str)

	// enable handle method not allowed
	r.HandleMethodNotAllowed = true

	// no handler
	s.reset()
	mockRequest(r, POST, "/users/21", "")
	art.Equal("", s.str)

	// add handler
	r.NotAllowed(func(c *Context) {
		s.set("not-allowed")
	})
	s.reset()
	mockRequest(r, POST, "/users/23", "")
	art.Equal("not-allowed", s.str)
}

func TestContext(t *testing.T) {
	art := assert.New(t)
	r := New()

	route := r.GET("/ctx", namedHandler) // main handler

	route.Use(func(c *Context) { // middle 1
		art.NotEmpty(c.Handler())
		art.Equal("github.com/gookit/sux.namedHandler", c.HandlerName())
		// set a new context data
		c.Set("newKey", "val")
		c.Next()
		art.Equal("namedHandler1", c.Get("name").(string))
	}, func(c *Context) { // middle 2
		_, ok := c.Values()["newKey"]
		art.True(ok)
		art.Equal("val", c.Get("newKey").(string))
		c.Next()
		art.Equal("namedHandler", c.Get("name").(string))
		c.Set("name", "namedHandler1") // change value
	})

	// Call sequence: middle 1 -> middle 2 -> main handler -> middle 1 -> middle 2
	mockRequest(r, GET, "/ctx", "data")
}

func TestRouterListen(t *testing.T) {
	art := assert.New(t)
	r := New()

	// multi params
	art.Panics(func() {
		r.Listen(":8080", "9090")
	})

	art.Error(r.Listen("invalid"))
	art.Error(r.ListenTLS("invalid", "", ""))
	art.Error(r.ListenUnix(""))
}
