package main

import (
	"github.com/gookit/sux"
)

// go run ./_examples/serve.go
func main() {
	r := sux.New()
	r.GET("/", func(c *sux.Context) {
		c.Text(200, "hello " + c.URL().Path)
	})
	r.GET("/users/{id}", func(c *sux.Context) {
		c.Text(200, "hello " + c.URL().Path)
	})
	r.POST("/post", func(c *sux.Context) {
		c.Text(200, "hello " + c.URL().Path)
	})
	r.Group("/articles", func(g *sux.Router) {
		g.GET("", func(c *sux.Context) {
			c.Text(200, "view list")
		})
		g.POST("", func(c *sux.Context) {
			c.Text(200, "create ok")
		})
		g.GET(`/{id:\d+}`, func(c *sux.Context) {
			c.Text(200, "view detail, id: " + c.Param("id"))
		})
	})

	r.Controller("/blog", &BlogController{})

	r.Listen(":18080")
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
	ctx.WriteString("\n ok")
}

// Post action
func (c *BlogController) Post(ctx *sux.Context) {
	ctx.Text(200, "hello, in " + ctx.URL().Path)
}