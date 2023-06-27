package rux

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
)

/*************************************************************
 * internal vars
 *************************************************************/

// "/users/{id}" "/users/{id:\d+}" `/users/{uid:\d+}/blog/{id}`
var varRegex = regexp.MustCompile(`{[^/]+}`)

var internal404Handler HandlerFunc = func(c *Context) {
	http.NotFound(c.Resp, c.Req)
}

var internal405Handler HandlerFunc = func(c *Context) {
	allowed := c.SafeGet(CTXAllowedMethods).([]string)
	sort.Strings(allowed)
	c.SetHeader("Allow", strings.Join(allowed, ", "))

	if c.Req.Method == OPTIONS {
		c.SetStatus(200)
	} else {
		http.Error(c.Resp, "Method not allowed", 405)
	}
}

/*************************************************************
 * starting HTTP serve
 *************************************************************/

// Listen quick create a HTTP server with the router
//
// Usage:
//
//	r.Listen("8090")
//	r.Listen("IP:PORT")
//	r.Listen("IP", "PORT")
func (r *Router) Listen(addr ...string) {
	defer func() {
		debugPrintError(r.err)
	}()

	address := resolveAddress(addr)

	fmt.Printf("Serve listen on %s. Go to http://%s\n", address, address)
	r.err = http.ListenAndServe(address, r)
}

// ListenTLS attaches the router to a http.Server and starts listening and serving HTTPS (secure) requests.
func (r *Router) ListenTLS(addr, certFile, keyFile string) {
	var err error
	defer func() { debugPrintError(err) }()
	address := resolveAddress([]string{addr})

	fmt.Printf("Serve listen on %s. Go to https://%s\n", address, address)
	err = http.ListenAndServeTLS(address, certFile, keyFile, r)
}

// ListenUnix attaches the router to a http.Server and starts listening and serving HTTP requests
// through the specified unix socket (i.e. a file)
func (r *Router) ListenUnix(file string) {
	var err error
	defer func() { debugPrintError(err) }()
	fmt.Printf("Serve listen on unix:/%s\n", file)

	if err = os.Remove(file); err != nil {
		return
	}

	listener, err := net.Listen("unix", file)
	if err != nil {
		return
	}
	// defer listener.Close()

	err = http.Serve(listener, r)
	_ = listener.Close()
}

// WrapHTTPHandlers apply some pre http handlers for the router.
//
// Usage:
//
//		import "github.com/gookit/rux/handlers"
//		r := rux.New()
//	 // ... add routes
//		handler := r.WrapHTTPHandlers(handlers.HTTPMethodOverrideHandler)
//		http.ListenAndServe(":8080", handler)
func (r *Router) WrapHTTPHandlers(preHandlers ...func(h http.Handler) http.Handler) http.Handler {
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

// ServeHTTP for handle HTTP request, response data to client.
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// get new context
	ctx := r.ctxPool.Get().(*Context)
	// init and reset ctx
	ctx.Init(res, req)

	// handle HTTP Request
	r.handleHTTPRequest(ctx)

	// ctx.Reset()
	// release ctx
	r.ctxPool.Put(ctx)
}

// HandleContext handle a given context
func (r *Router) HandleContext(c *Context) {
	c.Reset()
	r.handleHTTPRequest(c)
	r.ctxPool.Put(c)
}

// handle HTTP Request
func (r *Router) handleHTTPRequest(ctx *Context) {
	// has panic handler
	if r.OnPanic != nil {
		defer func() {
			if ret := recover(); ret != nil {
				ctx.Set(CTXRecoverResult, ret)
				r.OnPanic(ctx)
			}
		}()
	}

	path := ctx.Req.URL.Path
	if r.useEncodedPath {
		path = ctx.Req.URL.EscapedPath()
	}

	// matching route
	route, params, allowed := r.QuickMatch(ctx.Req.Method, path)

	var handlers HandlersChain
	if route != nil { // found route
		// save route params
		ctx.Params = params
		ctx.Set(CTXCurrentRouteName, route.name)
		ctx.Set(CTXCurrentRoutePath, path)

		// append main handler to last
		handlers = append(route.handlers, route.handler)
	} else if len(allowed) > 0 { // method not allowed
		if len(r.noAllowed) == 0 {
			r.noAllowed = HandlersChain{internal405Handler}
		}

		// add allowed methods to context
		ctx.Set(CTXAllowedMethods, allowed)
		handlers = r.noAllowed
	} else { // not found route
		if len(r.noRoute) == 0 {
			r.noRoute = HandlersChain{internal404Handler}
		}

		handlers = r.noRoute
	}

	// has global middleware handlers
	if len(r.handlers) > 0 {
		handlers = append(r.handlers, handlers...)
	}

	ctx.SetHandlers(handlers)
	ctx.Next() // handle processing

	// has errors and has error handler
	if r.OnError != nil && len(ctx.Errors) > 0 {
		r.OnError(ctx)
	}

	ctx.writer.ensureWriteHeader()
}
