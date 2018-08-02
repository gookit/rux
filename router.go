package souter

import (
	"regexp"
	"strings"
	"net/http"
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

// Router definition
type Router struct {
	name string

	initialized  bool
	routeCounter int

	// static routes
	staticRoutes map[string]interface{}

	// stable routes
	stableRoutes map[string]interface{}

	// cached dynamic routes
	cachedRoutes map[string]interface{}

	// Regular dynamic routing 规律的动态路由
	regularRoutes map[string]interface{}

	// Irregular dynamic routing 无规律的动态路由
	irregularRoutes map[string]interface{}

	currentGroupPrefix  string
	currentGroupOption  string
	currentGroupMiddles []interface{}

	// global middleware list
	mds []interface{}

	// intercept all request. eg. "/site/error"
	interceptAll string
	// Ignore last slash char('/'). If is True, will clear last '/'.
	ignoreLastSlash bool
	// If True, the router checks if another method is allowed for the current route,
	handleMethodNotAllowed bool
}

// "/users/:id" "/users/:id(\d+)"
var varPattern = regexp.MustCompile(`:[a-zA-Z0-9]+`)
var anyPattern = regexp.MustCompile(`[^/]+`)

var globalPatterns = map[string]string{
	"all": `.*`,
	"any": `[^/]+`,
	"num": `[1-9][0-9]*`,
}

// New
func New() *Router {
	return &Router{name: "default", ignoreLastSlash:true}
}

func (r *Router) IgnoreLastSlash(ignoreLastSlash bool) *Router {
	r.ignoreLastSlash = ignoreLastSlash
	return r
}

func (r *Router) HandleMethodNotAllowed(val bool) *Router {
	r.handleMethodNotAllowed = val
	return r
}

func (r *Router) InterceptAll(interceptAll string) *Router {
	r.interceptAll = interceptAll
	return r
}

/*************************************************************
 * register routes
 *************************************************************/

// Add a route to router
func (r *Router) Add(method, path string, handler Handle, mds ...interface{}) *Route {
	path = r.formatPath(path)

	if handler == nil {
		panic("router: nil handler")
	}

	route := &Route{}

	return route
}

// AddRoute a route to router by Route
func (r *Router) AddRoute(route *Route) *Route {
	return route
}

func (r *Router) GET(path string, handler Handle, mds ...interface{}) *Route {
	route := &Route{}

	return route
}

func (r *Router) Group(path string, register func(*Router), mds ...interface{}) {
	prevPrefix := r.currentGroupPrefix
	r.currentGroupPrefix = prevPrefix + r.formatPath(path)

	prevMiddles := r.currentGroupMiddles
	if len(mds) > 0 {
		if len(prevMiddles) > 0 {
			r.currentGroupMiddles = append(r.currentGroupMiddles, prevMiddles...)
			r.currentGroupMiddles = append(r.currentGroupMiddles, mds...)
		} else {
			r.currentGroupMiddles = mds
		}
	}

	register(r)

	r.currentGroupPrefix = prevPrefix
	r.currentGroupMiddles = prevMiddles
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

	r.GET(path, func(w http.ResponseWriter, req *http.Request, ps Params) {
		req.URL.Path = ps.ByName("filepath")
		fileServer.ServeHTTP(w, req)
	})
}

/*************************************************************
 * helper methods
 *************************************************************/

func (r *Router) formatPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}

	if path[0] != '/' {
		path = "/" + path
	}

	if r.ignoreLastSlash {
		path = strings.TrimRight(path, "/")
	}

	return path
}
