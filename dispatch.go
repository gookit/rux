package sux

import (
	"log"
	"net/http"
	"strings"
)

/*************************************************************
 * dispatch http request
 *************************************************************/

// Listen create a http server
func (r *Router) Listen(addr string) {
	ss := strings.SplitN(addr, ":", 2)
	ip, port := ss[0], ss[1]
	if ip == "" {
		ip = "0.0.0.0"
		addr = ip + ":" + port
	}

	log.Printf("About to listen on %s. Go to http://%s", port, addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

// ListenTLS create a https server
func (r *Router) ListenTLS(addr string) {
	ss := strings.SplitN(addr, ":", 2)
	ip, port := ss[0], ss[1]
	if ip == "" {
		ip = "0.0.0.0"
		addr = ip + ":" + port
	}

	log.Printf("About to listen on %s. Go to https://%s", port, addr)
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
	switch result.Status {
	case NotFound:
		if len(r.noRoute) == 0 {
			http.NotFound(res, req)
			return
		}

		handlers = r.noRoute
	case NotAllowed:
		if len(r.noAllowed) == 0 {
			http.Error(res, "method not allowed. allow: "+result.JoinAllowedMethods(","), 405)
			return
		}

		handlers = r.noAllowed
	default:
		handlers = result.Handlers

		// has global middleware handlers
		if len(r.handlers) > 0 {
			handlers = append(handlers, r.handlers...)
		}

		// add main handler
		handlers = append(handlers, result.Handler)
	}

	// get context
	ctx := r.pool.Get().(*Context)
	ctx.Reset()
	ctx.SetParams(result.Params)
	ctx.InitRequest(res, req, handlers)

	// add allowed methods to context
	if result.Status == NotAllowed {
		ctx.Set("allowedMethods", result.AllowedMethods)
	}

	ctx.Next()
	// ctx.Res.WriteHeaderNow()

	// clean
	result = nil
	r.pool.Put(ctx)
}
