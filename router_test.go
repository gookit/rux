package rux

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gookit/goutil/envutil"
	"github.com/gookit/goutil/testutil"
	"github.com/stretchr/testify/assert"
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
	r.GET("", c.Index)
	r.GET("{id}", c.Get)
	r.POST("", c.Post)
}

func (c *SiteController) Index(ctx *Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
}

func (c *SiteController) Get(ctx *Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
	ctx.WriteString("\n ok")
}

func (c *SiteController) Post(ctx *Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
}

type Product struct {
}

func (Product) Uses() map[string][]HandlerFunc {
	// HTTPBasicAuth alias
	return map[string][]HandlerFunc{
		"Edit": {
			func(users map[string]string) HandlerFunc {
				return func(c *Context) {
					user, pwd, ok := c.Req.BasicAuth()
					if !ok {
						c.SetHeader("WWW-Authenticate", `Basic realm="THE REALM"`)
						c.AbortWithStatus(401, "Unauthorized")
						return
					}

					if len(users) > 0 {
						srcPwd, ok := users[user]
						if !ok || srcPwd != pwd {
							c.AbortWithStatus(403)
						}
					}

					c.Set("username", user)
					c.Set("password", pwd)
				}
			}(map[string]string{"test": "123"}),
		},
	}
}

// get:all /restful/
func (c *Product) Index(ctx *Context) {
	ctx.WriteString(ctx.Req.Method + " Index")
}

// get:create new record /restful/create
func (c *Product) Create(ctx *Context) {
	ctx.WriteString(ctx.Req.Method + " Create")
}

// post:save record for create /restful
func (c *Product) Store(ctx *Context) {
	ctx.WriteString(ctx.Req.Method + " Store")
}

// get:show record /restful/{id}
func (c *Product) Show(ctx *Context) {
	ctx.WriteString(ctx.Req.Method + " Show " + ctx.Param("id"))
}

// get:edit record /resetful/{id}/edit
func (c *Product) Edit(ctx *Context) {
	ctx.WriteString(ctx.Req.Method + " Edit " + ctx.Param("id"))
}

// put|patch:update record /resetful/{id}
func (c *Product) Update(ctx *Context) {
	ctx.WriteString(ctx.Req.Method + " Update " + ctx.Param("id"))
}

// delete:delete record /resetful/{id}
func (c *Product) Delete(ctx *Context) {
	ctx.WriteString(ctx.Req.Method + " Delete " + ctx.Param("id"))
}

// cannot exported method
func (c *Product) invaid() {
}

// cannot exported method
func (c *Product) invaid2(*Context) {
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

	Debug(true)

	r := New()
	is.NotEmpty(r)

	// no handler
	is.PanicsWithValue("the route handler cannot be empty.(path: '/get')", func() {
		r.GET("/get", nil)
	})

	// invalid method
	is.PanicsWithValue("invalid method name 'INVALID', must in: "+StringMethods, func() {
		r.Add("/get", emptyHandler, "invalid")
	})

	// empty method
	is.PanicsWithValue("the route allowed methods cannot be empty.(path: '/')", func() {
		r.AddRoute(&Route{path: "/", handler: emptyHandler})
	})

	// overflow max num of the route handlers
	is.PanicsWithValue("too many handlers(number: 65)", func() {
		var i int8 = -1
		var hs HandlersChain
		for ; i <= abortIndex; i++ {
			hs = append(hs, emptyHandler)
		}

		r.GET("/overflow", emptyHandler, hs...)
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

	Debug(false)
}

func TestNameRoute(t *testing.T) {
	is := assert.New(t)
	r := New()

	// named route
	r.GET("/path1", emptyHandler).NamedTo("route1", r)

	r2 := r.AddNamed("route2", "/path2[.html]", emptyHandler, POST)
	is.Equal("route2", r2.Name())

	r3 := NamedRoute("route3", "/path3/{id}", emptyHandler, "get")
	r3.AttachTo(r)

	r4 := NewNamedRoute("route4", "/path4/some/{id}", emptyHandler, GET)
	r.AddRoute(r4)

	r5 := NamedRoute("route5", "/path5/some/{id}", emptyHandler, PUT)
	r.AddRoute(r5)

	is.Len(r.Routes(), 5)

	route := r.GetRoute("route1")
	is.NotEmpty(route)
	is.Equal("/path1", route.Path())
	is.Equal("GET", route.MethodString(""))
	is.Equal([]string{"GET"}, route.Methods())

	info := route.Info()
	is.Equal("/path1", info.Path)
	is.Equal([]string{"GET"}, info.Methods)

	route = r.GetRoute("route2")
	is.NotEmpty(route)
	is.Equal(route, r2)
	is.Equal("route2", route.Name())
	_, ok := route.match("/path2")
	is.True(ok)

	route = r.GetRoute("route3")
	is.NotEmpty(route)
	is.Equal(route, r3)
	is.Equal("", route.start)

	route = r.GetRoute("route4")
	is.NotEmpty(route)
	is.Equal(route, r4)
	is.Equal("/path4/some/", route.start)

	route = r.GetRoute("route5")
	is.NotEmpty(route)
	is.Equal(route, r5)
	is.Equal("/path5/some/", route.start)

	route = r.GetRoute("not-exist")
	is.Nil(route)
}

func TestRouter_Group(t *testing.T) {
	is := assert.New(t)

	r := New()
	is.NotEmpty(r)

	r.Group("/users", func() {
		r.GET("", emptyHandler)
		r.GET("/{id}", emptyHandler)
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

	// overflow max num of the route handlers
	is.PanicsWithValue("too many handlers(number: 65)", func() {
		var i int8 = -1
		var hs HandlersChain
		for ; i <= abortIndex; i++ {
			hs = append(hs, emptyHandler)
		}

		r.Group("/test", func() {
			r.GET("", emptyHandler)
			r.GET("/{id}", emptyHandler)
		}, hs...)
	})
}

func TestRouter_Controller(t *testing.T) {
	is := assert.New(t)
	r := New()
	Debug(true)

	r.Controller("/", &SiteController{})

	w := testutil.MockRequest(r, http.MethodGet, "/", nil)
	is.Equal(200, w.Code)
	is.Equal("hello, in /", w.Body.String())

	w = testutil.MockRequest(r, http.MethodGet, "/", nil)
	is.Equal(200, w.Code)
	is.Equal("hello, in /", w.Body.String())

	Debug(false)
}

func TestDynamicRoute(t *testing.T) {
	is := assert.New(t)

	r := New()
	is.NotEmpty(r)

	r0 := r.GET("/users/{id}", emptyHandler)
	is.Equal("", r0.start)

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

	r1 := r.GET("/site/settings/{id}", emptyHandler)
	ret = r.Match(GET, "/site/exist")
	is.Equal(NotFound, ret.Status)

	// test start check.
	is.Equal("/site/settings/", r1.start)
	ps, ok := r1.match("/get")
	is.False(ok)
	is.Nil(ps)

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

func TestMultiPathParam(t *testing.T) {
	ris := assert.New(t)

	r := New()
	r.PATCH(`/news/{category_id}/{new_id:\d+}/detail`, emptyHandler)

	ret := r.Match(PATCH, "/news/100/20/detail")
	ris.Equal(Found, ret.Status)
	ris.Len(ret.Params, 2)
	ris.True(ret.Params.Has("category_id"))
	ris.Equal(100, ret.Params.Int("category_id"))
	ris.True(ret.Params.Has("new_id"))
	ris.Equal(20, ret.Params.Int("new_id"))

	r2 := r.GET(`/news/{category_id}/{new_id:\d+}/{tid:\d+}/detail`, emptyHandler)
	ris.Equal("/news/{category_id}/{new_id}/{tid}/detail", r2.spath)

	ret = r.Match(GET, "/news/100/20/10/detail")
	ris.Equal(Found, ret.Status)
	ris.Len(ret.Params, 3)
	ris.True(ret.Params.Has("category_id"))
	ris.True(ret.Params.Has("new_id"))
	ris.True(ret.Params.Has("tid"))

	ris.PanicsWithValue(`invalid path var regex string, dont allow char '('. var: new_id, regex: (\d+)`, func() {
		r.GET(`/news/{category_id}/{new_id:(\d+)}/{tid:(\d+)}/detail`, emptyHandler)
	})
}

func TestOptionalRoute(t *testing.T) {
	is := assert.New(t)

	r := New()
	is.NotEmpty(r)

	// invalid
	is.Panics(func() {
		r.Add("/blog[/{category}]/{id}", emptyHandler, GET)
	})

	// simple
	r.Add("/about[.html]", emptyHandler, GET)

	ret := r.Match(GET, "about")
	is.Equal(Found, ret.Status)
	ret = r.Match(GET, "/about")
	is.Equal(Found, ret.Status)
	ret = r.Match(GET, "/about.html")
	is.Equal(Found, ret.Status)

	// with Params
	r.Add("/blog[/{category}]", emptyHandler, GET)

	ret = r.Match(GET, "/blog")
	is.Equal(Found, ret.Status)
	ret = r.Match(GET, "/blog/golang")
	is.Equal(Found, ret.Status)

	r = New()
	r.GET("/[{invite_name}]", emptyHandler)
	ret = r.Match(GET, "/")
	is.Equal(Found, ret.Status)
	ret = r.Match(GET, "/blog")
	is.Equal(Found, ret.Status)
}

func TestMethodNotAllowed(t *testing.T) {
	is := assert.New(t)

	// enable handle not allowed
	r := New(HandleMethodNotAllowed)
	is.True(r.handleMethodNotAllowed)

	r.Add("/path/some", emptyHandler)
	r.Add("/path/{var}", emptyHandler, PUT)
	r.Add("/path[/{var}]", emptyHandler, DELETE)

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
	is.Equal(1, r.cachedRoutes.Len())

	for id := range []int{19: 0} {
		idStr := fmt.Sprint(id)
		w = mockRequest(r, "GET", "/users/"+idStr, nil)
		is.Equal("id:"+idStr, w.Body.String())
	}

	// Option: MaxMultipisMemory 8M
	// r = New(MaxMultipisMemory(8 << 20))
	// is.Equal(8 << 20, r.maxMultipisMemory)
}

func TestAccessStaticAssets(t *testing.T) {
	r := New()
	is := assert.New(t)

	checkJsAssetHeader := func(contentType string) {
		if envutil.IsWin() {
			is.Equal("text/plain; charset=utf-8", contentType)
		} else {
			is.Equal("application/javascript", contentType)
		}
	}

	// one file
	r.StaticFile("/site.js", "testdata/site.js")
	w := mockRequest(r, "GET", "/site.js", nil)
	is.Equal(200, w.Code)

	checkJsAssetHeader(w.Header().Get("Content-Type"))

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

	checkJsAssetHeader(w.Header().Get("Content-Type"))

	is.Contains(w.Body.String(), "console.log")
	w = mockRequest(r, "GET", "/static/site.md", nil)
	is.Equal(200, w.Code)

	// add file type limit
	// r.StaticFiles("", "testdata", "css|js")
	r.StaticFiles("/assets", "testdata", "css|js")
	w = mockRequest(r, "GET", "/assets/site.js", nil)
	is.Equal(200, w.Code)

	checkJsAssetHeader(w.Header().Get("Content-Type"))

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

func TestResetful(t *testing.T) {
	var methodOverride = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				om := r.Header.Get("X-HTTP-Method-Override")

				// only allow: PUT, PATCH or DELETE.
				if om == "PUT" || om == "PATCH" || om == "DELETE" {
					r.Method = om
				}
			}

			h.ServeHTTP(w, r)
		})
	}

	product := &Product{}

	// Debug(true)
	r := New()
	// test StrictLastSlash option
	// r := New(StrictLastSlash)
	is := assert.New(t)

	h := methodOverride(r)

	r.Resource("/", product)
	w := mockRequest(r, "GET", "/product", nil)
	is.Equal(w.Body.String(), "GET Index")
	w = mockRequest(r, "GET", "/product/create", nil)
	is.Equal(w.Body.String(), "GET Create")
	w = mockRequest(r, "GET", "/product/123456", nil)
	is.Equal(w.Body.String(), "GET Show 123456")
	w = mockRequest(r, "GET", "/product/123456/edit", &md{H: m{"Authorization": fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("test:123")))}})
	is.Equal(w.Body.String(), "GET Edit 123456")
	w = mockRequest(r, "POST", "/product", nil)
	is.Equal(w.Body.String(), "POST Store")
	w = mockRequest(h, "POST", "/product/123456", &md{H: m{"X-HTTP-Method-Override": "PUT"}})
	is.Equal(w.Body.String(), "PUT Update 123456")
	w = mockRequest(h, "POST", "/product/123456", &md{H: m{"X-HTTP-Method-Override": "PATCH"}})
	is.Equal(w.Body.String(), "PATCH Update 123456")
	w = mockRequest(h, "POST", "/product/123456", &md{H: m{"X-HTTP-Method-Override": "DELETE"}})
	is.Equal(w.Body.String(), "DELETE Delete 123456")

	resPaincPtr := Product{}
	r = New()

	is.PanicsWithValue("controller must type ptr", func() {
		r.Resource("/", resPaincPtr)
	})

	resPaincString := "test"
	r = New()

	is.PanicsWithValue("controller must type struct", func() {
		r.Resource("/", &resPaincString)
	})
}

func TestGetRoutes(t *testing.T) {
	r := New()
	is := assert.New(t)

	r.GET("/homepage", func(c *Context) {}).NamedTo("homepage", r)
	r.GET("/users/{id}", func(c *Context) {}, func(c *Context) {
		c.Next()
	}, func(c *Context) {
		c.Next()
	}).NamedTo("users_id", r)
	r.GET("/news/{id}", func(c *Context) {}, func(c *Context) {
		c.Next()
	}).NamedTo("news_id", r)

	is.Len(r.NamedRoutes(), 3)

	//for _, r := range r.Routes() {
	//	fmt.Printf("%#v\n\n", r)
	//}
}
