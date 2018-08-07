package sux

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
	// Call sequence: middle 1 -> middle 2 -> main handler -> middle 2 -> middle 1
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
	// Call sequence: middle 1 -> middle 2 -> main handler -> middle 2 -> middle 1
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

func TestGroupMiddleware(t *testing.T) {
	art := assert.New(t)
	r := New()
	art.NotEmpty(r)

	r.Group("/grp", func() {
		r.GET("/middle", namedHandler) // main handler
	})
}
