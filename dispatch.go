package souter

import (
	"net/http"
	"strings"
)

/*************************************************************
 * running with http server
 *************************************************************/

func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	status, route, allowed := r.Match(req.Method, req.URL.Path)

	// not found
	if status == NotFound {
		if len(r.noRoute) > 0 {
			// fn := r.noRoute.Last()
			// fn()
		} else {
			res.WriteHeader(404)
			res.Write([]byte("page not found."))
		}
	} else if status == NotAllowed {
		res.WriteHeader(405)
		res.Write([]byte("method not allowed. allow: " + strings.Join(allowed, ",")))
	} else {
		ctx := NewContext(res, req, r.Handlers)
		ctx.Params = route.Params

		// ctx.Next()
	}
}
