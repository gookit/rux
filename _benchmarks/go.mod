module ruxbench

go 1.12

require (
	github.com/gin-gonic/gin v1.9.0
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/gookit/rux v1.2.9
	github.com/gorilla/mux v1.7.4
	github.com/julienschmidt/httprouter v1.3.0
	github.com/kataras/muxie v1.1.1
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/valyala/fasthttp v1.34.0
	github.com/valyala/fasttemplate v1.2.0 // indirect
)

replace github.com/gookit/rux => ../
