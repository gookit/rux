package sux

import (
	"log"
	"net/http"
)

/*************************************************************
 * dispatch http request
 *************************************************************/

func (r *Router) RunServe(addr string) {
	log.Fatal(http.ListenAndServe(addr, r))
}

/*************************************************************
 * dispatch http request
 *************************************************************/

func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var handlers HandlersChain

	path := req.URL.Path
	if r.UseEncodedPath {
		path = req.URL.EscapedPath()
	}

	result := r.Match(req.Method, path)

	if result.Status == NotFound {
		if len(r.noRoute) == 0 {
			http.NotFound(res, req)
			return
		}

		handlers = r.noRoute
	} else if result.Status == NotAllowed {
		if len(r.noAllowed) == 0 {
			http.Error(
				res,
				"method not allowed. allow: "+result.JoinAllowedMethods(","),
				405,
			)
			return
		}

		handlers = r.noAllowed
	} else {
		handlers = result.Handlers
	}

	ctx := r.pool.Get().(*Context)
	ctx.Reset()

	ctx.Init(res, req, r.handlers)
	ctx.SetParams(result.Params)
	ctx.AppendHandlers(handlers...)

	ctx.Next()
	// ctx.Res.WriteHeaderNow()

	// clean
	result = nil
	r.pool.Put(ctx)
}
