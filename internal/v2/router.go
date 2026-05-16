package v2

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
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
		return &Context{index: -1, router: r}
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
// It merges globalChain into each route's finalChain and mirrors GET routes
// onto HEAD. Idempotent — safe to call multiple times.
func (r *Router) Freeze() {
	if !r.frozen.CompareAndSwap(false, true) {
		return
	}
	for _, route := range r.routeList {
		if len(r.globalChain) == 0 {
			route.finalChain = route.chain
			continue
		}
		merged := make(HandlersChain, 0, len(r.globalChain)+len(route.chain))
		merged = append(merged, r.globalChain...)
		merged = append(merged, route.chain...)
		route.finalChain = merged
	}
	r.mirrorGetToHead()
}

// mirrorGetToHead copies every GET route to HEAD unless an explicit HEAD
// route already exists at that path. Required by P-9.
func (r *Router) mirrorGetToHead() {
	getIdx := methodIndex(GET)
	headIdx := methodIndex(HEAD)

	// Static.
	if getStatic := r.staticRoutes[getIdx]; getStatic != nil {
		if r.staticRoutes[headIdx] == nil {
			r.staticRoutes[headIdx] = make(map[string]*Route, len(getStatic))
		}
		head := r.staticRoutes[headIdx]
		for path, route := range getStatic {
			if _, exists := head[path]; !exists {
				head[path] = route
			}
		}
	}

	// Dynamic.
	if get := r.dynamicTrees[getIdx]; get != nil {
		if r.dynamicTrees[headIdx] == nil {
			r.dynamicTrees[headIdx] = newRadixTree()
		}
		head := r.dynamicTrees[headIdx]
		get.walk(func(path string, leaf *node) {
			if !head.hasExact(path) {
				head.insert(path, leaf.route)
			}
		})
	}
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

	// Preserve the registered (group-prefixed) path with {name} placeholders
	// so Route.ToURL can substitute them at URL-build time.
	route.originalPath = route.path

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

/*************************************************************
 * Task 3.3: Group / Use / NotFound / NotAllowed
 *************************************************************/

// Group registers all routes added inside fn under the given path prefix.
// Middlewares supplied as middles run before route-level middlewares.
// Nested Group calls compose: prefixes concatenate, handlers stack.
func (r *Router) Group(prefix string, fn func(), middles ...HandlerFunc) {
	prevPrefix := r.currentGroupPrefix
	r.currentGroupPrefix = prevPrefix + r.formatPath(prefix)

	prevHandlers := r.currentGroupHandlers
	if len(middles) > 0 {
		if len(prevHandlers) > 0 {
			// Create a fresh slice to avoid aliasing prevHandlers's backing array.
			r.currentGroupHandlers = append(append(HandlersChain{}, prevHandlers...), middles...)
		} else {
			r.currentGroupHandlers = middles
		}
	}

	fn()

	r.currentGroupPrefix = prevPrefix
	r.currentGroupHandlers = prevHandlers
}

// Use appends global middleware. Per Q6 of the design, Use must be called
// before any route registration — calling it later panics.
func (r *Router) Use(handlers ...HandlerFunc) {
	if r.frozen.Load() {
		panic("rux: cannot Use after router is frozen")
	}
	if len(r.routeList) > 0 {
		panic("rux: Use must be called before any route registration (Q6)")
	}
	r.globalChain = append(r.globalChain, handlers...)
}

// NotFound sets the handlers chain for unmatched routes (404).
func (r *Router) NotFound(handlers ...HandlerFunc) {
	r.noRoute = handlers
}

// NotAllowed sets the handlers chain for HTTP 405 responses.
// Only consulted when HandleMethodNotAllowed is enabled.
func (r *Router) NotAllowed(handlers ...HandlerFunc) {
	r.noAllowed = handlers
}

// Handlers returns the global middleware chain.
func (r *Router) Handlers() HandlersChain { return r.globalChain }

/*************************************************************
 * Task 3.5: Controller / Resource
 *************************************************************/

// ControllerFace is implemented by structs registered via Router.Controller.
// Each implementation places its route registrations inside AddRoutes; the
// Router applies the basePath as a group prefix.
type ControllerFace interface {
	AddRoutes(g *Router)
}

// RESTful action names, mapped to default HTTP methods.
const (
	IndexAction  = "Index"
	CreateAction = "Create"
	StoreAction  = "Store"
	ShowAction   = "Show"
	EditAction   = "Edit"
	UpdateAction = "Update"
	DeleteAction = "Delete"
)

// RESTFulActions maps action method names to their default HTTP verbs.
var RESTFulActions = map[string][]string{
	IndexAction:  {GET},
	CreateAction: {GET},
	StoreAction:  {POST},
	ShowAction:   {GET},
	EditAction:   {GET},
	UpdateAction: {PUT, PATCH},
	DeleteAction: {DELETE},
}

// Controller registers all routes from the controller's AddRoutes method
// under the given basePath, with shared middlewares.
func (r *Router) Controller(basePath string, c ControllerFace, middles ...HandlerFunc) {
	r.Group(basePath, func() {
		c.AddRoutes(r)
	}, middles...)
}

// Resource registers RESTful routes for the given controller struct.
//
//	Methods     Path                Action    Route Name
//	GET         /resource            Index    resource_index
//	GET         /resource/create     Create   resource_create
//	POST        /resource            Store    resource_store
//	GET         /resource/{id}       Show     resource_show
//	GET         /resource/{id}/edit  Edit     resource_edit
//	PUT/PATCH   /resource/{id}       Update   resource_update
//	DELETE      /resource/{id}       Delete   resource_delete
//
// If the controller has a Uses() map[string][]HandlerFunc method, per-action
// middlewares are wired automatically.
func (r *Router) Resource(basePath string, controller any, middles ...HandlerFunc) {
	cv := reflect.ValueOf(controller)
	if cv.Kind() != reflect.Ptr {
		panic("rux: Resource controller must be a pointer")
	}
	ct := cv.Type()
	if cv.Elem().Type().Kind() != reflect.Struct {
		panic("rux: Resource controller must be a pointer to struct")
	}

	var perActionMW = map[string][]HandlerFunc{}
	if m := cv.MethodByName("Uses"); m.IsValid() {
		if uses, ok := m.Interface().(func() map[string][]HandlerFunc); ok {
			perActionMW = uses()
		}
	}

	resName := strings.ToLower(ct.Elem().Name())
	basePath = strings.TrimRight(basePath, "/") + "/" + resName

	r.Group(basePath, func() {
		for action, methods := range RESTFulActions {
			m := cv.MethodByName(action)
			if !m.IsValid() {
				continue
			}
			handler, ok := m.Interface().(func(*Context))
			if !ok {
				continue
			}
			routeName := resName + "_" + strings.ToLower(action)

			var route *Route
			switch action {
			case IndexAction, StoreAction:
				route = r.AddNamed(routeName, "/", handler, methods...)
			case CreateAction:
				route = r.AddNamed(routeName, "/"+strings.ToLower(action)+"/", handler, methods...)
			case EditAction:
				route = r.AddNamed(routeName, "{id}/"+strings.ToLower(action)+"/", handler, methods...)
			default: // Show, Update, Delete
				route = r.AddNamed(routeName, "{id}/", handler, methods...)
			}

			if mws, ok := perActionMW[action]; ok {
				route.Use(mws...)
			}
		}
	}, middles...)
}

/*************************************************************
 * Static file helpers (Task 3.6)
 *************************************************************/

// StaticFile registers a single static file under the given path.
func (r *Router) StaticFile(path, filePath string) *Route {
	return r.GET(path, func(c *Context) { c.File(filePath) })
}

// StaticDir serves files from rootDir under prefixURL using http.FileServer.
func (r *Router) StaticDir(prefixURL, rootDir string) *Route {
	fs := http.StripPrefix(prefixURL, http.FileServer(http.Dir(rootDir)))
	return r.GET(prefixURL+"/*file", func(c *Context) {
		fs.ServeHTTP(c.Resp, c.Req)
	})
}

// StaticFS serves files from the given http.FileSystem under prefixURL.
func (r *Router) StaticFS(prefixURL string, fs http.FileSystem) *Route {
	handler := http.StripPrefix(prefixURL, http.FileServer(fs))
	return r.GET(prefixURL+"/*file", func(c *Context) {
		handler.ServeHTTP(c.Resp, c.Req)
	})
}

// StaticFiles serves files from rootDir under prefixURL. The exts argument
// is reserved for future extension filtering and is currently ignored.
func (r *Router) StaticFiles(prefixURL, rootDir, exts string) *Route {
	fs := http.FileServer(http.Dir(rootDir))
	_ = exts // reserved for future extension filtering
	return r.GET(fmt.Sprintf("%s/*file", prefixURL), func(c *Context) {
		c.Req.URL.Path = c.Param("file")
		fs.ServeHTTP(c.Resp, c.Req)
	})
}

/*************************************************************
 * Inspection (Task 3.7 subset — used here by Resource tests)
 *************************************************************/

// GetRoute returns a named route or nil.
func (r *Router) GetRoute(name string) *Route { return r.namedRoutes[name] }

// NamedRoutes returns the map of named routes.
func (r *Router) NamedRoutes() map[string]*Route { return r.namedRoutes }

// Routes returns all routes as RouteInfo snapshots in registration order.
func (r *Router) Routes() []RouteInfo {
	out := make([]RouteInfo, 0, len(r.routeList))
	for _, route := range r.routeList {
		out = append(out, route.Info())
	}
	return out
}

// IterateRoutes calls fn for each registered route in registration order.
func (r *Router) IterateRoutes(fn func(*Route)) {
	for _, route := range r.routeList {
		fn(route)
	}
}

// Err returns the most recent error (e.g., from a future Listen* helper).
func (r *Router) Err() error { return r.err }

// String returns a human-readable snapshot of registered routes.
func (r *Router) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Routes Count: %d\n", r.counter)
	for _, route := range r.routeList {
		fmt.Fprintf(&b, "  %s\n", route)
	}
	return b.String()
}

// BuildURL builds a request URL for the named route. buildArgs are forwarded
// to Route.ToURL; see that method for accepted shapes.
func (r *Router) BuildURL(name string, buildArgs ...any) *url.URL {
	route := r.GetRoute(name)
	if route == nil {
		panic("rux: BuildURL: unknown route " + name)
	}
	return route.ToURL(buildArgs...)
}

// BuildRequestURL is an alias of BuildURL retained for v1 compatibility.
func (r *Router) BuildRequestURL(name string, buildArgs ...any) *url.URL {
	return r.BuildURL(name, buildArgs...)
}

// Match looks up route + params for an offline test or debugging.
// The hot ServeHTTP path does NOT call this — it uses an internal
// signature that writes params directly into Context with zero allocation.
func (r *Router) Match(method, path string) (*Route, []Param, bool) {
	if !r.frozen.Load() {
		r.Freeze()
	}
	idx := methodIndex(strings.ToUpper(method))
	if idx < 0 {
		return nil, nil, false
	}
	path = r.formatPath(path)

	if m := r.staticRoutes[idx]; m != nil {
		if route, ok := m[path]; ok {
			return route, nil, true
		}
	}
	if tree := r.dynamicTrees[idx]; tree != nil {
		var ps Params
		if route, ok := tree.lookup(path, &ps); ok {
			return route, ps.Snapshot(), true
		}
	}
	return nil, nil, false
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
