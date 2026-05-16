package v2

import (
	"strings"
	"sync"
	"sync/atomic"
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
