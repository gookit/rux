package sux

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// all supported HTTP verb methods name
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

// AllMethods all supported methods string, use for method check
// more: ,COPY,PURGE,LINK,UNLINK,LOCK,UNLOCK,VIEW,SEARCH,CONNECT,TRACE
const AllMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS,HEAD,CONNECT,TRACE"

// match status: 1 found 2 not found 3 method not allowed
const (
	Found uint8 = iota + 1
	NotFound
	NotAllowed
)

type patternType int8

const (
	PatternStatic   patternType = iota // /home
	PatternRegexp                      // /:id([0-9]+)
	PatternPathExt                     // /*.*
	PatternHolder                      // /:user
	PatternMatchAll                    // /*
)

// IController a simple controller interface
type IController interface {
	// AddRoutes for support register routes in the controller.
	AddRoutes(g *Router)
}

var debug bool
var anyMethods = []string{GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD, CONNECT, TRACE}

// Debug switch debug mode
func Debug(val bool) {
	debug = val
}

// IsDebug return sux is debug mode.
func IsDebug() bool {
	return debug
}

/*************************************************************
 * Router definition
 *************************************************************/

type routes []*Route

// like "GET": [ Route, ...]
type methodRoutes map[string]routes

// Router definition
type Router struct {
	// sux rux
	name string
	pool sync.Pool

	counter int
	// mark init is completed
	initialized bool

	// static routes
	staticRoutes map[string]interface{}

	// stable/fixed routes
	// {
	// 	"GET /users": Route,
	// 	"POST /users/register": Route,
	// }
	stableRoutes map[string]*Route

	// cached dynamic routes
	// {
	// 	"GET /users/12": Route,
	// }
	cachedRoutes map[string]*Route

	// regular dynamic routing 规律的动态路由
	// key is "METHOD first-node":
	// first node string in the route path. "/users/{id}" -> "user"
	// {
	// 	"GET blog": [ Route{path:"/blog/:id"}, ...],
	// 	"POST blog": [ Route{path:"/blog/:user/add"}, ...],
	// 	"GET users": [ Route{path:"/users/:id"}, ...],
	// 	...
	// }
	regularRoutes methodRoutes

	// irregular dynamic routing 无规律的动态路由
	// {
	// 	"GET": [Route, ...],
	// 	"POST": [Route, Route, ...],
	// }
	irregularRoutes methodRoutes

	// some data for group
	currentGroupPrefix   string
	currentGroupHandlers HandlersChain

	// handlers chain
	noRoute   HandlersChain
	noAllowed HandlersChain
	handlers  HandlersChain

	//
	// Router Options:
	//
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
	// whether checks if another method is allowed for the current route. default is False
	handleMethodNotAllowed bool
}

// New router instance, can with some options.
// quick start:
// 		r := New()
// 		r.GET("/path", MyAction)
//
// with options:
// 		r := New(EnableCaching, MaxNumCaches(1000))
// 		r.GET("/path", MyAction)
//
func New(options ...func(*Router)) *Router {
	router := &Router{
		name: "default",

		maxNumCaches: 1000,
		stableRoutes: make(map[string]*Route),

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

// HandleMethodNotAllowed enable for the router
func HandleMethodNotAllowed(r *Router) {
	r.handleMethodNotAllowed = true
}

// WithOptions for the router
func (r *Router) WithOptions(options ...func(*Router)) {
	if r.initialized {
		panic("router: unable to set options after initialization is complete")
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
	return r.Add(GET, path, handler, middleware...)
}

// HEAD add routing and only allow HEAD request methods
func (r *Router) HEAD(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(HEAD, path, handler, middleware...)
}

// POST add routing and only allow POST request methods
func (r *Router) POST(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(POST, path, handler, middleware...)
}

// PUT add routing and only allow PUT request methods
func (r *Router) PUT(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(PUT, path, handler, middleware...)
}

// PATCH add routing and only allow PATCH request methods
func (r *Router) PATCH(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(PATCH, path, handler, middleware...)
}

// TRACE add routing and only allow TRACE request methods
func (r *Router) TRACE(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(TRACE, path, handler, middleware...)
}

// OPTIONS add routing and only allow OPTIONS request methods
func (r *Router) OPTIONS(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(OPTIONS, path, handler, middleware...)
}

// DELETE add routing and only allow OPTIONS request methods
func (r *Router) DELETE(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(DELETE, path, handler, middleware...)
}

// CONNECT add routing and only allow CONNECT request methods
func (r *Router) CONNECT(path string, handler HandlerFunc, middleware ...HandlerFunc) *Route {
	return r.Add(CONNECT, path, handler, middleware...)
}

// Any add route and allow any request methods
func (r *Router) Any(path string, handler HandlerFunc, middleware ...HandlerFunc) {
	for _, method := range anyMethods {
		r.Add(method, path, handler, middleware...)
	}
}

// Add a route to router
func (r *Router) Add(method, path string, handler HandlerFunc, middleware ...HandlerFunc) (route *Route) {
	if handler == nil {
		panic("router: must set handler for the route " + path)
	}

	if !r.initialized {
		r.initialized = true
	}

	if r.currentGroupPrefix != "" {
		path = r.currentGroupPrefix + r.formatPath(path)
	}

	if len(r.currentGroupHandlers) > 0 {
		// middleware = append(r.currentGroupHandlers, middleware...)
		middleware = combineHandlers(r.currentGroupHandlers, middleware)
	}

	path = r.formatPath(path)
	method = strings.ToUpper(method)
	if strings.Index(","+AllMethods, ","+method) == -1 {
		panic("router: invalid method name, must in: " + AllMethods)
	}

	// create new route instance
	route = newRoute(method, path, handler, middleware)

	// path is fixed(no param vars). eg. "/users"
	if r.isFixedPath(path) {
		key := method + " " + path
		r.counter++
		r.stableRoutes[key] = route
		debugPrintRoute(route)
		return
	}

	// parsing route path with parameters
	first := r.parseParamRoute(path, route)
	if first != "" {
		key := method + " " + first
		rs, has := r.regularRoutes[key]
		if !has {
			rs = routes{}
		}

		r.regularRoutes[key] = append(rs, route)
	} else {
		rs, has := r.irregularRoutes[method]
		if has {
			rs = routes{}
		}

		r.irregularRoutes[method] = append(rs, route)
	}

	r.counter++
	debugPrintRoute(route)
	return
}

// Group add an group routes
func (r *Router) Group(prefix string, register func(), middleware ...HandlerFunc) {
	prevPrefix := r.currentGroupPrefix
	r.currentGroupPrefix = prevPrefix + r.formatPath(prefix)

	// handle prev middleware
	prevHandlers := r.currentGroupHandlers
	if len(middleware) > 0 {
		// multi level group routes.
		if len(prevHandlers) > 0 {
			r.currentGroupHandlers = append(r.currentGroupHandlers, middleware...)
		} else {
			r.currentGroupHandlers = middleware
		}
	}

	// call register
	register()

	// revert
	r.currentGroupPrefix = prevPrefix
	r.currentGroupHandlers = prevHandlers
}

// Controller register some routes by a controller
func (r *Router) Controller(basePath string, controller IController, middleware ...HandlerFunc) {
	r.Group(basePath, func() {
		controller.AddRoutes(r)
	}, middleware...)
}

// NotFound handlers for router
func (r *Router) NotFound(handlers ...HandlerFunc) {
	r.noRoute = handlers
}

// NotAllowed handlers for router
func (r *Router) NotAllowed(handlers ...HandlerFunc) {
	r.noAllowed = handlers
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

// StaticHandle add a static asset file handle
func (r *Router) StaticFunc(path string, handler func(c *Context)) {
	r.GET(path, handler)
}

// StaticHandle add a static asset file handle
// usage:
// 		r.StaticDir("/assets", "/static")
// access GET /assets/css/site.css -> will find /static/css/site.css
func (r *Router) StaticDir(prefixUrl string, fileDir string) {
	fsHandler := http.StripPrefix(prefixUrl, http.FileServer(http.Dir(fileDir)))

	r.GET(prefixUrl+"/"+allMatch, func(c *Context) {
		fsHandler.ServeHTTP(c.Resp, c.Req)
	})
}

// StaticFiles static files from the given file system root. and allow limit extensions.
// usage:
//     router.ServeFiles("/src", "/var/www", "css|js|html")
func (r *Router) StaticFiles(prefixUrl string, rootDir string, exts string) {
	fsHandler := http.FileServer(http.Dir(rootDir))
	// ignore prefix when find real file.
	fsHandler = http.StripPrefix(prefixUrl, fsHandler)

	// eg "/assets/(?:.+\.(?:css|js|html))"
	r.GET(fmt.Sprintf(`%s/{file:.+\.(?:%s)}`, prefixUrl, exts), func(c *Context) {
		fsHandler.ServeHTTP(c.Resp, c.Req)
	})
}
