package main

import (
	"github.com/gookit/souter"
	"fmt"
)

func main() {
	r := souter.New()
	r.Use()

	r.GET("/", func(ctx *souter.Context) {
		ctx.WriteBytes([]byte("hello, in " + ctx.URL().Path))
	})

	r.GET("/about[.html]", defHandle)

	r.GET("/hi-{name}", defHandle)

	r.Group("/users", func(sub *souter.Router) {
		sub.GET("/", func(ctx *souter.Context) {
			ctx.WriteBytes([]byte("hello, in " + ctx.URL().Path))
		})

		sub.GET("/{id}", func(ctx *souter.Context) {
			ctx.WriteBytes([]byte("hello, in " + ctx.URL().Path))
		})
	})

	r.Controller("/site", &SiteController{})

	fmt.Println(r)

	ret := r.Match("GET", "/hi-tom")
	ret1 := r.Match("GET", "/hi-john")

	fmt.Println(ret)
	fmt.Println(ret1)

	// r.RunServe(":8090")
}

func defHandle(ctx *souter.Context) {
	ctx.WriteBytes([]byte("hello, in " + ctx.URL().Path))
}

type SiteController struct {
}

func (c *SiteController) AddRoutes(r *souter.Router) {
	r.GET("{id}", c.Get)
	r.POST("", c.Post)
}

func (c *SiteController) Get(ctx *souter.Context) {
	ctx.WriteBytes([]byte("hello, in " + ctx.URL().Path))
	ctx.WriteBytes([]byte("\n ok"))
}

func (c *SiteController) Post(ctx *souter.Context) {
	ctx.WriteBytes([]byte("hello, in " + ctx.URL().Path))
}