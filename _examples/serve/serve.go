package main

import (
	"fmt"
	"github.com/gookit/sux"
	"github.com/gookit/sux/handlers"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// go run ./_examples/serve/serve.go
func main() {
	// open debug
	sux.Debug(true)

	r := sux.New()

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

	r.Use(handlers.RequestLogger(), sux.WarpHttpHandler(gh))

	r.GET("/", func(c *sux.Context) {
		c.Text(200, "hello " + c.URL().Path)
	}).Use(handlers.HTTPBasicAuth(map[string]string{
		// "test": "123",
	}))
	r.GET("/routes", handlers.DumpRoutesHandler())
	r.GET("/about[.html]", defHandle)
	r.GET("/hi-{name}", defHandle)
	r.GET("/users/{id}", func(c *sux.Context) {
		c.Text(200, "hello " + c.URL().Path)
	})
	r.POST("/post", func(c *sux.Context) {
		c.Text(200, "hello " + c.URL().Path)
	})
	r.Group("/articles", func() {
		r.GET("", func(c *sux.Context) {
			c.Text(200, "view list")
		})
		r.POST("", func(c *sux.Context) {
			c.Text(200, "create ok")
		})
		r.GET(`/{id:\d+}`, func(c *sux.Context) {
			c.Text(200, "view detail, id: " + c.Param("id"))
		})
	})

	// a simple proxy
	// proxy := proxy("http://yzone.net/page/about-me")
	pxy := newProxy("https://inhere.github.io/")
	r.GET("/pxy", func(c *sux.Context) {
		pxy.ServeHTTP(c.Resp, c.Req)
	})

	// use middleware for the route
	route := r.GET("/middle", func(c *sux.Context) { // main handler
		c.WriteString("-O-")
	}, func(c *sux.Context) { // middle 1
		c.WriteString("a")
		c.Next() // Notice: call Next()
		c.WriteString("A")
		// if call Abort(), will abort at the end of this middleware run
		// c.Abort()
	})
	// add by Use()
	route.Use(func(c *sux.Context) { // middle 2
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
	r := sux.New()

	// add routes
	r.GET("/", func(ctx *sux.Context) {
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

func defHandle(ctx *sux.Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
}

// SiteController define a controller
type SiteController struct {
}

// AddRoutes for the controller
func (c *SiteController) AddRoutes(r *sux.Router) {
	r.GET("{id}", c.Get)
	r.POST("", c.Post)
}

// Get action
func (c *SiteController) Get(ctx *sux.Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
	ctx.WriteString("\n ok")
}

// Post action
func (c *SiteController) Post(ctx *sux.Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
}

// BlogController define a controller
type BlogController struct {
}

// AddRoutes for the controller
func (c *BlogController) AddRoutes(r *sux.Router) {
	r.GET("{id}", c.Get)
	r.POST("", c.Post)
}

// Get action
func (c *BlogController) Get(ctx *sux.Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
	ctx.WriteString("\nok")
}

// Post action
func (c *BlogController) Post(ctx *sux.Context) {
	ctx.Text(200, "hello, in " + ctx.URL().Path)
}