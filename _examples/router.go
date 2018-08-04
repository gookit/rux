package main

import (
	"github.com/gookit/souter"
	"fmt"
)

func main() {
	r := souter.New()
	r.Use()

	r.GET("/", func(ctx *souter.Context) {
		ctx.Res.Write([]byte("hello, in " + ctx.Req.URL.Path))
	})

	r.GET("/about[.html]", def_handle)

	r.GET("/hi-{name}", def_handle)

	r.Group("/users", func(sub *souter.Router) {
		sub.GET("/", func(ctx *souter.Context) {
			ctx.Res.Write([]byte("hello, in " + ctx.Req.URL.Path))
		})

		sub.GET("/{id}", func(ctx *souter.Context) {
			ctx.Res.Write([]byte("hello, in " + ctx.Req.URL.Path))
		})
	})

	r.Controller("/site", &SiteController{})

	fmt.Println(r)

	st, route, _ := r.Match("GET", "/hi-tom")
	st, route1, _ := r.Match("GET", "/hi-john")

	fmt.Println(st, route, route.Params)
	fmt.Println(st, route1, route1.Params)

	// log.Fatal(http.ListenAndServe(":8090", r))
}

func def_handle(ctx *souter.Context) {
	ctx.Res.Write([]byte("hello, in " + ctx.Req.URL.Path))
}

type SiteController struct {
}

func (c *SiteController) AddRoutes(r *souter.Router) {
	r.GET("{id}", c.Get)
	r.POST("", c.Post)
}

func (c *SiteController) Get(ctx *souter.Context) {
	ctx.Res.Write([]byte("hello, in " + ctx.Req.URL.Path))
}

func (c *SiteController) Post(ctx *souter.Context) {
	ctx.Res.Write([]byte("hello, in " + ctx.Req.URL.Path))
}