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

// WrapHttpHandlers apply some pre http handlers for the router.
// usage:
// 	import "github.com/gookit/sux/handlers"
// 	r := sux.New()
//  // ... add routes
// 	handler := r.WrapHttpHandlers(handlers.HTTPMethodOverrideHandler)
// 	http.ListenAndServe(":8080", handler)
func (r *Router) WrapHttpHandlers(preHandlers ...func(h http.Handler) http.Handler) http.Handler {
	var wrapped http.Handler
	max := len(preHandlers)
	lst := make([]int, max)

	for i := range lst {
		current := max - i - 1
		if i == 0 {
			wrapped = preHandlers[current](r)
		} else {
			wrapped = preHandlers[current](wrapped)
		}
	}

	return wrapped
}

/*************************************************************
 * dispatch http request
 *************************************************************/

// CTXAllowedMethods key name in the context
const CTXAllowedMethods = "_allowedMethods"

var internal404Handler HandlerFunc = func(c *Context) {
	http.NotFound(c.Resp, c.Req)
}

var internal405Handler HandlerFunc = func(c *Context) {
	allowed := c.MustGet(CTXAllowedMethods).([]string)
	sort.Strings(allowed)
	c.SetHeader("Allow", strings.Join(allowed, ", "))

	if c.Req.Method == "OPTIONS" {
		c.SetStatus(200)
	} else {
		http.Error(c.Resp, "Method not allowed", 405)
	}
}

// ServeHTTP for handle HTTP request, response data to client.
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var handlers HandlersChain

	path := req.URL.Path
	if r.useEncodedPath {
		path = req.URL.EscapedPath()
	}

	if len(r.noRoute) == 0 {
		r.noRoute = HandlersChain{internal404Handler}
	}

	// match route
	result := r.Match(req.Method, path)

	// get new context
	ctx := r.pool.Get().(*Context)
	ctx.Init(res, req)
	ctx.Params = result.Params

	// check match result
	switch result.Status {
	case Found:
		// append main handler to last
		handlers = append(result.Handlers, result.Handler)
	case NotFound:
		handlers = r.noRoute
	case NotAllowed:
		if len(r.noAllowed) == 0 {
			r.noAllowed = HandlersChain{internal405Handler}
		}

		// add allowed methods to context
		ctx.Set(CTXAllowedMethods, result.AllowedMethods)
		handlers = r.noAllowed
	}

	// has global middleware handlers
	if len(r.handlers) > 0 {
		handlers = append(r.handlers, handlers...)
	}

	// add handlers
	ctx.SetHandlers(handlers)

	ctx.Next()  // handle processing
	ctx.Reset() // reset data
	// ctx.Resp.WriteHeaderNow()

	// release data
	r.pool.Put(ctx)
	result = nil
}
