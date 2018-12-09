package rux

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
	is := assert.New(t)

	r := New()
	is.NotEmpty(r)

	route := r.GET("/get", emptyHandler)
	route.Use(func(c *Context) {
		// do something...
	})

	// cannot set options on after init
	is.Panics(func() {
		r.WithOptions(HandleMethodNotAllowed)
	})

	ret := r.Match("GET", "/get")
	is.Equal(Found, ret.Status)
	is.Len(ret.Handlers, 1)
}

func TestAddRoute(t *testing.T) {
	is := assert.New(t)

	r := New()
	is.NotEmpty(r)

	// no handler
	is.Panics(func() {
		r.GET("/get", nil)
	})

	// invalid method
	is.Panics(func() {
		r.Add("invalid", "/get", emptyHandler)
	})

	route := r.GET("/get", namedHandler)
	is.NotEmpty(route.Handler())
	is.Equal("/get", route.path)
	// is.Equal(fmt.Sprint(*namedHandler), route.Handler())
	is.Equal("github.com/gookit/rux.namedHandler", route.HandlerName())

	ret := r.Match("GET", "/get")
	is.Equal(Found, ret.Status)
	is.NotEmpty(ret.Handler)

	ret = r.Match(HEAD, "/get")
	is.Equal(Found, ret.Status)

	// other methods
	r.HEAD("/head", emptyHandler)
	r.POST("/post", emptyHandler)
	r.PUT("/put", emptyHandler)
	r.PATCH("/patch", emptyHandler)
	r.TRACE("/trace", emptyHandler)
	r.OPTIONS("/options", emptyHandler)
	r.DELETE("/delete", emptyHandler)
	r.CONNECT("/connect", emptyHandler)

	for _, m := range AnyMethods() {
		ret = r.Match(m, "/"+strings.ToLower(m))
		is.Equal(Found, ret.Status)
	}

	r.Any("/any", emptyHandler)
	for _, m := range anyMethods {
		ret = r.Match(m, "/any")
		is.Equal(Found, ret.Status)
	}

	ret = r.Match(GET, "/not-exist")
	is.Equal(NotFound, ret.Status)
	is.Nil(ret.Handlers)
	is.Len(ret.Handlers, 0)
	is.Nil(ret.Handlers.Last())

	// add a controller
	r.Controller("/site", &SiteController{})
	ret = r.Match(GET, "/site/12")
	is.Equal(Found, ret.Status)
	ret = r.Match(POST, "/site")
	is.Equal(Found, ret.Status)

	// add fallback route
	r.Any("/*", emptyHandler)
	for _, m := range anyMethods {
		ret = r.Match(m, "/not-exist")
		is.Equal(Found, ret.Status)
	}
}

func TestNameRoute(t *testing.T) {
	is := assert.New(t)
	r := New()

	// named route
	r.GET("/path1", emptyHandler).NamedTo("route1", r)

	r2 := NewRoute("post", "/path2", emptyHandler)
	r2.SetName("route2").AttachTo(r)

	r3 := NewRoute("get", "/path3", emptyHandler).SetName("route3")
	r.AddRoute(r3)

	route := r.GetRoute("not-exist")
	is.Nil(route)

	route = r.GetRoute("route1")
	is.NotEmpty(route)
	is.Equal("/path1", route.Path())
	is.Equal("GET", route.Method())

	info := route.Info()
	is.Equal("/path1", info.Path)
	is.Equal("GET", info.Method)

	route = r.GetRoute("route2")
	is.NotEmpty(route)
	is.Equal(route, r2)

	route = r.GetRoute("route3")
	is.NotEmpty(route)
	is.Equal(route, r3)
}

func TestRouter_Group(t *testing.T) {
	is := assert.New(t)

	r := New()
	is.NotEmpty(r)

	r.Group("/users", func() {
		r.GET("", emptyHandler)
		r.GET("/{id}", emptyHandler)

		// overflow max num of the route handlers
		is.Panics(func() {
			var i int8 = -1
			var hs HandlersChain
			for ; i <= abortIndex; i++ {
				hs = append(hs, emptyHandler)
			}

			r.GET("/overflow", emptyHandler, hs...)
		})
	}, func(c *Context) {
		// ...
	})

	ret := r.Match(GET, "/users")
	is.Equal(Found, ret.Status)
	is.NotEmpty(ret.Handler)
	is.Len(ret.Handlers, 1)

	ret = r.Match(GET, "/users/23")
	is.Equal(Found, ret.Status)
	is.NotEmpty(ret.Handler)
	is.Len(ret.Handlers, 1)
}

func TestDynamicRoute(t *testing.T) {
	is := assert.New(t)

	r := New()
	is.NotEmpty(r)

	r.GET("/users/{id}", emptyHandler)

	ret := r.Match(GET, "/users/23")
	is.Equal(Found, ret.Status)
	is.Len(ret.Params, 1)
	is.False(ret.Params.Has("no-key"))
	is.True(ret.Params.Has("id"))
	// get param
	is.Equal("23", ret.Params["id"])
	is.Equal("", ret.Params.String("no-key"))
	is.Equal("23", ret.Params.String("id"))
	is.Equal(23, ret.Params.Int("id"))
	is.Equal(0, ret.Params.Int("no-key"))

	ret = r.Match(GET, "/users/str")
	is.Equal(Found, ret.Status)
	is.Equal("str", ret.Params["id"])
	ret = r.Match(GET, "/not/exist")
	is.Equal(NotFound, ret.Status)

	r.GET("/site/settings/{id}", emptyHandler)
	ret = r.Match(GET, "/site/exist")
	is.Equal(NotFound, ret.Status)

	// add regex for var
	r.GET(`/path1/{id:[1-9]\d*}`, emptyHandler)
	ret = r.Match(GET, "/path1/23")
	is.Equal(Found, ret.Status)
	ret = r.Match(GET, "/path1/err")
	is.Equal(NotFound, ret.Status)

	// use internal var
	r.GET(`/path2/{num}`, emptyHandler)
	ret = r.Match(GET, "/path2/23")
	is.Equal(Found, ret.Status)
	ret = r.Match(GET, "/path2/-23")
	is.Equal(NotFound, ret.Status)
	ret = r.Match(GET, "/path2/err")
	is.Equal(NotFound, ret.Status)

	r.GET(`/path3/{level:[1-9]{1,2}}`, emptyHandler)
	ret = r.Match(GET, "/path3/2")
	is.Equal(Found, ret.Status)
	is.True(ret.Params.Has("level"))
	is.Equal("2", ret.Params.String("level"))
	ret = r.Match(GET, "/path3/123")
	is.Equal(NotFound, ret.Status)

	r.GET(`/assets/{file:.+\.(?:css|js)}`, emptyHandler)
	ret = r.Match(GET, "/assets/site.css")
	is.Equal(Found, ret.Status)
	ret = r.Match(GET, "/assets/site.js")
	is.Equal(Found, ret.Status)
	ret = r.Match(GET, "/assets/site.tx")
	is.Equal(NotFound, ret.Status)
}

func TestFixFirstNodeOnlyOneChar(t *testing.T) {
	is := assert.New(t)

	r := New()
	r.PATCH(`/r/{name}/hq2hah9/dxt/g/hoovln`, emptyHandler)
	ret := r.Match(PATCH, "/r/lnamel/hq2hah9/dxt/g/hoovln")
	is.Equal(Found, ret.Status)
}

func TestOptionalRoute(t *testing.T) {
	is := assert.New(t)

	r := New()
	is.NotEmpty(r)

	// invalid
	is.Panics(func() {
		r.Add(GET, "/blog[/{category}]/{id}", emptyHandler)
	})

	// simple
	r.Add(GET, "/about[.html]", emptyHandler)

	ret := r.Match(GET, "about")
	is.Equal(Found, ret.Status)
	ret = r.Match(GET, "/about")
	is.Equal(Found, ret.Status)
	ret = r.Match(GET, "/about.html")
	is.Equal(Found, ret.Status)

	// with Params
	r.Add(GET, "/blog[/{category}]", emptyHandler)

	ret = r.Match(GET, "/blog")
	is.Equal(Found, ret.Status)
	ret = r.Match(GET, "/blog/golang")
	is.Equal(Found, ret.Status)
}

func TestMethodNotAllowed(t *testing.T) {
	is := assert.New(t)

	// enable handle not allowed
	r := New(HandleMethodNotAllowed)
	is.True(r.handleMethodNotAllowed)

	r.Add(GET, "/path/some", emptyHandler)
	r.Add(PUT, "/path/{var}", emptyHandler)
	r.Add(DELETE, "/path[/{var}]", emptyHandler)

	is.Contains(r.String(), "Routes Count: 3")

	ret := r.Match(GET, "/path/some")
	is.Equal(Found, ret.Status)

	ret = r.Match(POST, "/path/some")
	is.Equal(NotAllowed, ret.Status)
	is.Len(ret.Handlers, 0)
	is.Len(ret.AllowedMethods, 3)

	allowed := ret.AllowedMethods
	is.Contains(allowed, "GET")
	is.Contains(allowed, "PUT")
	is.Contains(allowed, "DELETE")
}

func TestOther(t *testing.T) {
	is := assert.New(t)

	SetGlobalVar("name", `\w+`)
	m := GetGlobalVars()
	is.Equal(`\w+`, m["name"])

	// open debug
	Debug(true)
	is.True(IsDebug())
	r := New()
	r.GET("/debug", emptyHandler)
	Debug(false)
	is.False(IsDebug())
}

func TestRouter_WithOptions(t *testing.T) {
	is := assert.New(t)

	// Option: StrictLastSlash
	r := New(StrictLastSlash)
	is.True(r.strictLastSlash)

	r.GET("/users", func(c *Context) {
		c.Text(200, "val0")
	})
	r.GET("/users/", func(c *Context) {
		c.Text(200, "val1")
	})

	w := mockRequest(r, "GET", "/users", nil)
	is.Equal("val0", w.Body.String())
	w = mockRequest(r, "GET", "/users/", nil)
	is.Equal("val1", w.Body.String())

	// Option: UseEncodedPath
	r = New()
	r.WithOptions(UseEncodedPath)
	r.GET("/users/with spaces", func(c *Context) {
		c.Text(200, "val0")
	})
	// "with spaces" -> "with%20spaces"
	r.GET("/users/with%20spaces", func(c *Context) {
		c.Text(200, "val1")
	})
	w = mockRequest(r, "GET", "/users/with%20spaces", nil)
	is.Equal("val1", w.Body.String())
	w = mockRequest(r, "GET", "/users/with spaces", nil)
	is.Equal("val1", w.Body.String())

	// Option: InterceptAll
	r = New(InterceptAll("/coming-soon"))
	// Notice: must add a route and path equals to 'InterceptAll' and use Any()
	r.Any("/coming-soon", func(c *Context) {
		c.Text(200, "coming-soon")
	})
	r.GET("/users", func(c *Context) {
		c.Text(200, "val0")
	})

	w = mockRequest(r, "GET", "/users", nil)
	is.Equal("coming-soon", w.Body.String())
	w = mockRequest(r, "GET", "/not-exist", nil)
	is.Equal("coming-soon", w.Body.String())
	w = mockRequest(r, "POST", "/not-exist", nil)
	is.Equal("coming-soon", w.Body.String())

	// Option: EnableCaching, MaxNumCaches
	r = New(EnableCaching, MaxNumCaches(10))
	simpleHandler := func(c *Context) {
		c.Text(200, "id:"+c.Param("id"))
	}
	r.GET("/users/{id}", simpleHandler)
	w = mockRequest(r, "GET", "/users/23", nil)
	is.Equal("id:23", w.Body.String())
	w = mockRequest(r, "GET", "/users/23", nil)
	is.Equal("id:23", w.Body.String())
	is.Len(r.cachedRoutes, 1)

	for id := range []int{19: 0} {
		idStr := fmt.Sprint(id)
		w = mockRequest(r, "GET", "/users/"+idStr, nil)
		is.Equal("id:"+idStr, w.Body.String())
	}

	// Option: MaxMultipisMemory 8M
	// r = New(MaxMultipisMemory(8 << 20))
	// is.Equal(8 << 20, r.maxMultipisMemory)
}

func TestRouterStaticAssets(t *testing.T) {
	r := New()
	is := assert.New(t)

	// one file
	r.StaticFile("/site.js", "testdata/site.js")
	w := mockRequest(r, "GET", "/site.js", nil)
	is.Equal(200, w.Code)
	is.Equal("application/javascript", w.Header().Get("Content-Type"))
	is.Contains(w.Body.String(), "console.log")
	// try again
	w = mockRequest(r, "GET", "/site.js?t=33455", nil)
	is.Equal(200, w.Code)

	// allow any files in the dir.
	r.StaticDir("/static", "testdata")
	w = mockRequest(r, "GET", "/static/site.css", nil)
	is.Equal(200, w.Code)
	is.Equal("text/css; charset=utf-8", w.Header().Get("Content-Type"))
	is.Contains(w.Body.String(), "max-width")
	w = mockRequest(r, "GET", "/static/site.js", nil)
	is.Equal(200, w.Code)
	is.Equal("application/javascript", w.Header().Get("Content-Type"))
	is.Contains(w.Body.String(), "console.log")
	w = mockRequest(r, "GET", "/static/site.md", nil)
	is.Equal(200, w.Code)

	// add file type limit
	// r.StaticFiles("", "testdata", "css|js")
	r.StaticFiles("/assets", "testdata", "css|js")
	w = mockRequest(r, "GET", "/assets/site.js", nil)
	is.Equal(200, w.Code)
	is.Equal("application/javascript", w.Header().Get("Content-Type"))
	is.Contains(w.Body.String(), "console.log")
	w = mockRequest(r, "GET", "/assets/site.md", nil)
	is.Equal(404, w.Code)

	// StaticFunc
	r.StaticFunc("/some/test.txt", func(c *Context) {
		c.Text(200, "content")
	})
	w = mockRequest(r, "GET", "/some/test.txt", nil)
	is.Equal(200, w.Code)
	is.Equal("text/plain; charset=UTF-8", w.Header().Get("Content-Type"))
	is.Contains(w.Body.String(), "content")
}
