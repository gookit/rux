// Package sux is a simple and fast request router for golang HTTP applications.
//
// Source code and other details for the project are available at GitHub:
// 		https://github.com/gookit/sux
//
// usage please ref examples and README
package sux

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
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

// MethodsStr all supported methods string, use for method check
// more: ,COPY,PURGE,LINK,UNLINK,LOCK,UNLOCK,VIEW,SEARCH,CONNECT,TRACE
const MethodsStr = "GET,POST,PUT,PATCH,DELETE,OPTIONS,HEAD,CONNECT,TRACE"

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

	// intercept all request. eg. "/site/error"
	InterceptAll string
	// use encoded path for match route
	UseEncodedPath bool
	// maximum number of cached dynamic routes. default is 1000
	MaxCachedRoute uint16
	// cache recently accessed dynamic routes. default is False
	EnableRouteCache bool
	// Ignore last slash char('/'). If is True, will clear last '/'. default is True
	IgnoreLastSlash bool
	// If True, the router checks if another method is allowed for the current route. default is False
	HandleMethodNotAllowed bool
}

// New router instance
func New() *Router {
	router := &Router{
		name: "default",

		MaxCachedRoute:  1000,
		IgnoreLastSlash: true,

		stableRoutes:  make(map[string]*Route),
		regularRoutes: make(methodRoutes),

		irregularRoutes: make(methodRoutes),
	}

	router.pool.New = func() interface{} {
		return &Context{index: -1}
	}
	return router
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

	if r.currentGroupPrefix != "" {
		path = r.currentGroupPrefix + r.formatPath(path)
	}

	if len(r.currentGroupHandlers) > 0 {
		middleware = combineHandlers(r.currentGroupHandlers, middleware)
	}

	path = r.formatPath(path)
	method = strings.ToUpper(method)
	if strings.Index(","+MethodsStr, ","+method) == -1 {
		panic("router: invalid method name, must in: " + MethodsStr)
	}

	// create new route instance
	route = newRoute(method, path, handler, middleware)

	// path is fixed(no param vars). eg. "/users"
	if r.isFixedPath(path) {
		key := method + " " + path
		r.counter++
		r.stableRoutes[key] = route
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
	return
}

// Group add an group routes
func (r *Router) Group(prefix string, register func(), middleware ...HandlerFunc) {
	prevPrefix := r.currentGroupPrefix
	r.currentGroupPrefix = prevPrefix + r.formatPath(prefix)

	// handle prev middleware
	prevHandlers := r.currentGroupHandlers
	if len(middleware) > 0 {
		if len(prevHandlers) > 0 {
			r.currentGroupHandlers = append(r.currentGroupHandlers, prevHandlers...)
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
func (r *Router) StaticFile(path, file string) {
}

// StaticFunc add a static assets handle func
func (r *Router) StaticFunc(path string) {
}

// StaticHandle add a static asset file handle
func (r *Router) StaticHandle(path string) {
}

// StaticFiles static files from the given file system root.
// use http.Dir:
//     router.ServeFiles("/src/*filepath", http.Dir("/var/www"))
func (r *Router) StaticFiles(path string, root http.FileSystem) {
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("path must end with /*filepath in path '" + path + "'")
	}

	fileServer := http.FileServer(root)
	r.GET(path, func(c *Context) {
		req := c.Req
		req.URL.Path = c.Param("filepath")
		fileServer.ServeHTTP(c.Resp, req)
	})
}

/*************************************************************
 * help methods
 *************************************************************/

// String all routes to string
func (r *Router) String() string {
	buf := new(bytes.Buffer)

	fmt.Fprintf(buf, "Routes Count: %d\n", r.counter)

	fmt.Fprint(buf, "Stable(fixed):\n")
	for _, route := range r.stableRoutes {
		fmt.Fprintf(buf, " %s\n", route)
	}

	fmt.Fprint(buf, "Regular(dynamic):\n")
	for pfx, routes := range r.regularRoutes {
		fmt.Fprintf(buf, " %s:\n", pfx)
		for _, route := range routes {
			fmt.Fprintf(buf, "   %s\n", route.String())
		}
	}

	fmt.Fprint(buf, "Irregular(dynamic):\n")
	for m, routes := range r.irregularRoutes {
		fmt.Fprintf(buf, " %s:\n", m)
		for _, route := range routes {
			fmt.Fprintf(buf, "   %s\n", route.String())
		}
	}

	return buf.String()
}

func debugPrintRoute(route *Route) {
	if debug {
		fmt.Println(route.String())
	}
}

func debugPrintError(err error) {
	if err != nil {
		debugPrint("[ERROR] %v\n", err)
	}
}

func debugPrint(f string, v ...interface{}) {
	if debug {
		msg := fmt.Sprintf(f, v...)
		fmt.Printf("[sux-DEBUG] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	}
}
