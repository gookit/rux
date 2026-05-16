package rux

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gookit/goutil/netutil/httpctype"
	"github.com/gookit/goutil/testutil"
	"github.com/gookit/goutil/testutil/assert"
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

	m1 := r.Match("GET", "/")
	fmt.Println(m1.Route.Path())
	m2 := r.Match("GET", "/users/23")
	fmt.Println(m2.Route.Path(), m2.Params)

	// run http server
	// r.Listen(":8080")

	// Output:
	// /
	// /users/{id} map[id:23]
}

var emptyHandler = func(c *Context) {}

// SiteController define a controller
type SiteController struct {
}

func (c *SiteController) AddRoutes(r *Router) {
	r.GET("{id}", c.Get)
	r.GET("", c.Index)
	r.POST("", c.Post)
	r.GET("about", c.About)
}

func (c *SiteController) MappingRoutes(r *Router) map[string]HandlerFunc {
	// r.GET("", c.Index)

	return map[string]HandlerFunc{
		"/ GET,POST": c.Index,
		"/about GET": c.About,
		// "GET" short as "/detail GET"
		"GET": c.Detail,
	}
}

func (c *SiteController) Index(ctx *Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
}

func (c *SiteController) Detail(ctx *Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
}

func (c *SiteController) About(ctx *Context) {
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
func (c *Product) invaid2(_ *Context) {
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

	route = r.Match("GET", "/get").Route
	is.NotEmpty(route)
	is.Len(route.Handlers(), 1)
}

func TestSimpleMatch(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.GET("/", func(c *Context) {
		_, _ = c.Resp.Write([]byte("Welcome!\n"))
	})

	r.GET("/user/{id}", func(c *Context) {
		c.WriteString(c.Param("id"))
	})

	route := r.Match("GET", "/").Route
	is.NotEmpty(route)

	route = r.Match("GET", "/user/42").Route
	is.NotEmpty(route)
}

func TestAddRoute(t *testing.T) {
	is := assert.New(t)

	Debug(true)

	r := New()
	is.NotEmpty(r)

	// no handler
	is.PanicsMsg(func() {
		r.GET("/get", nil)
	}, "the route handler cannot be empty.(path: '/get')")

	// invalid method
	is.PanicsMsg(func() {
		r.Add("/get", emptyHandler, "invalid")
	}, "invalid method name 'INVALID', must in: "+MethodsString())

	// empty method
	is.PanicsMsg(func() {
		r.AddRoute(&Route{path: "/", handler: emptyHandler})
	}, "the route allowed methods cannot be empty.(path: '/')")

	// overflow max num of the route handlers
	is.PanicsMsg(func() {
		var i int8 = -1
		var hs HandlersChain
		for ; i <= abortIndex; i++ {
			hs = append(hs, emptyHandler)
		}

		r.GET("/overflow", emptyHandler, hs...)
	}, fmt.Sprintf("too many handlers(number: %d)", int(abortIndex)+2))

	route := r.GET("/get", namedHandler)
	is.NotEmpty(route.Handler())
	is.Eq("/get", route.path)
	// is.Eq(fmt.Sprint(*namedHandler), route.Handler())
	is.Eq("github.com/gookit/rux.namedHandler", route.HandlerName())

	route = r.Match("GET", "/get").Route
	is.NotEmpty(route)
	is.NotEmpty(route.Handler())

	route = r.Match(HEAD, "/get").Route
	is.NotEmpty(route)

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
		route = r.Match(m, "/"+strings.ToLower(m)).Route
		is.NotEmpty(route)
	}

	r.Any("/any", emptyHandler)
	for _, m := range anyMethods {
		route = r.Match(m, "/any").Route
		is.NotEmpty(route)
	}

	route = r.Match(GET, "/not-exist").Route
	is.Nil(route)

	// add a controller
	r.Controller("/site", &SiteController{})
	route = r.Match(GET, "/site/12").Route
	is.NotEmpty(route)
	is.Eq("/site/{id}", route.Path())
	route = r.Match(POST, "/site").Route
	is.NotEmpty(route)
	is.Eq("/site", route.Path())

	Debug(false)
}

func TestHandleFallbackRoute(t *testing.T) {
	is := assert.New(t)
	r := New()

	var route *Route

	// fallback route(Need enable option: r.handleFallbackRoute)
	r.Any("/*", emptyHandler)
	for _, m := range AllMethods() {
		route = r.Match(m, "/not-exist").Route
		is.Nil(route)
	}

	r = New(HandleFallbackRoute)
	// add fallback route
	r.Any("/*", emptyHandler)
	for _, m := range AllMethods() {
		route = r.Match(m, "/not-exist").Route
		is.NotEmpty(route)
	}
}

func TestNameRoute(t *testing.T) {
	is := assert.New(t)
	r := New()

	// named route
	r.GET("/path1", emptyHandler).NamedTo("route1", r)

	r2 := r.AddNamed("route2", "/path2[.html]", emptyHandler, POST)
	is.Eq("route2", r2.Name())

	r3 := NamedRoute("route3", "/path3/{id}", emptyHandler, "get")
	r3.AttachTo(r)

	r4 := NewNamedRoute("route4", "/path4/some/{id}", emptyHandler, GET)
	r.AddRoute(r4)

	r5 := NamedRoute("route5", "/path5/some/{id}", emptyHandler, PUT)
	r.AddRoute(r5)

	is.Len(r.Routes(), 5)

	route := r.GetRoute("route1")
	is.NotEmpty(route)
	is.Eq("/path1", route.Path())
	is.Eq("GET", route.MethodString(""))
	is.Eq([]string{"GET"}, route.Methods())

	info := route.Info()
	is.Eq("/path1", info.Path)
	is.Eq([]string{"GET"}, info.Methods)

	route = r.GetRoute("route2")
	is.NotEmpty(route)
	is.Eq(route, r2)
	is.Eq("route2", route.Name())

	route = r.GetRoute("route3")
	is.NotEmpty(route)
	is.Eq(route, r3)

	route = r.GetRoute("route4")
	is.NotEmpty(route)
	is.Eq(route, r4)

	route = r.GetRoute("route5")
	is.NotEmpty(route)
	is.Eq(route, r5)

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

	route := r.Match(GET, "/users").Route
	is.NotEmpty(route)
	is.Len(route.Handlers(), 1)

	route = r.Match(GET, "/users/23").Route
	is.NotEmpty(route)
	is.Len(route.Handlers(), 1)

	// overflow max num of the route handlers
	is.PanicsMsg(func() {
		var i int8 = -1
		var hs HandlersChain
		for ; i <= abortIndex; i++ {
			hs = append(hs, emptyHandler)
		}

		r.Group("/test", func() {
			r.GET("", emptyHandler)
			r.GET("/{id}", emptyHandler)
		}, hs...)
	}, fmt.Sprintf("too many handlers(number: %d)", int(abortIndex)+2))
}

func TestRouter_Controller(t *testing.T) {
	is := assert.New(t)
	r := New()
	Debug(true)

	// root path
	r.Controller("/", &SiteController{})

	w := testutil.MockRequest(r, http.MethodGet, "/", nil)
	is.Eq(200, w.Code)
	is.Eq("hello, in /", w.Body.String())

	w = testutil.MockRequest(r, http.MethodGet, "", nil)
	is.Eq(200, w.Code)
	is.Eq("hello, in ", w.Body.String())

	Debug(false)
}

func TestRouter_Controller2(t *testing.T) {
	is := assert.New(t)
	r := New()

	// empty path
	r.Controller("", &SiteController{})

	w := testutil.MockRequest(r, http.MethodGet, "/", nil)
	is.Eq(200, w.Code)
	is.Eq("hello, in /", w.Body.String())

	w = testutil.MockRequest(r, http.MethodGet, "", nil)
	is.Eq(200, w.Code)
	is.Eq("hello, in ", w.Body.String())
}

func TestDynamicRoute(t *testing.T) {
	is := assert.New(t)

	r := New()
	is.NotEmpty(r)

	r.GET("/users/{id}", emptyHandler)

	res := r.Match(GET, "/users/23")
	route, ps := res.Route, res.Params
	is.NotEmpty(route)
	is.Len(ps, 1)
	is.False(ps.Has("no-key"))
	is.True(ps.Has("id"))
	// get param
	is.Eq("23", ps["id"])
	is.Eq("", ps.String("no-key"))
	is.Eq("23", ps.String("id"))
	is.Eq(23, ps.Int("id"))
	is.Eq(0, ps.Int("no-key"))

	res = r.Match(GET, "/users/str")
	route, ps = res.Route, res.Params
	is.NotEmpty(route)
	is.Eq("str", ps["id"])

	route = r.Match(GET, "/not/exist").Route
	is.Nil(route)

	r.GET("/site/settings/{id}", emptyHandler)
	route = r.Match(GET, "/site/exist").Route
	is.Nil(route)

	// add regex for var - note: regex is stripped, all segments match
	r.GET(`/path1/{id:[1-9]\d*}`, emptyHandler)
	route = r.Match(GET, "/path1/23").Route
	is.NotEmpty(route)
	// regex filtering is no longer supported; any segment matches
	route = r.Match(GET, "/path1/err").Route
	is.NotEmpty(route)

	// use param var
	r.GET(`/path2/{num}`, emptyHandler)
	route = r.Match(GET, "/path2/23").Route
	is.NotEmpty(route)
	// without regex filtering, all segments match
	route = r.Match(GET, "/path2/-23").Route
	is.NotEmpty(route)
	route = r.Match(GET, "/path2/err").Route
	is.NotEmpty(route)

	r.GET(`/path3/{level:[1-9]{1,2}}`, emptyHandler)
	res = r.Match(GET, "/path3/2")
	route, ps = res.Route, res.Params
	is.NotEmpty(route)
	is.True(ps.Has("level"))
	is.Eq("2", ps.String("level"))
	// regex filtering is no longer supported; any segment matches
	route = r.Match(GET, "/path3/123").Route
	is.NotEmpty(route)

	// wildcard file param (regex stripped to param)
	r.GET(`/assets/{file:.+\.(?:css|js)}`, emptyHandler)
	route = r.Match(GET, "/assets/site.css").Route
	is.NotEmpty(route)
	route = r.Match(GET, "/assets/site.js").Route
	is.NotEmpty(route)
	// regex filtering is no longer supported; any segment matches
	route = r.Match(GET, "/assets/site.tx").Route
	is.NotEmpty(route)
}

func TestFixFirstNodeOnlyOneChar(t *testing.T) {
	is := assert.New(t)

	r := New()
	r.PATCH(`/r/{name}/hq2hah9/dxt/g/hoovln`, emptyHandler)

	route := r.Match(PATCH, "/r/lnamel/hq2hah9/dxt/g/hoovln").Route
	is.NotEmpty(route)
}

func TestMultiPathParam(t *testing.T) {
	ris := assert.New(t)

	r := New()
	r.PATCH(`/news/{category_id}/{new_id:\d+}/detail`, emptyHandler)

	res := r.Match(PATCH, "/news/100/20/detail")
	route, ps := res.Route, res.Params
	ris.NotEmpty(route)
	ris.Len(ps, 2)
	ris.True(ps.Has("category_id"))
	ris.Eq(100, ps.Int("category_id"))
	ris.True(ps.Has("new_id"))
	ris.Eq(20, ps.Int("new_id"))

	r.GET(`/news/{category_id}/{new_id:\d+}/{tid:\d+}/detail`, emptyHandler)

	res = r.Match(GET, "/news/100/20/10/detail")
	route, ps = res.Route, res.Params
	ris.NotEmpty(route)
	ris.Len(ps, 3)
	ris.True(ps.Has("category_id"))
	ris.True(ps.Has("new_id"))
	ris.True(ps.Has("tid"))
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

	route := r.Match(GET, "about").Route
	is.NotEmpty(route)
	route = r.Match(GET, "/about").Route
	is.NotEmpty(route)
	route = r.Match(GET, "/about.html").Route
	is.NotEmpty(route)

	// with Params
	r.Add("/blog[/{category}]", emptyHandler, GET)

	route = r.Match(GET, "/blog").Route
	is.NotEmpty(route)
	route = r.Match(GET, "/blog/golang").Route
	is.NotEmpty(route)

	r = New()
	r.GET("/[{invite_name}]", emptyHandler)
	route = r.Match(GET, "/").Route
	is.NotEmpty(route)
	route = r.Match(GET, "/blog").Route
	is.NotEmpty(route)
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

	route := r.Match(GET, "/path/some").Route
	is.NotEmpty(route)

	res := r.Match(POST, "/path/some")
	route, allowed := res.Route, res.Allowed
	is.Nil(route)
	is.Len(allowed, 3)

	is.Contains(allowed, "GET")
	is.Contains(allowed, "PUT")
	is.Contains(allowed, "DELETE")
}

func TestOther(t *testing.T) {
	is := assert.New(t)

	SetGlobalVar("name", `\w+`)
	m := GetGlobalVars()
	is.Eq(`\w+`, m["name"])

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

	// Options: StrictLastSlash
	r := New(StrictLastSlash)
	is.True(r.strictLastSlash)

	r.GET("/users", func(c *Context) {
		c.Text(200, "val0")
	})
	r.GET("/users/", func(c *Context) {
		c.Text(200, "val1")
	})

	w := mockRequest(r, "GET", "/users", nil)
	is.Eq("val0", w.Body.String())
	w = mockRequest(r, "GET", "/users/", nil)
	is.Eq("val1", w.Body.String())

	// Options: UseEncodedPath
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
	is.Eq("val1", w.Body.String())
	w = mockRequest(r, "GET", "/users/with spaces", nil)
	is.Eq("val1", w.Body.String())

	// Options: InterceptAll
	r = New(InterceptAll("/coming-soon"))
	// Notice: must add a route and path equals to 'InterceptAll' and use Any()
	r.Any("/coming-soon", func(c *Context) {
		c.Text(200, "coming-soon")
	})
	r.GET("/users", func(c *Context) {
		c.Text(200, "val0")
	})

	w = mockRequest(r, "GET", "/users", nil)
	is.Eq("coming-soon", w.Body.String())
	w = mockRequest(r, "GET", "/not-exist", nil)
	is.Eq("coming-soon", w.Body.String())
	w = mockRequest(r, "POST", "/not-exist", nil)
	is.Eq("coming-soon", w.Body.String())

	// Options: MaxMultipisMemory 8M
	// r = New(MaxMultipisMemory(8 << 20))
	// is.Eq(8 << 20, r.maxMultipisMemory)
}

func TestAccessStaticAssets(t *testing.T) {
	r := New()
	is := assert.New(t)
	// gov := runtime.Version()[2:]

	checkJsAssetHeader := func(contentType string) {
		// new go version has been fixed
		// win: application/javascript
		// lin: text/javascript; charset=utf-8
		is.Contains(contentType, "javascript")
	}

	// one file
	r.StaticFile("/site.js", "testdata/site.js")
	w := mockRequest(r, "GET", "/site.js", nil)
	is.Eq(200, w.Code)

	checkJsAssetHeader(w.Header().Get("Content-Type"))

	is.Contains(w.Body.String(), "console.log")
	// try again
	w = mockRequest(r, "GET", "/site.js?t=33455", nil)
	is.Eq(200, w.Code)

	// allow any files in the dir.
	r.StaticDir("/static", "testdata")
	w = mockRequest(r, "GET", "/static/site.css", nil)
	is.Eq(200, w.Code)
	is.Eq("text/css; charset=utf-8", w.Header().Get("Content-Type"))
	is.Contains(w.Body.String(), "max-width")
	w = mockRequest(r, "GET", "/static/site.js", nil)
	is.Eq(200, w.Code)

	checkJsAssetHeader(w.Header().Get("Content-Type"))

	is.Contains(w.Body.String(), "console.log")
	w = mockRequest(r, "GET", "/static/site.md", nil)
	is.Eq(200, w.Code)

	// add file type limit
	// r.StaticFiles("", "testdata", "css|js")
	r.StaticFiles("/assets", "testdata", "css|js")
	w = mockRequest(r, "GET", "/assets/site.js", nil)
	is.Eq(200, w.Code)

	checkJsAssetHeader(w.Header().Get("Content-Type"))

	is.Contains(w.Body.String(), "console.log")
	// extension filtering is no longer supported; file is served if it exists
	w = mockRequest(r, "GET", "/assets/site.md", nil)
	is.Eq(200, w.Code)

	// StaticFunc
	r.StaticFunc("/some/test.txt", func(c *Context) {
		c.Text(200, "content")
	})
	w = mockRequest(r, "GET", "/some/test.txt", nil)
	is.Eq(200, w.Code)
	is.Eq(httpctype.Text, w.Header().Get(httpctype.Key))
	is.Contains(w.Body.String(), "content")
}

func TestRestFul(t *testing.T) {
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
	is.Eq(w.Body.String(), "GET Index")
	w = mockRequest(r, "GET", "/product/create", nil)
	is.Eq(w.Body.String(), "GET Create")
	w = mockRequest(r, "GET", "/product/123456", nil)
	is.Eq(w.Body.String(), "GET Show 123456")
	w = mockRequest(r, "GET", "/product/123456/edit", &md{
		H: m{"Authorization": fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("test:123")))},
	})
	is.Eq(w.Body.String(), "GET Edit 123456")
	w = mockRequest(r, "POST", "/product", nil)
	is.Eq(w.Body.String(), "POST Store")
	w = mockRequest(h, "POST", "/product/123456", &md{H: m{"X-HTTP-Method-Override": "PUT"}})
	is.Eq(w.Body.String(), "PUT Update 123456")
	w = mockRequest(h, "POST", "/product/123456", &md{H: m{"X-HTTP-Method-Override": "PATCH"}})
	is.Eq(w.Body.String(), "PATCH Update 123456")
	w = mockRequest(h, "POST", "/product/123456", &md{H: m{"X-HTTP-Method-Override": "DELETE"}})
	is.Eq(w.Body.String(), "DELETE Delete 123456")

	r = New()
	is.PanicsMsg(func() {
		r.Resource("/", Product{})
	}, "controller must type ptr")

	resPanicString := "test"
	is.PanicsMsg(func() {
		r.Resource("/", &resPanicString)
	}, "controller must type struct")
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

	// for _, r := range r.Routes() {
	//	fmt.Printf("%#v\n\n", r)
	// }
}
