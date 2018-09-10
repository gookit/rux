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

	// cannot set options on after init
	art.Panics(func() {
		r.WithOptions(HandleMethodNotAllowed)
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

	for _, m := range AnyMethods() {
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
	art.Len(ret.Handlers, 0)
	art.Nil(ret.Handlers.Last())

	// add a controller
	r.Controller("/site", &SiteController{})
	ret = r.Match(GET, "/site/12")
	art.Equal(Found, ret.Status)
	ret = r.Match(POST, "/site")
	art.Equal(Found, ret.Status)

	// add fallback route
	r.Any("/*", emptyHandler)
	for _, m := range anyMethods {
		ret = r.Match(m, "/not-exist")
		art.Equal(Found, ret.Status)
	}
}

func TestRouter_Group(t *testing.T) {
	art := assert.New(t)

	r := New()
	art.NotEmpty(r)

	r.Group("/users", func() {
		r.GET("", emptyHandler)
		r.GET("/{id}", emptyHandler)

		// overflow max num of the route handlers
		art.Panics(func() {
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

	r.GET(`/path3/{level:[1-9]{1,2}}`, emptyHandler)
	ret = r.Match(GET, "/path3/2")
	art.Equal(Found, ret.Status)
	art.True(ret.Params.Has("level"))
	art.Equal("2", ret.Params.String("level"))
	ret = r.Match(GET, "/path3/123")
	art.Equal(NotFound, ret.Status)

	r.GET(`/assets/{file:.+\.(?:css|js)}`, emptyHandler)
	ret = r.Match(GET, "/assets/site.css")
	art.Equal(Found, ret.Status)
	ret = r.Match(GET, "/assets/site.js")
	art.Equal(Found, ret.Status)
	ret = r.Match(GET, "/assets/site.tx")
	art.Equal(NotFound, ret.Status)
}

func TestFixFirstNodeOnlyOneChar(t *testing.T) {
	art := assert.New(t)

	r := New()
	r.PATCH(`/r/{name}/hq2hah9/dxt/g/hoovln`, emptyHandler)
	ret := r.Match(PATCH, "/r/lnamel/hq2hah9/dxt/g/hoovln")
	art.Equal(Found, ret.Status)
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

	// enable handle not allowed
	r := New(HandleMethodNotAllowed)
	art.True(r.handleMethodNotAllowed)

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

	// open debug
	Debug(true)
	art.True(IsDebug())
	r := New()
	r.GET("/debug", emptyHandler)
	Debug(false)
	art.False(IsDebug())
}

func TestRouter_WithOptions(t *testing.T) {
	art := assert.New(t)

	// Option: StrictLastSlash
	r := New(StrictLastSlash)
	art.True(r.strictLastSlash)

	r.GET("/users", func(c *Context) {
		c.Text(200, "val0")
	})
	r.GET("/users/", func(c *Context) {
		c.Text(200, "val1")
	})

	w := mockRequest(r, "GET", "/users", nil)
	art.Equal("val0", w.Body.String())
	w = mockRequest(r, "GET", "/users/", nil)
	art.Equal("val1", w.Body.String())

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
	art.Equal("val1", w.Body.String())
	w = mockRequest(r, "GET", "/users/with spaces", nil)
	art.Equal("val1", w.Body.String())

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
	art.Equal("coming-soon", w.Body.String())
	w = mockRequest(r, "GET", "/not-exist", nil)
	art.Equal("coming-soon", w.Body.String())
	w = mockRequest(r, "POST", "/not-exist", nil)
	art.Equal("coming-soon", w.Body.String())

	// Option: EnableCaching, MaxNumCaches
	r = New(EnableCaching, MaxNumCaches(10))
	simpleHandler := func(c *Context) {
		c.Text(200, "id:"+c.Param("id"))
	}
	r.GET("/users/{id}", simpleHandler)
	w = mockRequest(r, "GET", "/users/23", nil)
	art.Equal("id:23", w.Body.String())
	w = mockRequest(r, "GET", "/users/23", nil)
	art.Equal("id:23", w.Body.String())
	art.Len(r.cachedRoutes, 1)

	for id := range []int{19: 0} {
		idStr := fmt.Sprint(id)
		w = mockRequest(r, "GET", "/users/"+idStr, nil)
		art.Equal("id:"+idStr, w.Body.String())
	}

	// Option: MaxMultipartMemory 8M
	// r = New(MaxMultipartMemory(8 << 20))
	// art.Equal(8 << 20, r.maxMultipartMemory)
}

func TestRouter_StaticAssets(t *testing.T) {
	r := New()
	art := assert.New(t)

	// one file
	r.StaticFile("/site.js", "testdata/site.js")
	w := mockRequest(r, "GET", "/site.js", nil)
	art.Equal(200, w.Code)
	art.Equal("application/javascript", w.Header().Get("Content-Type"))
	art.Contains(w.Body.String(), "console.log")
	// try again
	w = mockRequest(r, "GET", "/site.js?t=33455", nil)
	art.Equal(200, w.Code)

	// allow any files in the dir.
	r.StaticDir("/static", "testdata")
	w = mockRequest(r, "GET", "/static/site.css", nil)
	art.Equal(200, w.Code)
	art.Equal("text/css; charset=utf-8", w.Header().Get("Content-Type"))
	art.Contains(w.Body.String(), "max-width")
	w = mockRequest(r, "GET", "/static/site.js", nil)
	art.Equal(200, w.Code)
	art.Equal("application/javascript", w.Header().Get("Content-Type"))
	art.Contains(w.Body.String(), "console.log")
	w = mockRequest(r, "GET", "/static/site.md", nil)
	art.Equal(200, w.Code)

	// add file type limit
	// r.StaticFiles("", "testdata", "css|js")
	r.StaticFiles("/assets", "testdata", "css|js")
	w = mockRequest(r, "GET", "/assets/site.js", nil)
	art.Equal(200, w.Code)
	art.Equal("application/javascript", w.Header().Get("Content-Type"))
	art.Contains(w.Body.String(), "console.log")
	w = mockRequest(r, "GET", "/assets/site.md", nil)
	art.Equal(404, w.Code)
}
