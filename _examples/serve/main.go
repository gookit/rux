package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gookit/rux"
	"github.com/gookit/rux/handlers"
)

// start: go run ./_examples/serve
// access: http://127.0.0.1:18080
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

	gh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("new-key", "val")
	})

	r.Use(handlers.RequestLogger(), rux.WrapHTTPHandler(gh))

	r.GET("/", func(c *rux.Context) {
		c.Text(200, "hello "+c.URL().Path)
	})

	r.GET("/bauth", func(c *rux.Context) {
		c.Text(200, "hello "+c.URL().Path)
	}).Use(handlers.HTTPBasicAuth(map[string]string{
		"test": "123",
	}))

	r.GET("/routes", handlers.DumpRoutesHandler())
	r.GET("/about[.html]", defHandle)
	r.GET("/hi-{name}", defHandle).NamedTo("my-route", r)
	r.GET("/users/{id}", func(c *rux.Context) {
		c.Text(200, "hello "+c.URL().Path)
	})
	r.POST("/post", func(c *rux.Context) {
		c.Text(200, "hello "+c.URL().Path)
	})
	r.Group("/articles", func() {
		r.GET("", func(c *rux.Context) {
			c.Text(200, "view list")
		})
		r.POST("", func(c *rux.Context) {
			c.Text(200, "create ok")
		})
		r.GET(`/{id:\d+}`, func(c *rux.Context) {
			c.Text(200, "view detail, id: "+c.Param("id"))
		})
	})

	// add multi method support for an route path
	r.Add("/post[/{id}]", func(c *rux.Context) {
		if c.Param("id") == "" {
			// do create post
			c.Text(200, "created")
			return
		}

		id := c.Params.Int("id")
		// do update post
		c.Text(200, "updated "+fmt.Sprint(id))
	}, rux.POST, rux.PUT)

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
	r.Controller("/site", &SiteController{})

	// fmt.Println(r)

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
