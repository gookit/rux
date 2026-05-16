package rux

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
)

/*************************************************************
 * internal vars
 *************************************************************/

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
	ctx := r.ctxPool.Get().(*Context)
	ctx.Init(res, req)

	r.handleHTTPRequest(ctx)

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

	method := ctx.Req.Method

	// Fast path: direct match without allocating MatchResult
	if r.interceptAll != "" {
		path = r.interceptAll
	} else {
		path = r.formatPath(path)
	}

	// Do match route directly
	route, params := r.match(method, path)

	var handlers HandlersChain
	if route != nil { // found route
		ctx.Params = params
		if route.name != "" {
			ctx.Set(CTXCurrentRouteName, route.name)
		}
		ctx.Set(CTXCurrentRoutePath, path)

		handlers = buildHandlers(route)
	} else {
		// for HEAD requests, attempt fallback to GET
		if method == HEAD {
			route, params = r.match(GET, path)
			if route != nil {
				ctx.Params = params
				if route.name != "" {
					ctx.Set(CTXCurrentRouteName, route.name)
				}
				ctx.Set(CTXCurrentRoutePath, path)
				handlers = buildHandlers(route)
				goto executeHandlers
			}
		}

		// handle fallback route
		if r.handleFallbackRoute {
			key := method + "/*"
			if fallbackRoute, ok := r.stableRoutes[key]; ok {
				route = fallbackRoute
				handlers = buildHandlers(route)
				goto executeHandlers
			}
		}

		// handle method not allowed
		if r.handleMethodNotAllowed {
			allowed := r.findAllowedMethods(method, path)
			if len(allowed) > 0 {
				if len(r.noAllowed) == 0 {
					r.noAllowed = HandlersChain{internal405Handler}
				}
				ctx.Set(CTXAllowedMethods, allowed)
				handlers = r.noAllowed
				goto executeHandlers
			}
		}

		// not found route
		if len(r.noRoute) == 0 {
			r.noRoute = HandlersChain{internal404Handler}
		}
		handlers = r.noRoute
	}

executeHandlers:
	// has global middleware handlers
	if len(r.handlers) > 0 {
		handlers = append(r.handlers, handlers...)
	}

	ctx.SetHandlers(handlers)
	ctx.Next()

	// has errors and has error handler
	if r.OnError != nil && len(ctx.Errors) > 0 {
		r.OnError(ctx)
	}

	ctx.writer.ensureWriteHeader()
}

/*************************************************************
 * route matching
 *************************************************************/

// Match route by given request METHOD and URI path
// Note: The returned MatchResult should not be modified or held for long periods.
// For performance-critical paths, consider using the internal match() method directly.
func (r *Router) Match(method, path string) *MatchResult {
	return r.QuickMatch(strings.ToUpper(method), path)
}

// QuickMatch match route by given request METHOD and URI path
// Note: The returned MatchResult should not be modified or held for long periods.
// For performance-critical paths, consider using the internal match() method directly.
func (r *Router) QuickMatch(method, path string) *MatchResult {
	result := r.matchResultPool.Get().(*MatchResult)
	result.Method = method
	result.Path = path
	result.Route = nil
	result.Params = nil
	result.Allowed = nil

	if r.interceptAll != "" {
		path = r.interceptAll
	} else {
		path = r.formatPath(path)
	}

	result.Path = path

	// do match route
	route, params := r.match(method, path)
	if route != nil {
		result.Route = route
		result.Params = params
		return result
	}

	// for HEAD requests, attempt fallback to GET
	if method == HEAD {
		route, params = r.match(GET, path)
		if route != nil {
			result.Route = route
			result.Params = params
			return result
		}
	}

	// handle fallback route. add by: router->Any("/*", handler)
	if r.handleFallbackRoute {
		key := method + "/*"
		if route, ok := r.stableRoutes[key]; ok {
			result.Route = route
			return result
		}
	}

	// handle method not allowed. will find allowed methods
	if r.handleMethodNotAllowed {
		alm := r.findAllowedMethods(method, path)
		if len(alm) > 0 {
			result.Allowed = alm
		}
	}

	return result
}

// ReleaseMatchResult returns a MatchResult to the pool for reuse.
// Only call this if you obtained the MatchResult from Match/QuickMatch and
// no longer need it. Do not call this on MatchResults you created yourself.
func (r *Router) ReleaseMatchResult(result *MatchResult) {
	if result != nil {
		result.Route = nil
		result.Params = nil
		result.Allowed = nil
		r.matchResultPool.Put(result)
	}
}

func (r *Router) match(method, path string) (rt *Route, ps Params) {
	// find in stable routes
	if route, ok := r.stableRoutes[method+path]; ok {
		return route, nil
	}

	// find in Radix Tree (dynamic route matching)
	if r.dynamicTrees != nil {
		if tree, ok := r.dynamicTrees.getTree(method); ok {
			if _, params, foundRoute, found := tree.FindRouteWithRoute(method, path, r.strictLastSlash); found {
				if foundRoute != nil {
					return foundRoute, params
				}
			}
		}
	}
	return
}

// buildHandlers builds the final handlers chain for a route.
// Order: [middleware..., mainHandler]
func buildHandlers(route *Route) HandlersChain {
	if route.handler != nil {
		return append(route.handlers, route.handler)
	}
	return route.handlers
}

// find allowed methods for current request
func (r *Router) findAllowedMethods(method, path string) (allowed []string) {
	mMap := map[string]int{}
	for _, m := range anyMethods {
		if m == method {
			continue
		}

		if rt, _ := r.match(m, path); rt != nil {
			mMap[m] = 1
		}
	}

	if len(mMap) > 0 {
		for m := range mMap {
			allowed = append(allowed, m)
		}
	}
	return
}
