package souter

import (
	"net/http"
	"strings"
)

/*************************************************************
 * dispatch http request
 *************************************************************/

func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var handlers HandlersChain
	status, route, allowed := r.Match(req.Method, req.URL.Path)

	// not found
	if status == NotFound {
		if len(r.noRoute) > 0 {
			handlers = r.noRoute
		} else {
			http.NotFound(res, req)
		}
	} else if status == NotAllowed {
		if len(r.noAllowed) > 0 {
			handlers = r.noAllowed
		} else {
			http.Error(
				res,
				"method not allowed. allow: "+strings.Join(allowed, ","),
				405,
			)
		}
	} else {
		handlers = route.handlers
	}

	ctx := newContext(res, req, r.Handlers)
	ctx.Params = route.Params
	ctx.appendHandlers(handlers...)

	ctx.Next()
	// ctx.Res.WriteHeaderNow()
}

func (r *Router) handleRequest(ctx *Context)  {

}
