package v2

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gookit/rux/internal/util"
)

// Router is the central registration and dispatch object.
//
// A Router moves through two lifecycle stages:
//   - Registration: routes, groups, middlewares are added.
//   - Frozen: read-only. Routes are dispatched; further registration panics.
//
// The transition is one-way and happens via Freeze (real Freeze logic lands
// in Phase 4 — this file provides a stub).
type Router struct {
	Name string

	// Per-method static routes. Indexed by methodIndex().
	// nil for methods that have no static routes registered yet.
	staticRoutes [methodCount]map[string]*Route

	// Per-method dynamic radix trees. nil if no dynamic routes for that method.
	dynamicTrees [methodCount]*radixTree

	namedRoutes map[string]*Route
	routeList   []*Route

	// Global middleware. Frozen into each route's finalChain on Freeze().
	globalChain HandlersChain

	currentGroupPrefix   string
	currentGroupHandlers HandlersChain

	noRoute   HandlersChain
	noAllowed HandlersChain

	// Settings.
	OnError                HandlerFunc
	OnPanic                HandlerFunc
	interceptAll           string
	useEncodedPath         bool
	strictLastSlash        bool
	handleMethodNotAllowed bool
	handleFallbackRoute    bool

	frozen  atomic.Bool
	counter int

	ctxPool sync.Pool

	err error
}

// New constructs a Router with optional configuration.
func New(opts ...func(*Router)) *Router {
	r := &Router{
		Name:        "default",
		namedRoutes: make(map[string]*Route),
	}
	for _, opt := range opts {
		opt(r)
	}
	r.ctxPool.New = func() any {
		return &Context{index: -1}
	}
	return r
}

// Frozen reports whether the router is in frozen (read-only) state.
func (r *Router) Frozen() bool { return r.frozen.Load() }

/*************************************************************
 * Options
 *************************************************************/

// StrictLastSlash makes /path and /path/ distinct routes.
func StrictLastSlash(r *Router) { r.strictLastSlash = true }

// UseEncodedPath uses req.URL.EscapedPath() for matching.
func UseEncodedPath(r *Router) { r.useEncodedPath = true }

// HandleMethodNotAllowed enables 405 detection across methods.
func HandleMethodNotAllowed(r *Router) { r.handleMethodNotAllowed = true }

// HandleFallbackRoute enables the "/*" wildcard route as a global fallback.
func HandleFallbackRoute(r *Router) { r.handleFallbackRoute = true }

// InterceptAll redirects all requests to the given path.
func InterceptAll(path string) func(*Router) {
	return func(r *Router) {
		r.interceptAll = strings.TrimSpace(path)
	}
}

// Freeze marks the router read-only. Subsequent registration calls panic.
// Real chain-merge logic is added in Phase 4 — this is a stub so Phase 3
// tests can exercise the frozen guard.
func (r *Router) Freeze() {
	if !r.frozen.CompareAndSwap(false, true) {
		return
	}
	// TODO(Phase 4): merge globalChain into route.finalChain + mirror GET to HEAD.
}

/*************************************************************
 * Route registration (Task 3.2)
 *************************************************************/

// Add registers a route on the given methods (defaults to GET if methods empty).
func (r *Router) Add(path string, handler HandlerFunc, methods ...string) *Route {
	route := newRoute(path, handler, methods)
	return r.AddRoute(route)
}

// AddNamed is like Add with a route name.
func (r *Router) AddNamed(name, path string, handler HandlerFunc, methods ...string) *Route {
	route := newNamedRoute(name, path, handler, methods)
	return r.AddRoute(route)
}

// AddRoute registers a pre-constructed Route.
func (r *Router) AddRoute(route *Route) *Route {
	r.appendRoute(route)
	return route
}

// Verb shortcuts.

func (r *Router) GET(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return r.Add(path, h, GET).Use(mw...)
}
func (r *Router) HEAD(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return r.Add(path, h, HEAD).Use(mw...)
}
func (r *Router) POST(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return r.Add(path, h, POST).Use(mw...)
}
func (r *Router) PUT(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return r.Add(path, h, PUT).Use(mw...)
}
func (r *Router) PATCH(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return r.Add(path, h, PATCH).Use(mw...)
}
func (r *Router) DELETE(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return r.Add(path, h, DELETE).Use(mw...)
}
func (r *Router) OPTIONS(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return r.Add(path, h, OPTIONS).Use(mw...)
}
func (r *Router) CONNECT(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return r.Add(path, h, CONNECT).Use(mw...)
}
func (r *Router) TRACE(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return r.Add(path, h, TRACE).Use(mw...)
}

// anyMethods is the canonical order used by Any.
var anyMethods = []string{GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD, CONNECT, TRACE}

// Any registers a route on every supported HTTP method.
func (r *Router) Any(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return r.Add(path, h, anyMethods...).Use(mw...)
}

// appendRoute formats the path, applies group context, expands optional
// segments, and dispatches to static or dynamic storage.
func (r *Router) appendRoute(route *Route) {
	if r.frozen.Load() {
		panic("rux: cannot add route after router is frozen")
	}

	// Apply group prefix and middlewares before dispatch.
	r.applyGroup(route)

	if route.name != "" {
		r.namedRoutes[route.name] = route
	}
	r.routeList = append(r.routeList, route)

	// Expand optional segments into one or more concrete paths.
	if hasOptionalSegment(route.path) {
		util.ValidateOptionalSegments(route.path)
		r.counter++ // count as one user-defined route regardless of expansion
		for _, expandedPath := range parseOptionalSegments(route.path) {
			expandedRoute := *route
			expandedRoute.path = normalizePath(expandedPath)
			r.registerSingleRoute(&expandedRoute)
		}
		return
	}

	r.counter++
	// Convert {id} -> :id syntax so static-vs-dynamic detection is consistent
	// with the tree's expected format.
	route.path = normalizePath(convertParamSyntax(route.path))
	r.registerSingleRoute(route)
}

// registerSingleRoute stores route in the right per-method bucket.
func (r *Router) registerSingleRoute(route *Route) {
	if isStaticPath(route.path) {
		for _, m := range route.methods {
			idx := methodIndex(m)
			if idx < 0 {
				panic("rux: unknown method " + m)
			}
			if r.staticRoutes[idx] == nil {
				r.staticRoutes[idx] = make(map[string]*Route, 4)
			}
			if _, dup := r.staticRoutes[idx][route.path]; dup {
				panic("rux: duplicate static route: " + m + " " + route.path)
			}
			r.staticRoutes[idx][route.path] = route
		}
		return
	}

	for _, m := range route.methods {
		idx := methodIndex(m)
		if idx < 0 {
			panic("rux: unknown method " + m)
		}
		if r.dynamicTrees[idx] == nil {
			r.dynamicTrees[idx] = newRadixTree()
		}
		r.dynamicTrees[idx].insert(route.path, route)
	}
}

// applyGroup merges current group prefix and handlers into the route.
func (r *Router) applyGroup(route *Route) {
	routePath := r.formatPath(route.path)
	if r.currentGroupPrefix != "" {
		routePath = r.formatPath(r.currentGroupPrefix + routePath)
	}
	route.path = routePath

	if len(r.currentGroupHandlers) > 0 {
		// Group middlewares run before route's own middlewares.
		merged := make(HandlersChain, 0, len(r.currentGroupHandlers)+len(route.chain))
		merged = append(merged, r.currentGroupHandlers...)
		merged = append(merged, route.chain...)
		route.chain = merged
	}
}

// formatPath applies the router's path policy.
func (r *Router) formatPath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}
	path = strings.TrimSpace(path)
	if !r.strictLastSlash && len(path) > 1 && path[len(path)-1] == '/' {
		path = strings.TrimRight(path, "/")
	}
	if path == "" || path == "/" {
		return "/"
	}
	if path[0] != '/' {
		return "/" + path
	}
	if len(path) > 1 && path[1] == '/' {
		return "/" + strings.TrimLeft(path, "/")
	}
	return path
}
