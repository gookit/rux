package main

import "github.com/gookit/rux"

// SiteController define a controller
type SiteController struct {
}

// AddRoutes for the controller
func (c *SiteController) AddRoutes(r *rux.Router) {
	r.GET("{id}", c.Get)
	r.POST("", c.Post)

	// mp := map[string]rux.HandlerFunc{
	// 	"get,{id}": c.Get,
	// }
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
	ctx.Text(200, "hello, in "+ctx.URL().Path)
}
