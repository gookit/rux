package pprof

import (
	"net/http/pprof"

	"github.com/gookit/rux"
)

// UsePProf enable PProf for the rux serve
func UsePProf(r *rux.Router) {
	routes := []struct {
		Method  string
		Path    string
		Handler rux.HandlerFunc
	}{
		{rux.GET, "/pprof", rux.HTTPHandlerFunc(pprof.Index)},
		{rux.GET, "/heap", rux.HTTPHandler(pprof.Handler("heap"))},
		{rux.GET, "/goroutine", rux.HTTPHandler(pprof.Handler("goroutine"))},
		{rux.GET, "/allocs", rux.HTTPHandler(pprof.Handler("allocs"))},
		{rux.GET, "/block", rux.HTTPHandler(pprof.Handler("block"))},
		{rux.GET, "/threadcreate", rux.HTTPHandler(pprof.Handler("threadcreate"))},
		{rux.GET, "/cmdline", rux.HTTPHandlerFunc(pprof.Cmdline)},
		{rux.GET, "/profile", rux.HTTPHandlerFunc(pprof.Profile)},
		{rux.GET, "/symbol", rux.HTTPHandlerFunc(pprof.Symbol)},
		{rux.POST, "/symbol", rux.HTTPHandlerFunc(pprof.Symbol)},
		{rux.GET, "/trace", rux.HTTPHandlerFunc(pprof.Trace)},
		{rux.GET, "/mutex", rux.HTTPHandler(pprof.Handler("mutex"))},
	}

	r.Group("/debug", func() {
		for _, route := range routes {
			r.Add(route.Path, route.Handler, route.Method)
		}
	})
}
