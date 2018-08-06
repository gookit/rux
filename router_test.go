package sux

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func Example() {
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

	ret := r.Match("GET", "/")
	fmt.Println(ret.Status)
	ret1 := r.Match("GET", "/users/23")
	fmt.Print(ret1.Status, ret1.Params)

	// run http server
	// r.Listen(":8080")

	// Output:
	// 1
	// 1 map[id:23]
}

var emptyHandler = func(c *Context) {}

func TestRouter(t *testing.T) {
	art := assert.New(t)

	r := New()
	art.NotEmpty(r)
}

func TestAddRoute(t *testing.T) {
	art := assert.New(t)

	r := New()
	art.NotEmpty(r)

	// no handler
	art.Panics(func() {
		r.GET("/get")
	})

	// invalid method
	art.Panics(func() {
		r.Add("invalid", "/get")
	})

	route := r.GET("/get", emptyHandler)
	art.NotEmpty(route)
	art.Equal("/get", route.path)

	ret := r.Match("GET", "/get")
	art.Equal(Found, ret.Status)
	art.Equal(1, ret.Handlers.Len())

	ret = r.Match(HEAD, "/get")
	art.Equal(Found, ret.Status)

	// other methods
	r.HEAD("/head", emptyHandler)
	r.POST("/post", emptyHandler)
	r.PUT("/put", emptyHandler)
	r.PATCH("/patch", emptyHandler)
	r.TRACE("/trace", emptyHandler)
	r.OPTIONS("/options", emptyHandler)
	r.DELETE("/delete", emptyHandler)
	r.CONNECT("/connect", emptyHandler)

	for _, m := range anyMethods {
		ret = r.Match(m, "/"+strings.ToLower(m))
		art.Equal(Found, ret.Status)
	}

	r.ANY("/any", emptyHandler)
	for _, m := range anyMethods {
		ret = r.Match(m, "/any")
		art.Equal(Found, ret.Status)
	}

	ret = r.Match(GET, "/not-exist")
	art.Equal(NotFound, ret.Status)
}

func TestRouter_Group(t *testing.T) {
	art := assert.New(t)

	r := New()
	art.NotEmpty(r)

	r.Group("/users", func(g *Router) {
		g.GET("", emptyHandler)
		g.GET("/{id}", emptyHandler)
	})

	ret := r.Match(GET, "/users")
	art.Equal(Found, ret.Status)

	ret = r.Match(GET, "/users/23")
	art.Equal(Found, ret.Status)
}

func TestDynamicRoute(t *testing.T) {
	art := assert.New(t)

	r := New()
	art.NotEmpty(r)

	r.GET("/users/{id}", emptyHandler)

	ret := r.Match(GET, "/users/23")
	art.Equal(Found, ret.Status)
	art.Len(ret.Params, 1)
	art.False(ret.Params.Has("no-key"))
	art.True(ret.Params.Has("id"))
	// get param
	art.Equal("23", ret.Params["id"])
	art.Equal("", ret.Params.String("no-key"))
	art.Equal("23", ret.Params.String("id"))
	art.Equal(23, ret.Params.Int("id"))
	art.Equal(0, ret.Params.Int("no-key"))
}

func TestOptionalRoute(t *testing.T) {
	art := assert.New(t)

	r := New()
	art.NotEmpty(r)

	// simple
	r.Add(GET, "/about[.html]", emptyHandler)

	ret := r.Match(GET, "about")
	art.Equal(Found, ret.Status)
	ret = r.Match(GET, "/about")
	art.Equal(Found, ret.Status)
	ret = r.Match(GET, "/about.html")
	art.Equal(Found, ret.Status)

	// with params
	r.Add(GET, "/blog[/{category}]", emptyHandler)

	ret = r.Match(GET, "/blog")
	art.Equal(Found, ret.Status)
	ret = r.Match(GET, "/blog/golang")
	art.Equal(Found, ret.Status)
}

func TestMethodNotAllowed(t *testing.T) {
	art := assert.New(t)

	r := New()
	art.NotEmpty(r)

	// enable handle not allowed
	r.HandleMethodNotAllowed = true

	r.Add(GET, "/path/some", emptyHandler)
	r.Add(PUT, "/path/{var}", emptyHandler)
	r.Add(DELETE, "/path[/{var}]", emptyHandler)

	ret := r.Match(GET, "/path/some")
	art.Equal(Found, ret.Status)

	ret = r.Match(POST, "/path/some")
	art.Equal(NotAllowed, ret.Status)
	art.Len(ret.Handlers, 0)
	art.Len(ret.AllowedMethods, 3)

	allowedStr := ret.JoinAllowedMethods(",")
	art.Contains(allowedStr, "GET")
	art.Contains(allowedStr, "PUT")
	art.Contains(allowedStr, "DELETE")
}

func TestOther(t *testing.T) {
	art := assert.New(t)

	SetGlobalVar("name", `\w+`)
	m := GetGlobalVars()
	art.Equal(`\w+`, m["name"])
}
