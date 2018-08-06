package sux

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
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

func (a *aStr) prepend(s string) {
	a.str = s + a.str
}

func TestRouter_ServeHTTP(t *testing.T) {
	art := assert.New(t)

	r := New()
	s := &aStr{}

	r.GET("/", func(c *Context) {
		s.set("ok")

		art.Equal(c.URL().Path, "/")
	})
	mockRequest(r, GET, "/", "")
	art.Equal("ok", s.str)

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

	// not found handle
	r.NotFound(func(c *Context) {
		s.set("not-found")
	})
	mockRequest(r, GET, "/not-exist", "")
	art.Equal("not-found", s.str)

	// not allowed handle
	r.HandleMethodNotAllowed = true
	r.NotAllowed(func(c *Context) {
		s.set("not-allowed")
	})

	mockRequest(r, POST, "/users/23", "")
	art.Equal("not-allowed", s.str)

}

func TestRoute_Use(t *testing.T) {
	art := assert.New(t)

	r := New()
	s := &aStr{}

	// add an middleware
	r.GET("/middle", func(c *Context) {
		s.append("-O-")
	}, func(c *Context) {
		s.prepend("A")
		c.Next()
		s.append("B")
	})
	mockRequest(r, GET, "/middle", "")
	art.Equal("A-O-B", s.str)

	// add multi middleware
	s.reset()
	r.GET("/middle2", func(c *Context) {
		s.append("-O-")
	}, func(c *Context) {
		s.prepend("a")
		c.Next()
		s.append("A")
	}, func(c *Context) {
		s.prepend("b")
		c.Next()
		s.append("B")
	})
	mockRequest(r, GET, "/middle2", "")
	art.Equal("ba-O-BA", s.str)

	// add multi middleware(Termination execution early)
	s.reset()
	r.GET("/middle3", func(c *Context) {
		s.set("-O-")
	}, func(c *Context) {
		s.prepend("a")
		// fmt.Println(s.str)
		// c.Next()
		s.append("A")
	}, func(c *Context) {
		s.prepend("b")
		c.Next()
		s.append("B")
	})
	mockRequest(r, GET, "/middle3", "")
	art.Equal("ba-O-AB", s.str)
}