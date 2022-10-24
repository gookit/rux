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

	route, _, _ := r.Match("GET", "/")
	fmt.Println(route.Path())
	route, params, _ := r.Match("GET", "/users/23")
	fmt.Println(route.Path(), params)

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

	route, _, _ = r.Match("GET", "/get")
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

	route, _, _ := r.Match("GET", "/")
	is.NotEmpty(route)

	route, _, _ = r.Match("GET", "/user/42")
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
	}, "too many handlers(number: 65)")

	route := r.GET("/get", namedHandler)
	is.NotEmpty(route.Handler())
	is.Eq("/get", route.path)
	// is.Eq(fmt.Sprint(*namedHandler), route.Handler())
	is.Eq("github.com/gookit/rux.namedHandler", route.HandlerName())

	route, _, _ = r.Match("GET", "/get")
	is.NotEmpty(route)
	is.NotEmpty(route.Handler())

	route, _, _ = r.Match(HEAD, "/get")
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
		route, _, _ = r.Match(m, "/"+strings.ToLower(m))
		is.NotEmpty(route)
	}

	r.Any("/any", emptyHandler)
	for _, m := range anyMethods {
		route, _, _ = r.Match(m, "/any")
		is.NotEmpty(route)
	}

	route, _, _ = r.Match(GET, "/not-exist")
	is.Nil(route)

	// add a controller
	r.Controller("/site", &SiteController{})
	route, _, _ = r.Match(GET, "/site/12")
	is.NotEmpty(route)
	is.Eq("/site/{id}", route.Path())
	route, _, _ = r.Match(POST, "/site")
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
		route, _, _ = r.Match(m, "/not-exist")
		is.Nil(route)
	}

	r = New(HandleFallbackRoute)
	// add fallback route
	r.Any("/*", emptyHandler)
	for _, m := range AllMethods() {
		route, _, _ = r.Match(m, "/not-exist")
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
	_, ok := route.match("/path2")
	is.True(ok)

	route = r.GetRoute("route3")
	is.NotEmpty(route)
	is.Eq(route, r3)
	is.Eq("", route.start)

	route = r.GetRoute("route4")
	is.NotEmpty(route)
	is.Eq(route, r4)
	is.Eq("/path4/some/", route.start)

	route = r.GetRoute("route5")
	is.NotEmpty(route)
	is.Eq(route, r5)
	is.Eq("/path5/some/", route.start)

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

	route, _, _ := r.Match(GET, "/users")
	is.NotEmpty(route)
	is.Len(route.Handlers(), 1)

	route, _, _ = r.Match(GET, "/users/23")
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
	}, "too many handlers(number: 65)")
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

	r0 := r.GET("/users/{id}", emptyHandler)
	is.Eq("", r0.start)

	route, ps, _ := r.Match(GET, "/users/23")
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

	route, ps, _ = r.Match(GET, "/users/str")
	is.NotEmpty(route)
	is.Eq("str", ps["id"])
	route, _, _ = r.Match(GET, "/not/exist")
	is.Nil(route)

	r1 := r.GET("/site/settings/{id}", emptyHandler)
	route, _, _ = r.Match(GET, "/site/exist")
	is.Nil(route)

	// test start check.
	is.Eq("/site/settings/", r1.start)
	ps, ok := r1.match("/get")
	is.False(ok)
	is.Nil(ps)

	// add regex for var
	r.GET(`/path1/{id:[1-9]\d*}`, emptyHandler)
	route, _, _ = r.Match(GET, "/path1/23")
	is.NotEmpty(route)
	route, _, _ = r.Match(GET, "/path1/err")
	is.Nil(route)

	// use internal var
	r.GET(`/path2/{num}`, emptyHandler)
	route, _, _ = r.Match(GET, "/path2/23")
	is.NotEmpty(route)
	route, _, _ = r.Match(GET, "/path2/-23")
	is.Nil(route)
	route, _, _ = r.Match(GET, "/path2/err")
	is.Nil(route)

	r.GET(`/path3/{level:[1-9]{1,2}}`, emptyHandler)
	route, ps, _ = r.Match(GET, "/path3/2")
	is.NotEmpty(route)
	is.True(ps.Has("level"))
	is.Eq("2", ps.String("level"))
	route, _, _ = r.Match(GET, "/path3/123")
	is.Nil(route)

	r.GET(`/assets/{file:.+\.(?:css|js)}`, emptyHandler)
	route, _, _ = r.Match(GET, "/assets/site.css")
	is.NotEmpty(route)
	route, _, _ = r.Match(GET, "/assets/site.js")
	is.NotEmpty(route)
	route, _, _ = r.Match(GET, "/assets/site.tx")
	is.Nil(route)
}

func TestFixFirstNodeOnlyOneChar(t *testing.T) {
	is := assert.New(t)

	r := New()
	r.PATCH(`/r/{name}/hq2hah9/dxt/g/hoovln`, emptyHandler)

	route, _, _ := r.Match(PATCH, "/r/lnamel/hq2hah9/dxt/g/hoovln")
	is.NotEmpty(route)
}

func TestMultiPathParam(t *testing.T) {
	ris := assert.New(t)

	r := New()
	r.PATCH(`/news/{category_id}/{new_id:\d+}/detail`, emptyHandler)

	route, ps, _ := r.Match(PATCH, "/news/100/20/detail")
	ris.NotEmpty(route)
	ris.Len(ps, 2)
	ris.True(ps.Has("category_id"))
	ris.Eq(100, ps.Int("category_id"))
	ris.True(ps.Has("new_id"))
	ris.Eq(20, ps.Int("new_id"))

	r2 := r.GET(`/news/{category_id}/{new_id:\d+}/{tid:\d+}/detail`, emptyHandler)
	ris.Eq("/news/{category_id}/{new_id}/{tid}/detail", r2.spath)

	route, ps, _ = r.Match(GET, "/news/100/20/10/detail")
	ris.NotEmpty(route)
	ris.Len(ps, 3)
	ris.True(ps.Has("category_id"))
	ris.True(ps.Has("new_id"))
	ris.True(ps.Has("tid"))

	ris.PanicsMsg(func() {
		r.GET(`/news/{category_id}/{new_id:(\d+)}/{tid:(\d+)}/detail`, emptyHandler)
	}, `invalid path var regex string, dont allow char '('. var: new_id, regex: (\d+)`)
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

	route, _, _ := r.Match(GET, "about")
	is.NotEmpty(route)
	route, _, _ = r.Match(GET, "/about")
	is.NotEmpty(route)
	route, _, _ = r.Match(GET, "/about.html")
	is.NotEmpty(route)

	// with Params
	r.Add("/blog[/{category}]", emptyHandler, GET)

	route, _, _ = r.Match(GET, "/blog")
	is.NotEmpty(route)
	route, _, _ = r.Match(GET, "/blog/golang")
	is.NotEmpty(route)

	r = New()
	r.GET("/[{invite_name}]", emptyHandler)
	route, _, _ = r.Match(GET, "/")
	is.NotEmpty(route)
	route, _, _ = r.Match(GET, "/blog")
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

	route, _, _ := r.Match(GET, "/path/some")
	is.NotEmpty(route)

	route, _, allowed := r.Match(POST, "/path/some")
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
	is.Eq("val0", w.Body.String())
	w = mockRequest(r, "GET", "/users/", nil)
	is.Eq("val1", w.Body.String())

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
	is.Eq("val1", w.Body.String())
	w = mockRequest(r, "GET", "/users/with spaces", nil)
	is.Eq("val1", w.Body.String())

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
	is.Eq("coming-soon", w.Body.String())
	w = mockRequest(r, "GET", "/not-exist", nil)
	is.Eq("coming-soon", w.Body.String())
	w = mockRequest(r, "POST", "/not-exist", nil)
	is.Eq("coming-soon", w.Body.String())

	// Option: EnableCaching, MaxNumCaches
	r = New(EnableCaching, MaxNumCaches(10))
	simpleHandler := func(c *Context) {
		c.Text(200, "id:"+c.Param("id"))
	}
	r.GET("/users/{id}", simpleHandler)
	w = mockRequest(r, "GET", "/users/23", nil)
	is.Eq("id:23", w.Body.String())
	w = mockRequest(r, "GET", "/users/23", nil)
	is.Eq("id:23", w.Body.String())
	is.Eq(1, r.cachedRoutes.Len())

	for id := range []int{19: 0} {
		idStr := fmt.Sprint(id)
		w = mockRequest(r, "GET", "/users/"+idStr, nil)
		is.Eq("id:"+idStr, w.Body.String())
	}

	// Option: MaxMultipisMemory 8M
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
		// is.Eq("application/javascript", contentType)

		// if envutil.IsWin() {
		// 	// go > 1.17: "application/javascript"
		// 	if gov >= "1.17" {
		// 		is.Eq("application/javascript", contentType)
		// 	} else {
		// 		is.Eq("text/plain; charset=utf-8", contentType)
		// 	}
		// } else {
		// 	is.Contains(contentType, "javascript")
		//
		// 	// go < 1.17: "application/javascript"
		// 	// go >= 1.17: text/javascript; charset=utf-8
		// 	if gov >= "1.17" {
		// 		is.Eq("application/javascript", contentType)
		// 	} else {
		// 		is.Eq("text/javascript; charset=utf-8", contentType)
		// 	}
		// }
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
	w = mockRequest(r, "GET", "/assets/site.md", nil)
	is.Eq(404, w.Code)

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
