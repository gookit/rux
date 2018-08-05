package souter

import (
	"net/http"
	"log"
)

/*************************************************************
 * dispatch http request
 *************************************************************/

func (r *Router) RunServe(addr string)  {
	log.Fatal(http.ListenAndServe(addr, r))
}

/*************************************************************
 * dispatch http request
 *************************************************************/

func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var handlers HandlersChain
	result := r.Match(req.Method, req.URL.Path)

	// not found
	if result.Status == NotFound {
		if len(r.noRoute) > 0 {
			handlers = r.noRoute
		} else {
			http.NotFound(res, req)
		}
	} else if result.Status == NotAllowed {
		if len(r.noAllowed) > 0 {
			handlers = r.noAllowed
		} else {
			http.Error(
				res,
				"method not allowed. allow: "+result.JoinAllowedMethods(","),
				405,
			)
		}
	} else {
		handlers = result.Handlers
	}

	ctx := newContext(res, req, r.handlers)
	ctx.params = result.Params
	ctx.appendHandlers(handlers...)

	ctx.Next()
	// ctx.Res.WriteHeaderNow()
}

func (r *Router) handleRequest(ctx *Context)  {

}
