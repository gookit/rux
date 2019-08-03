package rux

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// All supported HTTP verb methods name
const (
	GET     = "GET"
	PUT     = "PUT"
	HEAD    = "HEAD"
	POST    = "POST"
	PATCH   = "PATCH"
	TRACE   = "TRACE"
	DELETE  = "DELETE"
	CONNECT = "CONNECT"
	OPTIONS = "OPTIONS"
)

// StringMethods all supported methods string, use for method check
// more: ,COPY,PURGE,LINK,UNLINK,LOCK,UNLOCK,VIEW,SEARCH
const StringMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS,HEAD,CONNECT,TRACE"

// Match status:
// - 1: found
// - 2: not found
// - 3: method not allowed
const (
	Found uint8 = iota + 1
	NotFound
	NotAllowed
)

// ControllerFace a simple controller interface
type ControllerFace interface {
	// AddRoutes for support register routes in the controller.
	AddRoutes(g *Router)
}

var (
	debug bool
	// current supported HTTP method
	anyMethods = []string{GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD, CONNECT, TRACE}
)

// Debug switch debug mode
func Debug(val bool) {
	debug = val
}

// IsDebug return rux is debug mode.
func IsDebug() bool {
	return debug
}

// AnyMethods get
func AnyMethods() []string {
	return anyMethods
}

/*************************************************************
 * Router definition
 *************************************************************/

// Binder interface
type Binder interface {
	Bind(i interface{}, c *Context) error
}

// Renderer interface
type Renderer interface {
	Render(io.Writer, string, interface{}, *Context) error
}

// Validator interface
type Validator interface {
	Validate(i interface{}) error
}

type routes []*Route

// like "GET": [ Route, ...]
type methodRoutes map[string]routes

// Router definition
type Router struct {
	// router name
	Name string
	// context pool
	pool sync.Pool
	// count routes
	counter int

	// Static/stable/fixed routes, no path params.
	// {
	// 	"GET /users": Route,
	// 	"POST /users/register": Route,
	// }
	stableRoutes map[string]*Route

	// Cached dynamic routes
	// {
	// 	"GET /users/12": Route,
	// }
	cachedRoutes map[string]*Route

	// Regular dynamic routing
	// - key is "METHOD first-node":
	// - first node string in the route path. "/users/{id}" -> "user"
	// Data example:
	// {
	// 	"GET blog": [ Route{path:"/blog/{id}"}, ...],
	// 	"POST blog": [ Route{path:"/blog/{user}/add"}, ...],
	// 	"GET users": [ Route{path:"/users/{id}"}, ...],
	// 	...
	// }
	regularRoutes methodRoutes

	// Irregular dynamic routing
	// {
	// 	"GET": [Route, ...],
	// 	"POST": [Route, Route, ...],
	// }
	irregularRoutes methodRoutes

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
	// intercept all request, then redirect to the path. eg. "/coming-soon" "/in-maintenance"
	interceptAll string
	// maximum number of cached dynamic routes. default is 1000
	maxNumCaches uint16
	// cache recently accessed dynamic routes. default is False
	enableCaching bool
	// use encoded path for match route. default is False
	useEncodedPath bool
	// strict last slash char('/'). If is True, will strict compare last '/'. default is False
	strictLastSlash bool
	// the max memory limit for multipart forms
	// maxMultipartMemory int64
	// whether checks if another method is allowed for the current route. default is False
	handleMethodNotAllowed bool
	// bind form,params,json body,query value to struct interface
	Binder Binder
	// template(view) interface
	Renderer Renderer
	// validator interface
	Validator Validator
}

// New router instance, can with some options.
// Quick start:
// 	r := New()
// 	r.GET("/path", MyAction)
//
// With options:
// 	r := New(EnableCaching, MaxNumCaches(1000))
// 	r.GET("/path", MyAction)
//
func New(options ...func(*Router)) *Router {
	router := &Router{
		Name: "default",

		maxNumCaches: 1000,
		stableRoutes: make(map[string]*Route),
		namedRoutes:  make(map[string]*Route),

		regularRoutes:   make(methodRoutes),
		irregularRoutes: make(methodRoutes),
	}

	// with some options
	router.WithOptions(options...)
	router.pool.New = func() interface{} {
		return &Context{index: -1, router: router}
	}

	return router
}

/*************************************************************
 * Router options
 *************************************************************/

// InterceptAll setting for the router
func InterceptAll(path string) func(*Router) {
	return func(r *Router) {
		r.interceptAll = strings.TrimSpace(path)
	}
}

// MaxNumCaches setting for the router
func MaxNumCaches(num uint16) func(*Router) {
	return func(r *Router) {
		r.maxNumCaches = num
	}
}

// UseEncodedPath enable for the router
func UseEncodedPath(r *Router) {
	r.useEncodedPath = true
}

// EnableCaching for the router
func EnableCaching(r *Router) {
	r.enableCaching = true
}

// StrictLastSlash enable for the router
func StrictLastSlash(r *Router) {
	r.strictLastSlash = true
}

// MaxMultipartMemory set max memory limit for post forms
// func MaxMultipartMemory(max int64) func(*Router) {
// 	return func(r *Router) {
// 		r.maxMultipartMemory = max
// 	}
// }

// HandleMethodNotAllowed enable for the router
func HandleMethodNotAllowed(r *Router) {
	r.handleMethodNotAllowed = true
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

// DELETE add routing and only allow OPTIONS request methods
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
//	r.Add("/path", myHandler)
//	r.Add("/path1", myHandler, "GET", "POST")
func (r *Router) Add(path string, handler HandlerFunc, methods ...string) *Route {
	route := NewRoute(path, handler, methods...)
	return r.AddRoute(route)
}

// AddNamed add an named route to router, allow set multi method
func (r *Router) AddNamed(name, path string, handler HandlerFunc, methods ...string) *Route {
	route := NewNamedRoute(name, path, handler, methods...)
	return r.AddRoute(route)
}

// AddRoute add a route by Route instance. , methods ...string
func (r *Router) AddRoute(route *Route) *Route {
	r.appendRoute(route)
	return route
}

func (r *Router) appendRoute(route *Route) {
	// route check: methods, handler
	route.goodInfo()
	// format path and append group info
	r.appendGroupInfo(route)
	// print debug info
	debugPrintRoute(route)

	// has route name.
	if route.name != "" {
		r.namedRoutes[route.name] = route
	}

	// path is fixed(no param vars). eg. "/users"
	if isFixedPath(route.path) {
		path := route.path
		for _, method := range route.methods {
			key := method + " " + path

			r.counter++
			r.stableRoutes[key] = route
		}
		return
	}

	// parsing route path with parameters
	if first := r.parseParamRoute(route); first != "" {
		for _, method := range route.methods {
			key := method + " " + first
			rs, has := r.regularRoutes[key]
			if !has {
				rs = routes{}
			}

			r.counter++
			r.regularRoutes[key] = append(rs, route)
		}
		return
	}

	// it's irregular param route
	for _, method := range route.methods {
		rs, has := r.irregularRoutes[method]
		if has {
			rs = routes{}
		}

		r.counter++
		r.irregularRoutes[method] = append(rs, route)
	}
}

func (r *Router) appendGroupInfo(route *Route) {
	path := r.formatPath(route.path)
	if r.currentGroupPrefix != "" {
		path = r.formatPath(r.currentGroupPrefix + path)
	}

	if len(r.currentGroupHandlers) > 0 {
		route.handlers = combineHandlers(r.currentGroupHandlers, route.handlers)

		if finalSize := len(route.handlers); finalSize >= int(abortIndex) {
			panicf("too many handlers(number: %d)", finalSize)
		}
	}

	// re-set formatted path
	route.path = path
}

// Group add an group routes, can with middleware
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

	r.GET(prefixURL+`/{file:.+}`, func(c *Context) {
		fsHandler.ServeHTTP(c.Resp, c.Req)
	})
}

// StaticDir add a static asset file handle
// Usage:
// 	r.StaticDir("/assets", "/static")
// 	// access GET /assets/css/site.css -> will find /static/css/site.css
func (r *Router) StaticDir(prefixURL string, fileDir string) {
	fsHandler := http.StripPrefix(prefixURL, http.FileServer(http.Dir(fileDir)))

	r.GET(prefixURL+`/{file:.+}`, func(c *Context) {
		// c.Req.URL.Path = c.Param("file") // can also.
		fsHandler.ServeHTTP(c.Resp, c.Req)
	})
}

// StaticFiles static files from the given file system root. and allow limit extensions.
// Usage:
// 	router.ServeFiles("/src", "/var/www", "css|js|html")
//
// Notice: if the rootDir is relation path, it is relative the server runtime dir.
func (r *Router) StaticFiles(prefixURL string, rootDir string, exts string) {
	fsHandler := http.FileServer(http.Dir(rootDir))

	// eg "/assets/(?:.+\.(?:css|js|html))"
	r.GET(fmt.Sprintf(`%s/{file:.+\.(?:%s)}`, prefixURL, exts), func(c *Context) {
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

	for _, routes := range r.regularRoutes {
		for _, route := range routes {
			fn(route)
		}
	}

	for _, routes := range r.irregularRoutes {
		for _, route := range routes {
			fn(route)
		}
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

	_, _ = fmt.Fprint(buf, "Regular(dynamic):\n")
	for pfx, routes := range r.regularRoutes {
		_, _ = fmt.Fprintf(buf, " %s:\n", pfx)
		for _, route := range routes {
			_, _ = fmt.Fprintf(buf, "   %s\n", route.String())
		}
	}

	_, _ = fmt.Fprint(buf, "Irregular(dynamic):\n")
	for m, routes := range r.irregularRoutes {
		_, _ = fmt.Fprintf(buf, " %s:\n", m)
		for _, route := range routes {
			_, _ = fmt.Fprintf(buf, "   %s\n", route.String())
		}
	}

	return buf.String()
}

func (r *Router) formatPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "/"
	}

	if path[0] != '/' {
		path = "/" + path
	}

	if !r.strictLastSlash {
		path = strings.TrimRight(path, "/")
	}

	return path
}
