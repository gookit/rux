package main

import "github.com/gookit/rux"

// SiteController define a controller
type SiteController struct {
}

// AddRoutes for the controller
func (c *SiteController) AddRoutes(r *rux.Router) {
	r.GET("{id}", c.Get)
	r.POST("", c.Post)
	r.GET("delcookie", c.DelCookie)
	r.GET("setcookie", c.SetCookie)
	r.GET("getcookie", c.GetCookie)

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

// SetCookie action
func (c *SiteController) SetCookie(ctx *rux.Context) {
	// NOTICE: if you write body before set headers or cookies, the set headers/cookies will invalid.
	// ctx.WriteString("hello, in " + ctx.URL().Path)
	ctx.SetHeader("rux-header", "header-value")
	ctx.SetCookie("rux_cookie", "test value", 3600, "/", ctx.Req.URL.Host, false, true)
	ctx.FastSetCookie("rux_cookie2", "test value", 3600)
	ctx.WriteString("hello, in " + ctx.URL().Path)
}

// DelCookie action
func (c *SiteController) DelCookie(ctx *rux.Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)
	ctx.SetCookie("rux_cookie", "", 0, "/", ctx.Req.URL.Host, true, true)
	ctx.FastSetCookie("rux_cookie2", "", 0)
}

// GetCookie action
func (c *SiteController) GetCookie(ctx *rux.Context) {
	ctx.WriteString("hello, in " + ctx.URL().Path)

	key := "rux_cookie"
	val := ctx.Cookie(key)
	ctx.WriteString(key + "=" + val)

	key = "rux_cookie2"
	val = ctx.Cookie(key)
	ctx.WriteString(key + "=" + val)
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
