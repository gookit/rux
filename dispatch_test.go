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

	// use params
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

func TestRouteMiddleware(t *testing.T) {
	art := assert.New(t)

	r := New()
	s := &aStr{}

	// add one middleware
	r.GET("/middle", func(c *Context) {
		s.append("-O-")
	}, func(c *Context) {
		s.append("a")
		c.Next()
		s.append("b")
	})

	mockRequest(r, GET, "/middle", "")
	art.Equal("a-O-b", s.str)

	// add multi middleware
	r.GET("/middle2", func(c *Context) { // main handler
		s.append("-O-")
	}, func(c *Context) { // middle 1
		s.append("a")
		c.Next()
		s.append("A")
	}, func(c *Context) { // middle 2
		s.append("b")
		c.Next()
		s.append("B")
	})
	// Call sequence: middle 1 -> middle 2 -> main handler -> middle 1 -> middle 2
	s.reset()
	mockRequest(r, GET, "/middle2", "")
	art.Equal("ab-O-BA", s.str)

	// add multi middleware(don't call next)
	r.GET("/middle3", func(c *Context) { // main handler
		s.append("-O-")
	}, func(c *Context) { // middle 1
		s.append("a")
		// c.Next()
		s.append("A")
	}, func(c *Context) { // middle 2
		s.append("b")
		// c.Next()
		s.append("B")
	})
	// Call sequence: middle 1 -> middle 2 -> main handler
	s.reset()
	mockRequest(r, GET, "/middle3", "")
	art.Equal("aAbB-O-", s.str)

	// add middleware use method Use()
	route := r.GET("/middle4", func(c *Context) { // main handler
		s.append("-O-")
	})
	route.Use(func(c *Context) { // middle 1
		s.append("a")
		c.Next()
		s.append("A")
	}, func(c *Context) { // middle 2
		s.append("b")
		c.Next()
		s.append("B")
	})
	// Call sequence: middle 1 -> middle 2 -> main handler -> middle 1 -> middle 2
	s.reset()
	mockRequest(r, GET, "/middle4", "")
	art.Equal("ab-O-BA", s.str)
}

func TestMiddlewareAbort(t *testing.T) {
	art := assert.New(t)

	r := New()
	s := &aStr{}

	// use middleware, will termination execution early by Abort()
	r.GET("/abort", func(c *Context) { // Will not execute
		s.append("-O-")
	}, func(c *Context) {
		s.append("a")
		// c.Next()
		c.Abort() // Will abort at the end of this middleware run
		s.append("A")
	}, func(c *Context) { // Will not execute
		s.append("b")
		c.Next()
		s.append("B")
	})
	// Call sequence: middle 1
	s.reset()
	mockRequest(r, GET, "/abort", "")
	art.Equal("aA", s.str)
}

func TestContext(t *testing.T) {
	art := assert.New(t)
	r := New()

	route := r.GET("/ctx", namedHandler)
	route.Use(func(c *Context) {
		art.NotEmpty(c.Handler())
		art.Equal("github.com/gookit/sux.namedHandler", c.HandlerName())
		// set a new context data
		c.Set("newKey", "val")
		c.Next()
		art.Equal("namedHandler1", c.Get("name").(string))
	}, func(c *Context) { // Will not execute
		art.Equal("val", c.Get("newKey").(string))
		c.Next()
		art.Equal("namedHandler", c.Get("name").(string))
		c.Set("name", "namedHandler1") // change value
	})

	mockRequest(r, GET, "/ctx", "data")
}
