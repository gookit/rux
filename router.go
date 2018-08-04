package souter

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

const (
	// all http verb methods
	GET     = "GET"
	PUT     = "PUT"
	HEAD    = "HEAD"
	POST    = "POST"
	PATCH   = "PATCH"
	TRACE   = "TRACE"
	DELETE  = "DELETE"
	CONNECT = "CONNECT"
	OPTIONS = "OPTIONS"

	// some help constants
	FavIcon = "/favicon.ico"
	// supported methods string
	// more: ,COPY,PURGE,LINK,UNLINK,LOCK,UNLOCK,VIEW,SEARCH,CONNECT,TRACE
	MethodsStr = "GET,POST,PUT,PATCH,DELETE,OPTIONS,HEAD,CONNECT,TRACE"

	// match status
	Found      = 1
	NotFound   = 2
	NotAllowed = 3 // method not allowed
)

type patternType int8

const (
	PatternStatic   patternType = iota // /home
	PatternRegexp                      // /:id([0-9]+)
	PatternPathExt                     // /*.*
	PatternHolder                      // /:user
	PatternMatchAll                    // /*
)

/*************************************************************
 * Router definition
 *************************************************************/

type routes []*Route

// "GET": [ Route, ...]
type methodRoutes map[string]routes

// Router definition
type Router struct {
	name string

	initialized  bool
	routeCounter int

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
	// first node string in the route pattern. "/users/{id}" -> "user"
	// {
	// 	"GET blog": [ Route{pattern:"/blog/:id"}, ...],
	// 	"POST blog": [ Route{pattern:"/blog/:user/add"}, ...],
	// 	"GET users": [ Route{pattern:"/users/:id"}, ...],
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

	noRoute   HandlersChain
	noAllowed HandlersChain
	Handlers  HandlersChain

	// intercept all request. eg. "/site/error"
	interceptAll string
	// Ignore last slash char('/'). If is True, will clear last '/'.
	ignoreLastSlash bool
	// If True, the router checks if another method is allowed for the current route,
	handleMethodNotAllowed bool
}

var anyMethods = []string{GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD, CONNECT, TRACE}

// New
func New() *Router {
	return &Router{
		name: "default",

		ignoreLastSlash: true,

		stableRoutes:  make(map[string]*Route),
		cachedRoutes:  make(map[string]*Route),
		regularRoutes: make(methodRoutes),

		irregularRoutes: make(methodRoutes),
	}
}

// IgnoreLastSlash
func (r *Router) IgnoreLastSlash(ignoreLastSlash bool) *Router {
	r.ignoreLastSlash = ignoreLastSlash
	return r
}

// HandleMethodNotAllowed
func (r *Router) HandleMethodNotAllowed(val bool) *Router {
	r.handleMethodNotAllowed = val
	return r
}

// InterceptAll
func (r *Router) InterceptAll(interceptAll string) *Router {
	r.interceptAll = interceptAll
	return r
}

// String
func (r *Router) String() string {
	buf := new(bytes.Buffer)

	fmt.Fprintf(buf, "Routes Count: %d\n", r.routeCounter)

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

/*************************************************************
 * register routes
 *************************************************************/

func (r *Router) GET(path string, handlers ...HandlerFunc) *Route {
	return r.Add(GET, path, handlers...)
}

func (r *Router) HEAD(path string, handlers ...HandlerFunc) *Route {
	return r.Add(HEAD, path, handlers...)
}

func (r *Router) POST(path string, handlers ...HandlerFunc) *Route {
	return r.Add(POST, path, handlers...)
}

func (r *Router) PUT(path string, handlers ...HandlerFunc) *Route {
	return r.Add(PUT, path, handlers...)
}

func (r *Router) PATCH(path string, handlers ...HandlerFunc) *Route {
	return r.Add(PATCH, path, handlers...)
}

func (r *Router) TRACE(path string, handlers ...HandlerFunc) *Route {
	return r.Add(TRACE, path, handlers...)
}

func (r *Router) OPTIONS(path string, handlers ...HandlerFunc) *Route {
	return r.Add(OPTIONS, path, handlers...)
}

func (r *Router) DELETE(path string, handlers ...HandlerFunc) *Route {
	return r.Add(DELETE, path, handlers...)
}

func (r *Router) CONNECT(path string, handlers ...HandlerFunc) *Route {
	return r.Add(CONNECT, path, handlers...)
}

func (r *Router) ANY(path string, handlers ...HandlerFunc) {
	for _, method := range anyMethods {
		r.Add(method, path, handlers...)
	}
}

// Add a route to router
func (r *Router) Add(method, path string, handlers ...HandlerFunc) (route *Route) {
	if len(handlers) == 0 {
		panic("router: must set handler for the route " + path)
	}

	if r.currentGroupPrefix != "" {
		path = r.currentGroupPrefix + r.formatPath(path)
	}

	if len(r.currentGroupHandlers) > 0 {
		handlers = combineHandlers(r.currentGroupHandlers, handlers)
	}

	path = r.formatPath(path)
	method = strings.ToUpper(method)

	if strings.Index(MethodsStr+",", method) == -1 {
		panic("router: invalid method name, must in: " + MethodsStr)
	}

	// create new route instance
	r.routeCounter++
	route = newRoute(method, path, handlers)

	// path is fixed(no param vars). eg. "/users"
	if r.isFixedPath(path) {
		key := method + " " + path
		r.stableRoutes[key] = route

		return
	}

	// parsing route path with parameters
	first := r.parseParamRoute(path, route)
	if first != "" {
		rKey := method + " " + first
		rs, has := r.regularRoutes[rKey]
		if has {
			rs = append(rs, route)
		} else {
			rs = routes{route}
		}

		r.regularRoutes[rKey] = rs
	} else {
		rs, has := r.irregularRoutes[method]
		if has {
			rs = append(rs, route)
		} else {
			rs = routes{route}
		}

		r.irregularRoutes[method] = rs
	}

	return
}

func (r *Router) Group(path string, register func(grp *Router), handlers ...HandlerFunc) {
	prevPrefix := r.currentGroupPrefix
	r.currentGroupPrefix = prevPrefix + r.formatPath(path)

	prevHandlers := r.currentGroupHandlers
	if len(handlers) > 0 {
		if len(prevHandlers) > 0 {
			r.currentGroupHandlers = append(r.currentGroupHandlers, prevHandlers...)
			r.currentGroupHandlers = append(r.currentGroupHandlers, handlers...)
		} else {
			r.currentGroupHandlers = handlers
		}
	}

	register(r)

	r.currentGroupPrefix = prevPrefix
	r.currentGroupHandlers = prevHandlers
}

// IController
type IController interface {
	// [":id": c.GetAction]
	AddRoutes(grp *Router)
}

// Controller
func (r *Router) Controller(basePath string, controller IController, handlers ...HandlerFunc) {
	r.Group(basePath, controller.AddRoutes)
}

// NotFound
func (r *Router) NotFound(handlers ...HandlerFunc) {
	r.noRoute = handlers
}

// NotAllowed
func (r *Router) NotAllowed(handlers ...HandlerFunc) {
	r.noAllowed = handlers
}

/*************************************************************
 * route match
 *************************************************************/

// Match route by given request METHOD and URI path
func (r *Router) Match(method, path string) (status uint8, route *Route, allowed []string) {
	path = r.formatPath(path)
	status = NotFound
	method = strings.ToUpper(method)

	// do match
	status, route = r.match(method, path)
	if status == Found {
		return
	}

	// for HEAD requests, attempt fallback to GET
	if method == HEAD {
		status, route = r.match(method, path)
		if status == Found {
			return
		}
	}

	// don't handle method not allowed, will return not found
	if !r.handleMethodNotAllowed {
		return
	}

	// find allowed methods
	allowed = r.findAllowedMethods(method, path)
	if len(allowed) > 0 {
		status = NotAllowed
	}

	return
}

func (r *Router) match(method, path string) (status uint8, route *Route) {
	// find in stable routes
	key := method + " " + path
	if route, ok := r.stableRoutes[key]; ok {
		return Found, route
	}

	// find in cached routes
	if route, ok := r.cachedRoutes[key]; ok {
		return Found, route
	}

	// find in regular routes
	if pos := strings.Index(path[1:], "/"); pos > 1 {
		first := path[1 : pos-1]
		key = method + " " + first

		if rs, ok := r.regularRoutes[key]; ok {
			for _, route := range rs {
				if route.match(path) {
					// always return a new Route instance
					return Found, route.withParams(route.Params)
				}
			}
		}
	}

	// find in irregular routes
	if rs, ok := r.irregularRoutes[method]; ok {
		for _, route := range rs {
			if route.match(path) {
				return Found, route.withParams(route.Params)
			}
		}
	}

	status = NotFound
	return
}

func (r *Router) findAllowedMethods(method, path string) (allowed []string) {
	// in stable routes
	for _, m := range anyMethods {
		if m == method {
			continue
		}

		key := m + " " + path
		if _, ok := r.stableRoutes[key]; ok {
			allowed = append(allowed, m)
		}
	}

	// in regular routes
	if pos := strings.Index(path[1:], "/"); pos > 1 {
		for _, m := range anyMethods {
			if m == method {
				continue
			}

			first := path[1 : pos-1]
			key := m + " " + first

			if rs, ok := r.regularRoutes[key]; ok {
				for _, route := range rs {
					if route.match(path) {
						allowed = append(allowed, m)
					}
				}
			}
		}
	}

	// in irregular routes
	for _, m := range anyMethods {
		if m == method {
			continue
		}

		if rs, ok := r.irregularRoutes[m]; ok {
			for _, route := range rs {
				if route.match(path) {
					allowed = append(allowed, m)
				}
			}
		}
	}

	return
}

/*************************************************************
 * static file handle methods
 *************************************************************/

func (r *Router) StaticFile(path string) {
}

func (r *Router) StaticFunc(path string) {
}

func (r *Router) StaticHandle(path string) {
}

// StaticFiles static files from the given file system root.
// The path must end with "/*filepath", files are then served from the local
// path /defined/root/dir/*filepath.
// For example if root is "/etc" and *filepath is "passwd", the local file
// "/etc/passwd" would be served.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
// To use the operating system's file system implementation,
// use http.Dir:
//     router.ServeFiles("/src/*filepath", http.Dir("/var/www"))
func (r *Router) StaticFiles(path string, root http.FileSystem) {
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("path must end with /*filepath in path '" + path + "'")
	}

	fileServer := http.FileServer(root)

	r.GET(path, func(ctx *Context) {
		req := ctx.Req
		req.URL.Path = ctx.Params["filepath"]
		fileServer.ServeHTTP(ctx.Res, req)
	})
}
