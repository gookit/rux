package main

import (
	"github.com/gookit/sux"
	"net/http"
	"github.com/gookit/sux/handlers"
)

// go run ./_examples/serve.go
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
	})
	r.GET("/routes", handlers.DumpRoutesHandler())
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

	// quick start
	r.Listen(":18080")

	// apply pre-handlers
	// http.ListenAndServe(":18080", handlers.HTTPMethodOverrideHandler(r))
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