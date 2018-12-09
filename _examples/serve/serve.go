package main

import (
	"fmt"
	"github.com/gookit/rux"
	"github.com/gookit/rux/handlers"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// go run ./_examples/serve/serve.go
func main() {
	// open debug
	rux.Debug(true)

	r := rux.New()

	// one file
	r.StaticFile("/site.js", "testdata/site.js")
	// allow any files in the dir.
	r.StaticDir("/static", "testdata")
	// add file type limit
	// r.StaticFiles("", "testdata", "css|js")
	r.StaticFiles("/assets", "testdata", "css|js")

	gh := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("new-key", "val")
	})

	r.Use(handlers.RequestLogger(), rux.WrapHTTPHandler(gh))

	r.GET("/", func(c *rux.Context) {
		c.Text(200, "hello " + c.URL().Path)
	}).Use(handlers.HTTPBasicAuth(map[string]string{
		// "test": "123",
	}))
	r.GET("/routes", handlers.DumpRoutesHandler())
	r.GET("/about[.html]", defHandle)
	r.GET("/hi-{name}", defHandle).NamedTo("my-route", r)
	r.GET("/users/{id}", func(c *rux.Context) {
		c.Text(200, "hello " + c.URL().Path)
	})
	r.POST("/post", func(c *rux.Context) {
		c.Text(200, "hello " + c.URL().Path)
	})
	r.Group("/articles", func() {
		r.GET("", func(c *rux.Context) {
			c.Text(200, "view list")
		})
		r.POST("", func(c *rux.Context) {
			c.Text(200, "create ok")
		})
		r.GET(`/{id:\d+}`, func(c *rux.Context) {
			c.Text(200, "view detail, id: " + c.Param("id"))
		})
	})

	// a simple proxy
	// proxy := proxy("http://yzone.net/page/about-me")
	pxy := newProxy("https://inhere.github.io/")
	r.GET("/pxy", func(c *rux.Context) {
		pxy.ServeHTTP(c.Resp, c.Req)
	})

	// use middleware for the route
	route := r.GET("/middle", func(c *rux.Context) { // main handler
		c.WriteString("-O-")
	}, func(c *rux.Context) { // middle 1
		c.WriteString("a")
		c.Next() // Notice: call Next()
		c.WriteString("A")
		// if call Abort(), will abort at the end of this middleware run
		// c.Abort()
	})
	// add by Use()
	route.Use(func(c *rux.Context) { // middle 2
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})

	r.Controller("/blog", &BlogController{})

	fmt.Println(r)

	// quick start
	r.Listen(":18080")

	// apply pre-handlers
	// http.ListenAndServe(":18080", handlers.HTTPMethodOverrideHandler(r))
}

func customServer() {
	r := rux.New()

	// add routes
	r.GET("/", func(ctx *rux.Context) {
		ctx.WriteString("hello")
	})

	s := &http.Server{
		Addr:    ":8080",
		Handler: r,

		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}

func newProxy(targetUrl string) *httputil.ReverseProxy {
	target, _ := url.Parse(targetUrl)
	// target, _ := url.Parse(playgroundURL)

	p := httputil.NewSingleHostReverseProxy(target)
	// p.Transport = &urlfetch.Transport{Context: appengine.NewContext(r)}
	// p.ServeHTTP(w, r)

	return p
}

func defHandle(ctx *rux.Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
}

// SiteController define a controller
type SiteController struct {
}

// AddRoutes for the controller
func (c *SiteController) AddRoutes(r *rux.Router) {
	r.GET("{id}", c.Get)
	r.POST("", c.Post)
}

// Get action
func (c *SiteController) Get(ctx *rux.Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
	ctx.WriteString("\n ok")
}

// Post action
func (c *SiteController) Post(ctx *rux.Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
}

// BlogController define a controller
type BlogController struct {
}

// AddRoutes for the controller
func (c *BlogController) AddRoutes(r *rux.Router) {
	r.GET("{id}", c.Get)
	r.POST("", c.Post)
}

// Get action
func (c *BlogController) Get(ctx *rux.Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
	ctx.WriteString("\nok")
}

// Post action
func (c *BlogController) Post(ctx *rux.Context) {
	ctx.Text(200, "hello, in " + ctx.URL().Path)
}