package pprof

import (
	"net/http/pprof"

	"github.com/gookit/rux"
)

// UsePProf enable for the router
func UsePProf(r *rux.Router) {
	routers := []struct {
		Method  string
		Path    string
		Handler rux.HandlerFunc
	}{
		{rux.GET, "/", rux.WrapHTTPHandlerFunc(pprof.Index)},
		{rux.GET, "/heap", rux.WrapHTTPHandler(pprof.Handler("heap"))},
		{rux.GET, "/goroutine", rux.WrapHTTPHandler(pprof.Handler("goroutine"))},
		{rux.GET, "/allocs", rux.WrapHTTPHandler(pprof.Handler("allocs"))},
		{rux.GET, "/block", rux.WrapHTTPHandler(pprof.Handler("block"))},
		{rux.GET, "/threadcreate", rux.WrapHTTPHandler(pprof.Handler("threadcreate"))},
		{rux.GET, "/cmdline", rux.WrapHTTPHandlerFunc(pprof.Cmdline)},
		{rux.GET, "/profile", rux.WrapHTTPHandlerFunc(pprof.Profile)},
		{rux.GET, "/symbol", rux.WrapHTTPHandlerFunc(pprof.Symbol)},
		{rux.POST, "/symbol", rux.WrapHTTPHandlerFunc(pprof.Symbol)},
		{rux.GET, "/trace", rux.WrapHTTPHandlerFunc(pprof.Trace)},
		{rux.GET, "/mutex", rux.WrapHTTPHandler(pprof.Handler("mutex"))},
	}

	r.Group("/debug/pprof", func() {
		for _, route := range routers {
			r.Add(route.Path, route.Handler, route.Method)
		}
	})
}
