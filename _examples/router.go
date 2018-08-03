package main

import (
	"github.com/gookit/souter"
	"net/http"
	"log"
)

func main() {
	r := souter.New()
	r.Use()

	r.GET("/", func(ctx *souter.Context) {
		ctx.Res.Write([]byte(`hello`))
	})

	r.Group("/users", func(sub *souter.Router) {
		sub.GET("/", func(ctx *souter.Context) {

		})

		sub.GET("/:id", func(ctx *souter.Context) {

		})
	})

	r.Controller("/site", &SiteController{})

	log.Fatal(http.ListenAndServe(":8090", r))
}

type SiteController struct {
}

func (c *SiteController) AddRoutes(r *souter.Router) {
	r.GET(":id", c.Get)
	r.POST("", c.Post)
}

func (c *SiteController) Get(ctx *souter.Context) {
	ctx.Res.Write([]byte(`hello GET`))
}

func (c *SiteController) Post(ctx *souter.Context) {
	ctx.Res.Write([]byte(`hello POST`))
}