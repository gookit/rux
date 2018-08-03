package souter

import (
	"strings"
	"net/http"
	"bytes"
	"fmt"
)

const (
	ANY = "ANY"

	// all http verb methods
	GET     = "GET"
	PUT     = "PUT"
	HEAD    = "HEAD"
	POST    = "POST"
	PATCH   = "PATCH"
	DELETE  = "DELETE"
	OPTIONS = "OPTIONS"

	// some help constants
	FavIcon = "/favicon.ico"
	// supported methods string
	// more: ,COPY,PURGE,LINK,UNLINK,LOCK,UNLOCK,VIEW,SEARCH,CONNECT,TRACE
	MethodsStr = "ANY,GET,POST,PUT,PATCH,DELETE,OPTIONS,HEAD"

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

type tree struct {
	node *Route
}

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
	// 	"POST /users": Route,
	// }
	stableRoutes map[string]*Route

	// cached dynamic routes
	cachedRoutes map[string]*Route

	// Regular dynamic routing 规律的动态路由
	// {
	// 	"/blog": {
	// 		"GET": [ Route{pattern:"/blog/:id"}, ...]
	// 		"POST": [ Route{pattern:"/blog/:user/add"}]
	// 	},
	//	...
	// }
	regularRoutes map[string]methodRoutes

	// Irregular dynamic routing 无规律的动态路由
	// {
	// 	"GET": [Route, ...],
	// 	"POST": [Route, Route, ...],
	// }
	irregularRoutes methodRoutes

	currentGroupPrefix  string
	currentGroupHandlers HandlersChain

	noRoute   HandlersChain
	noAllowed HandlersChain
	Handlers   HandlersChain

	// intercept all request. eg. "/site/error"
	interceptAll string
	// Ignore last slash char('/'). If is True, will clear last '/'.
	ignoreLastSlash bool
	// If True, the router checks if another method is allowed for the current route,
	handleMethodNotAllowed bool
}

// "/users/:id" "/users/:id(\d+)"
var anyMethods = []string{GET,POST,PUT,PATCH,DELETE,OPTIONS,HEAD}

var globalPatterns = map[string]string{
	"all": `.*`,
	"any": `[^/]+`,
	"num": `[1-9][0-9]*`,
}

// New
func New() *Router {
	return &Router{name: "default", ignoreLastSlash: true}
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

	fmt.Fprintf(buf, "routes count: %d\n", r.routeCounter)

	fmt.Fprint(buf, "Stable(fixed) routes:\n")
	for uri, route := range r.stableRoutes {
		fmt.Fprintf(buf, " %s %v\n", uri, route)
	}

	fmt.Fprint(buf, "Regular(dynamic) routes:\n")
	for start, mrs := range r.regularRoutes {
		fmt.Fprintf(buf, " %s:\n", start)
		for m, routes := range mrs {
			fmt.Fprintf(buf, "  %s:\n", m)
			for _, route := range routes {
				fmt.Fprintf(buf, "  %s %s %v:\n", m, route.pattern, route)
			}
		}
	}

	fmt.Fprint(buf, "Irregular(dynamic) routes:\n")
	for m, routes := range r.irregularRoutes {
		fmt.Fprintf(buf, "  %s:\n", m)
		for _, route := range routes {
			fmt.Fprintf(buf, "  %s %s %v:\n", m, route.pattern, route)
		}
	}

	return buf.String()
}

/*************************************************************
 * register routes
 *************************************************************/

// Add a route to router
func (r *Router) Add(method, path string, handler HandlerFunc, handlers ...HandlerFunc) (route *Route) {
	if len(handlers) == 0 {
		panic("router: must set handler")
	}

	if r.currentGroupPrefix != "" {
		path = r.currentGroupPrefix + path
	}

	path = r.formatPath(path)
	method = strings.ToUpper(method)

	if strings.Index(MethodsStr + ",", method) == -1 {
		panic("router: invalid method name, must in: " + MethodsStr)
	}

	// create new route instance
	route = &Route{
		method:   method,
		pattern:  path,
		Handler:  handler,
		handlers: handlers,
	}

	// path is fixed. eg. "/users"
	if isFixedPath(path) {
		if method == ANY {
			for _, m := range anyMethods {
				key := m + " " + path
				r.routeCounter++
				r.stableRoutes[key] = route.withMethod(m)
			}
		} else {
			key := method + " " + path
			r.routeCounter++
			r.stableRoutes[key] = route
		}

		return
	}

	// parsing route path with parameters
	r.parseParamRoute(path, route)

	ms := []string{method}
	if method == ANY {
		ms = anyMethods
	}

	if route.first != "" {
		mrs, has := r.regularRoutes[route.first]
		if !has {
			mrs = methodRoutes{}
		}

		for _, m := range ms {
			rs, has := mrs[m]
			if has {
				rs = append(rs, route.withMethod(m))
			} else {
				rs = routes{route.withMethod(m)}
			}

			r.routeCounter++
			r.regularRoutes[route.first][m] = rs
		}
	} else {

	}

	return
}

func (r *Router) ANY(path string, handler HandlerFunc, handlers ...HandlerFunc) *Route {
	return r.Add(ANY, path, handler, handlers...)
}

func (r *Router) GET(path string, handler HandlerFunc, handlers ...HandlerFunc) *Route {
	return r.Add(GET, path, handler, handlers...)
}

func (r *Router) HEAD(path string, handler HandlerFunc, handlers ...HandlerFunc) *Route {
	return r.Add(HEAD, path, handler, handlers...)
}

func (r *Router) POST(path string, handler HandlerFunc, handlers ...HandlerFunc) *Route {
	return r.Add(POST, path, handler, handlers...)
}

func (r *Router) PUT(path string, handler HandlerFunc, handlers ...HandlerFunc) *Route {
	return r.Add(PUT, path, handler, handlers...)
}

func (r *Router) PATCH(path string, handler HandlerFunc, handlers ...HandlerFunc) *Route {
	return r.Add(PATCH, path, handler, handlers...)
}

func (r *Router) OPTIONS(path string, handler HandlerFunc, handlers ...HandlerFunc) *Route {
	return r.Add(OPTIONS, path, handler, handlers...)
}

func (r *Router) DELETE(path string, handler HandlerFunc, handlers ...HandlerFunc) *Route {
	return r.Add(DELETE, path, handler, handlers...)
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
 * global middleware
 *************************************************************/

func (r *Router) Use(handlers ...HandlerFunc) {
	r.Handlers = append(r.Handlers, handlers...)
}

/*************************************************************
 * route match
 *************************************************************/

func (r *Router) Match(method, path string) (route *Route, err error) {
	path = r.formatPath(path)

	return
}

/*************************************************************
 * running with http server
 *************************************************************/

func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	route, err := r.Match(req.Method, req.URL.Path)
	if err != nil {
		panic(err)
	}

	ctx := NewContext(res, req, r.Handlers)
	ctx.Params = route.Params

	// ctx.Next()
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
