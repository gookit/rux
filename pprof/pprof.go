package pprof

import (
	"github.com/gookit/rux"
	"net/http/pprof"
)

// UsePProf enable for the router
func UsePProf(r *rux.Router) {
	r.GET("/debug/pprof/", func(c *rux.Context) {
		c.Resp.Header().Set(rux.ContentType, "text/html; charset=utf-8")

		pprof.Index(c.Resp, c.Req)
	})
	r.GET("/debug/pprof/heap", rux.WrapHTTPHandler(pprof.Handler("heap")))
	r.GET("/debug/pprof/goroutine", rux.WrapHTTPHandler(pprof.Handler("goroutine")))
	r.GET("/debug/pprof/block", rux.WrapHTTPHandler(pprof.Handler("block")))
	r.GET("/debug/pprof/threadcreate", rux.WrapHTTPHandler(pprof.Handler("threadcreate")))
	r.GET("/debug/pprof/cmdline", rux.WrapHTTPHandlerFunc(pprof.Cmdline))
	r.GET("/debug/pprof/profile", rux.WrapHTTPHandlerFunc(pprof.Profile))
	r.GET("/debug/pprof/symbol", rux.WrapHTTPHandlerFunc(pprof.Symbol))
	r.GET("/debug/pprof/mutex", rux.WrapHTTPHandler(pprof.Handler("mutex")))
}
