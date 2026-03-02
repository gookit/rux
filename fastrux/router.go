package fastrux

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

/*************************************************************
 * Router definition
 *************************************************************/

// Renderer interface
type Renderer interface {
	Render(io.Writer, string, any, *Context) error
}

// Router definition
type Router struct {
	// router name
	Name string
	// server start error
	err error
	// count routes
	counter int
	// context pool
	ctxPool sync.Pool
	// match result pool
	matchResultPool sync.Pool

	// Static/stable/fixed routes, no path params.
	// {
	// 	"GET/users": Route,
	// 	"POST/users/register": Route,
	// }
	stableRoutes map[string]*Route

	// Dynamic routes using Radix Tree for high-performance matching
	dynamicTrees *methodTrees

	// storage named routes. {"name": Route}
	namedRoutes map[string]*Route

	// some data for group
	currentGroupPrefix   string
	currentGroupHandlers HandlersChain

	// handlers chain
	noRoute   HandlersChain
	noAllowed HandlersChain
	handlers  HandlersChain

	//
	// Router Settings:
	//

	// OnError on happen error
	OnError HandlerFunc
	// OnPanic on happen panic
	OnPanic HandlerFunc
	// intercept all request, then redirect to the path
	interceptAll string
	// use encoded path for match route
	useEncodedPath bool
	// strict match last slash char('/')
	strictLastSlash bool
	// whether checks if another method is allowed for the current route
	handleMethodNotAllowed bool
	// whether handle the fallback route "/*"
	handleFallbackRoute bool

	// Renderer template(view) interface
	Renderer Renderer
}

// New router instance, can with some options.
//
// Quick start:
//
//	r := New()
//	r.GET("/path", MyAction)
//
// With options:
//
//	r := New(StrictLastSlash, HandleMethodNotAllowed)
//	r.GET("/path", MyAction)
func New(options ...func(*Router)) *Router {
	router := &Router{
		Name:         "default",
		stableRoutes: make(map[string]*Route),
		namedRoutes:  make(map[string]*Route),
		dynamicTrees: newMethodTrees(),
	}

	// with some options
	router.WithOptions(options...)
	router.ctxPool.New = func() any {
		return &Context{index: -1, router: router}
	}
	router.matchResultPool.New = func() any {
		return &MatchResult{}
	}

	return router
}

// WithOptions for the router
func (r *Router) WithOptions(options ...func(*Router)) {
	if r.counter > 0 {
		panic("router: unable to set options after add route")
	}

	for _, opt := range options {
		opt(r)
	}
}

/*************************************************************
 * register routes
 *************************************************************/

// GET add routing and only allow GET request methods
func (r *Router) GET(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(path, handler, GET).Use(middleware...)
}

// HEAD add routing and only allow HEAD request methods
func (r *Router) HEAD(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(path, handler, HEAD).Use(middleware...)
}

// POST add routing and only allow POST request methods
func (r *Router) POST(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(path, handler, POST).Use(middleware...)
}

// PUT add routing and only allow PUT request methods
func (r *Router) PUT(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(path, handler, PUT).Use(middleware...)
}

// PATCH add routing and only allow PATCH request methods
func (r *Router) PATCH(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(path, handler, PATCH).Use(middleware...)
}

// TRACE add routing and only allow TRACE request methods
func (r *Router) TRACE(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(path, handler, TRACE).Use(middleware...)
}

// OPTIONS add routing and only allow OPTIONS request methods
func (r *Router) OPTIONS(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(path, handler, OPTIONS).Use(middleware...)
}

// DELETE add routing and only allow DELETE request methods
func (r *Router) DELETE(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(path, handler, DELETE).Use(middleware...)
}

// CONNECT add routing and only allow CONNECT request methods
func (r *Router) CONNECT(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(path, handler, CONNECT).Use(middleware...)
}

// Any add route and allow any request methods
func (r *Router) Any(path string, handler HandlerFunc, middles ...HandlerFunc) {
	route := NewRoute(path, handler, anyMethods...)
	route.Use(middles...)

	r.AddRoute(route)
}

// Add a route to router, allow set multi method
// Usage:
//
//	r.Add("/path", myHandler)
//	r.Add("/path1", myHandler, "GET", "POST")
func (r *Router) Add(path string, handler HandlerFunc, methods ...string) *Route {
	route := NewRoute(path, handler, methods...)
	return r.AddRoute(route)
}

// AddNamed add a named route to router, allow set multi method
func (r *Router) AddNamed(name, path string, handler HandlerFunc, methods ...string) *Route {
	route := NewNamedRoute(name, path, handler, methods...)
	return r.AddRoute(route)
}

// AddRoute add a route by Route instance.
func (r *Router) AddRoute(route *Route) *Route {
	r.appendRoute(route)
	return route
}

// Group add a group routes, can with middleware
func (r *Router) Group(prefix string, register func(), middles ...HandlerFunc) {
	prevPrefix := r.currentGroupPrefix
	r.currentGroupPrefix = prevPrefix + r.formatPath(prefix)

	// handle prev middleware
	prevHandlers := r.currentGroupHandlers
	if len(middles) > 0 {
		// in multi level group routes.
		if len(prevHandlers) > 0 {
			r.currentGroupHandlers = append(r.currentGroupHandlers, middles...)
		} else {
			r.currentGroupHandlers = middles
		}
	}

	// call register
	register()

	// revert
	r.currentGroupPrefix = prevPrefix
	r.currentGroupHandlers = prevHandlers
}

// Controller register some routes by a controller
func (r *Router) Controller(basePath string, controller ControllerFace, middles ...HandlerFunc) {
	r.Group(basePath, func() {
		controller.AddRoutes(r)
	}, middles...)
}

// Resource register RESTFul style routes by a controller
//
//	Methods     Path                Action    Route Name
//	GET        /resource            index    resource_index
//	GET        /resource/create     create   resource_create
//	POST       /resource            store    resource_store
//	GET        /resource/{id}       show     resource_show
//	GET        /resource/{id}/edit  edit     resource_edit
//	PUT/PATCH  /resource/{id}       update   resource_update
//	DELETE     /resource/{id}       delete   resource_delete
func (r *Router) Resource(basePath string, controller any, middles ...HandlerFunc) {
	cv := reflect.ValueOf(controller)
	ct := cv.Type()

	if cv.Kind() != reflect.Ptr {
		panic("controller must type ptr")
	}

	if cv.Elem().Type().Kind() != reflect.Struct {
		panic("controller must type struct")
	}

	var handlerFuncs = make(map[string][]HandlerFunc)

	// can custom add middleware for actions
	if m := cv.MethodByName("Uses"); m.IsValid() {
		if uses, ok := m.Interface().(func() map[string][]HandlerFunc); ok {
			handlerFuncs = uses()
		}
	}

	resName := strings.ToLower(ct.Elem().Name())
	basePath += resName

	r.Group(basePath, func() {
		for name, methods := range RESTFulActions {
			m := cv.MethodByName(name)
			if !m.IsValid() {
				continue
			}

			action, ok := m.Interface().(func(*Context))
			if !ok {
				continue
			}

			var route *Route

			routeName := resName + "_" + strings.ToLower(name)
			if name == IndexAction || name == StoreAction {
				route = r.AddNamed(routeName, "/", action, methods...)
			} else if name == CreateAction {
				route = r.AddNamed(routeName, "/"+strings.ToLower(name)+"/", action, methods...)
			} else if name == EditAction {
				route = r.AddNamed(routeName, "{id}/"+strings.ToLower(name)+"/", action, methods...)
			} else { // if name == SHOW || name == UPDATE || name == DELETE
				route = r.AddNamed(routeName, "{id}/", action, methods...)
			}

			if handlers, ok := handlerFuncs[name]; ok {
				route.Use(handlers...)
			}
		}
	}, middles...)
}

// NotFound handlers for router
func (r *Router) NotFound(handlers ...HandlerFunc) {
	r.noRoute = handlers
}

// NotAllowed handlers for router
func (r *Router) NotAllowed(handlers ...HandlerFunc) {
	r.noAllowed = handlers
}

// Handlers get global handlers
func (r *Router) Handlers() HandlersChain {
	return r.handlers
}

/*************************************************************
 * static assets file handle
 *************************************************************/

// StaticFile add a static asset file handle
func (r *Router) StaticFile(path, filePath string) {
	r.GET(path, func(c *Context) {
		c.File(filePath)
	})
}

// StaticFunc add a static asset file handle
func (r *Router) StaticFunc(path string, handler func(c *Context)) {
	r.GET(path, handler)
}

// StaticFS add a file system handle.
func (r *Router) StaticFS(prefixURL string, fs http.FileSystem) {
	fsHandler := http.StripPrefix(prefixURL, http.FileServer(fs))

	r.GET(prefixURL+`/*file`, func(c *Context) {
		fsHandler.ServeHTTP(c.Resp, c.Req)
	})
}

// StaticDir add a static asset file handle
//
// Usage:
//
//	r.StaticDir("/assets", "/static")
//	// access GET /assets/css/site.css -> will find /static/css/site.css
func (r *Router) StaticDir(prefixURL string, fileDir string) {
	fsHandler := http.StripPrefix(prefixURL, http.FileServer(http.Dir(fileDir)))

	r.GET(prefixURL+`/*file`, func(c *Context) {
		fsHandler.ServeHTTP(c.Resp, c.Req)
	})
}

// StaticFiles static files from the given file system root. and allow limit extensions.
//
// Usage:
//
//	router.StaticFiles("/src", "/var/www", "css|js|html")
//
// Notice: if the rootDir is relation path, it is relative the server runtime dir.
func (r *Router) StaticFiles(prefixURL string, rootDir string, exts string) {
	fsHandler := http.FileServer(http.Dir(rootDir))

	r.GET(fmt.Sprintf(`%s/*file`, prefixURL), func(c *Context) {
		c.Req.URL.Path = c.Param("file")
		fsHandler.ServeHTTP(c.Resp, c.Req)
	})
}

/*************************************************************
 * help methods
 *************************************************************/

// GetRoute get a named route.
func (r *Router) GetRoute(name string) *Route {
	return r.namedRoutes[name]
}

// NamedRoutes get all named routes.
func (r *Router) NamedRoutes() map[string]*Route {
	return r.namedRoutes
}

// Routes get all route basic info
func (r *Router) Routes() (rs []RouteInfo) {
	r.IterateRoutes(func(route *Route) {
		rs = append(rs, route.Info())
	})
	return
}

// IterateRoutes iterate all routes
func (r *Router) IterateRoutes(fn func(route *Route)) {
	for _, route := range r.stableRoutes {
		fn(route)
	}
}

// String convert all routes to string
func (r *Router) String() string {
	buf := new(bytes.Buffer)
	_, _ = fmt.Fprintf(buf, "Routes Count: %d\n", r.counter)

	_, _ = fmt.Fprint(buf, "Stable(fixed):\n")
	for _, route := range r.stableRoutes {
		_, _ = fmt.Fprintf(buf, " %s\n", route)
	}

	_, _ = fmt.Fprint(buf, "Dynamic(Radix Tree): (see dynamicTrees)\n")

	return buf.String()
}

// BuildURL build Request URL one arg can be set buildRequestURL or M
func (r *Router) BuildURL(name string, buildArgs ...any) string {
	route := r.GetRoute(name)
	if route == nil {
		return ""
	}

	return route.ToURL(buildArgs...).String()
}

// Err get
func (r *Router) Err() error {
	return r.err
}

func (r *Router) formatPath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}

	path = strings.TrimSpace(path)
	// clear last slash: '/'
	if !r.strictLastSlash && path[len(path)-1] == '/' {
		path = strings.TrimRight(path, "/")
	}

	if path == "" || path == "/" {
		return "/"
	}

	// fix: "home" -> "/home"
	if path[0] != '/' {
		return "/" + path
	}

	// fix: "//home" -> "/home"
	if path[1] == '/' {
		return "/" + strings.TrimLeft(path, "/")
	}
	return path
}

func (r *Router) appendRoute(route *Route) {
	// route check: methods, handler
	route.goodInfo()

	// Initialize handlers chain from handler if needed
	if route.handler != nil && len(route.handlers) == 0 {
		route.handlers = []HandlerFunc{route.handler}
	}

	// format path and append group info
	r.appendGroupInfo(route)
	// print debug info
	debugPrintRoute(route)

	// has route name.
	if route.name != "" {
		r.namedRoutes[route.name] = route
	}

	// Check for optional segments - expand into multiple routes
	if hasOptionalSegment(route.path) {
		validateOptionalSegments(route.path)
		expandedPaths := parseOptionalSegments(route.path)

		for _, expandedPath := range expandedPaths {
			expandedPath = normalizePath(expandedPath)
			expandedRoute := *route
			expandedRoute.path = expandedPath
			r.registerSingleRoute(&expandedRoute)
		}
		return
	}

	r.registerSingleRoute(route)
}

// registerSingleRoute registers a single route (without optional segment expansion)
func (r *Router) registerSingleRoute(route *Route) {
	// path is fixed (no param vars). eg. "/users"
	if isFixedPath(route.path) {
		path := route.path
		for _, method := range route.methods {
			key := method + path
			r.counter++
			r.stableRoutes[key] = route
		}
		return
	}

	// Use Radix Tree for dynamic routes
	// Convert path format: {param} -> :param
	radixPath := convertParamSyntax(route.path)
	for _, method := range route.methods {
		tree := r.dynamicTrees.ensureTree(method)
		tree.AddRouteWithRoute(radixPath, route.handlers, route.methods, route)
	}
	r.counter++
}

func (r *Router) appendGroupInfo(route *Route) {
	routePath := r.formatPath(route.path)
	if r.currentGroupPrefix != "" {
		routePath = r.formatPath(r.currentGroupPrefix + routePath)
	}

	if len(r.currentGroupHandlers) > 0 {
		route.handlers = combineHandlers(r.currentGroupHandlers, route.handlers)

		if finalSize := len(route.handlers); finalSize >= int(abortIndex) {
			panic(fmt.Sprintf("too many handlers(number: %d)", finalSize))
		}
	}

	// re-set formatted path
	route.path = routePath
}

// hasOptionalSegment checks if path contains optional segments
func hasOptionalSegment(path string) bool {
	inBraces := false
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '{':
			inBraces = true
		case '}':
			inBraces = false
		case '[':
			if !inBraces {
				if i+1 < len(path) && path[i+1] == '/' {
					return true
				}
			}
		}
	}
	return false
}
