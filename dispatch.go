package sux

import (
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

/*************************************************************
 * create http server
 *************************************************************/

// Listen quick create a http server with router
func (r *Router) Listen(addr ...string) {
	ip, port := resolveAddress(addr)
	address := ip + ":" + port

	log.Printf("About to listen on %s. Go to http://%s", port, address)
	log.Fatal(http.ListenAndServe(address, r))
}

// ListenTLS create a https server
func (r *Router) ListenTLS(addr ...string) {
	ip, port := resolveAddress(addr)
	address := ip + ":" + port

	log.Printf("About to listen on %s. Go to https://%s", port, address)
	log.Fatal(http.ListenAndServe(address, r))
}

func resolveAddress(addr []string) (ip, port string) {
	ip = "0.0.0.0"
	switch len(addr) {
	case 0:
		if port := os.Getenv("PORT"); len(port) > 0 {
			debugPrint("Environment variable PORT=\"%s\"", port)
			return ip, port
		}
		debugPrint("Environment variable PORT is undefined. Using port :8080 by default")
		return ip, "8080"
	case 1:
		if strings.Index(addr[0], ":") != -1 {
			ss := strings.SplitN(addr[0], ":", 2)
			if ss[0] != "" {
				ip = ss[0]
			}
			port = ss[1]
		} else {
			port = addr[0]
		}

		return
	default:
		panic("too much parameters")
	}
}

// WrapHttpHandlers apply some pre http handlers for the router
// usage:
// 	import "github.com/gookit/sux/handlers"
//  // ... create router and add routes
//	handler := r.WrapHttpHandlers(handlers.HTTPMethodOverrideHandler)
// 	http.ListenAndServe(":8080", handler)
func (r *Router) WrapHttpHandlers(preHandlers ...func(h http.Handler) http.Handler) http.Handler {
	var wrapped http.Handler
	for i, handler := range preHandlers {
		if i == 0 {
			wrapped = handler(r)
		} else {
			wrapped = handler(wrapped)
		}
	}

	return wrapped
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

	// match route
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
			allowed := result.AllowedMethods
			sort.Strings(allowed)

			res.Header().Set("Allow", strings.Join(allowed, ", "))
			if req.Method == "OPTIONS" {
				res.WriteHeader(200)
			} else {
				http.Error(res, "Method not allowed", 405)
			}
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

	// now, call all handlers

	// get context
	ctx := r.pool.Get().(*Context)
	ctx.Reset()
	ctx.Params = result.Params
	ctx.InitRequest(res, req, handlers)

	// add allowed methods to context
	if result.Status == NotAllowed {
		ctx.Set("allowedMethods", result.AllowedMethods)
	}

	ctx.Next()
	// ctx.Resp.WriteHeaderNow()

	// release
	result = nil
	r.pool.Put(ctx)
}
