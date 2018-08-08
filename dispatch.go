package sux

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
)

/*************************************************************
 * starting HTTP serve
 *************************************************************/

// Listen quick create a HTTP server with the router
func (r *Router) Listen(addr ...string) (err error) {
	defer func() { debugPrintError(err) }()
	address := resolveAddress(addr)

	fmt.Printf("Serve listen on %s. Go to http://%s\n", address, address)
	err = http.ListenAndServe(address, r)
	return
}

// ListenTLS attaches the router to a http.Server and starts listening and serving HTTPS (secure) requests.
func (r *Router) ListenTLS(addr, certFile, keyFile string) (err error) {
	defer func() { debugPrintError(err) }()
	address := resolveAddress([]string{addr})

	fmt.Printf("Serve listen on %s. Go to https://%s\n", address, address)
	err = http.ListenAndServeTLS(address, certFile, keyFile, r)
	return
}

// ListenUnix attaches the router to a http.Server and starts listening and serving HTTP requests
// through the specified unix socket (ie. a file)
func (r *Router) ListenUnix(file string) (err error) {
	defer func() { debugPrintError(err) }()
	fmt.Printf("Serve listen on unix:/%s\n", file)

	os.Remove(file)
	listener, err := net.Listen("unix", file)
	if err != nil {
		return
	}
	defer listener.Close()

	err = http.Serve(listener, r)
	return
}

func resolveAddress(addr []string) (fullAddr string) {
	ip := "0.0.0.0"
	switch len(addr) {
	case 0:
		if port := os.Getenv("PORT"); len(port) > 0 {
			debugPrint("Environment variable PORT=\"%s\"", port)
			return ip + ":" + port
		}
		debugPrint("Environment variable PORT is undefined. Using port :8080 by default")
		return ip + ":8080"
	case 1:
		var port string
		if strings.Index(addr[0], ":") != -1 {
			ss := strings.SplitN(addr[0], ":", 2)
			if ss[0] != "" {
				return addr[0]
			}
			port = ss[1]
		} else {
			port = addr[0]
		}

		return ip + ":" + port
	default:
		panic("too much parameters")
	}
}

// WrapHttpHandlers apply some pre http handlers for the router.
// usage:
// 	import "github.com/gookit/sux/handlers"
//	r := sux.New()
//  // ... add routes
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

var default404Handler = func(c *Context) {
	http.NotFound(c.Resp, c.Req)
}

// ServeHTTP for handle HTTP request, response data to client.
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if r.UseEncodedPath {
		path = req.URL.EscapedPath()
	}

	// match route
	result := r.Match(req.Method, path)
	handlers := result.Handlers
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
		// has global middleware handlers
		if len(r.handlers) > 0 {
			handlers = append(r.handlers, handlers...)
		}

		// append main handler
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

	// processing
	ctx.Next()
	// ctx.Resp.WriteHeaderNow()

	// release
	result = nil
	r.pool.Put(ctx)
}
