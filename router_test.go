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

// SiteController define a controller
type SiteController struct {
}

func (c *SiteController) AddRoutes(r *Router) {
	r.GET("{id}", c.Get)
	r.POST("", c.Post)
}

func (c *SiteController) Get(ctx *Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
	ctx.WriteString("\n ok")
}

func (c *SiteController) Post(ctx *Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
}

func namedHandler(c *Context) {
	c.Set("name", "namedHandler")
}

func TestRouter(t *testing.T) {
	art := assert.New(t)

	r := New()
	art.NotEmpty(r)

	route := r.GET("/get", emptyHandler)
	route.Use(func(c *Context) {
		// do something...
	})

	ret := r.Match("GET", "/get")
	art.Equal(Found, ret.Status)
	art.Len(ret.Handlers, 1)
}

func TestAddRoute(t *testing.T) {
	art := assert.New(t)

	r := New()
	art.NotEmpty(r)

	// no handler
	art.Panics(func() {
		r.GET("/get", nil)
	})

	// invalid method
	art.Panics(func() {
		r.Add("invalid", "/get", emptyHandler)
	})

	route := r.GET("/get", namedHandler)
	art.NotEmpty(route.Handler())
	art.Equal("/get", route.path)
	// art.Equal(fmt.Sprint(*namedHandler), route.Handler())
	art.Equal("github.com/gookit/sux.namedHandler", route.HandlerName())

	ret := r.Match("GET", "/get")
	art.Equal(Found, ret.Status)
	art.NotEmpty(ret.Handler)

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

	r.Any("/any", emptyHandler)
	for _, m := range anyMethods {
		ret = r.Match(m, "/any")
		art.Equal(Found, ret.Status)
	}

	ret = r.Match(GET, "/not-exist")
	art.Equal(NotFound, ret.Status)
	art.Nil(ret.Handlers)

	// add a controller
	r.Controller("/site", &SiteController{})
	ret = r.Match(GET, "/site/12")
	art.Equal(Found, ret.Status)
	ret = r.Match(POST, "/site")
	art.Equal(Found, ret.Status)
}

func TestRouter_Group(t *testing.T) {
	art := assert.New(t)

	r := New()
	art.NotEmpty(r)

	r.Group("/users", func(g *Router) {
		g.GET("", emptyHandler)
		g.GET("/{id}", emptyHandler)
	}, func(c *Context) {
		// add middleware handlers for group
	})

	ret := r.Match(GET, "/users")
	art.Equal(Found, ret.Status)
	art.NotEmpty(ret.Handler)
	art.Len(ret.Handlers, 1)

	ret = r.Match(GET, "/users/23")
	art.Equal(Found, ret.Status)
	art.NotEmpty(ret.Handler)
	art.Len(ret.Handlers, 1)
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

	ret = r.Match(GET, "/users/str")
	art.Equal(Found, ret.Status)
	art.Equal("str", ret.Params["id"])
	ret = r.Match(GET, "/not/exist")
	art.Equal(NotFound, ret.Status)

	r.GET("/site/settings/{id}", emptyHandler)
	ret = r.Match(GET, "/site/exist")
	art.Equal(NotFound, ret.Status)

	// add regex for var
	r.GET(`/path1/{id:[1-9]\d*}`, emptyHandler)
	ret = r.Match(GET, "/path1/23")
	art.Equal(Found, ret.Status)
	ret = r.Match(GET, "/path1/err")
	art.Equal(NotFound, ret.Status)

	// use internal var
	r.GET(`/path2/{num}`, emptyHandler)
	ret = r.Match(GET, "/path2/23")
	art.Equal(Found, ret.Status)
	ret = r.Match(GET, "/path2/-23")
	art.Equal(NotFound, ret.Status)
	ret = r.Match(GET, "/path2/err")
	art.Equal(NotFound, ret.Status)
}

func TestOptionalRoute(t *testing.T) {
	art := assert.New(t)

	r := New()
	art.NotEmpty(r)

	// invalid
	art.Panics(func() {
		r.Add(GET, "/blog[/{category}]/{id}", emptyHandler)
	})

	// simple
	r.Add(GET, "/about[.html]", emptyHandler)

	ret := r.Match(GET, "about")
	art.Equal(Found, ret.Status)
	ret = r.Match(GET, "/about")
	art.Equal(Found, ret.Status)
	ret = r.Match(GET, "/about.html")
	art.Equal(Found, ret.Status)

	// with Params
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

	art.Contains(r.String(), "Routes Count: 3")

	ret := r.Match(GET, "/path/some")
	art.Equal(Found, ret.Status)

	ret = r.Match(POST, "/path/some")
	art.Equal(NotAllowed, ret.Status)
	art.Len(ret.Handlers, 0)
	art.Len(ret.AllowedMethods, 3)

	allowed := ret.AllowedMethods
	art.Contains(allowed, "GET")
	art.Contains(allowed, "PUT")
	art.Contains(allowed, "DELETE")
}

func TestOther(t *testing.T) {
	art := assert.New(t)

	SetGlobalVar("name", `\w+`)
	m := GetGlobalVars()
	art.Equal(`\w+`, m["name"])
}
