package souter

import (
	"net/http"
	"strconv"
	"regexp"
)

type MiddlewareFunc func(http.Handler) http.Handler

// middleware interface is anything which implements a MiddlewareFunc named Middleware.
type middleware interface {
	Middleware(handler http.Handler) http.Handler
}

// Middleware allows MiddlewareFunc to implement the middleware interface.
func (fn MiddlewareFunc) Middleware(handler http.Handler) http.Handler {
	return fn(handler)
}

type Handle func(http.ResponseWriter, *http.Request, Params)

/*************************************************************
 * Route definition
 *************************************************************/

// Route in the router
type Route struct {
	Name   string
	method string

	Params Params

	// path/pattern definition for the route. eg "/users" "/users/{id}"
	pattern string

	// start string in the route pattern. "/users/{id}" -> "/user/"
	start string
	// first node string in the route pattern. "/users/{id}" -> "user"
	first string

	// regexp for the route pattern
	regexp *regexp.Regexp

	// handler for the route. eg. myFunc, &MyController.SomeAction
	Handler HandlerFunc
	// some options data for the route
	Opts map[string]interface{}
	// metadata
	meta map[string]string

	vars map[string]string

	// middleware list
	handlers HandlersChain
	// domains
	// defaults
}

// Use some middleware handlers
func (r *Route) Use(handlers ...HandlerFunc) *Route {
	r.handlers = append(r.handlers, handlers...)

	return r
}

// Vars add vars pattern for the route path
func (r *Route) Vars(vars map[string]string) *Route {
	for name, pattern := range vars {
		r.vars[name] = pattern
	}

	return r
}

func (r *Route) withMethod(method string) *Route {
	nr := &Route{
		method:   method,
		pattern:  r.pattern,
		Handler:  r.Handler,
		handlers: r.handlers,
	}

	return nr
}

func (r *Route) clone() *Route {
	nr := &Route{
		pattern:  r.pattern,
		Handler:  r.Handler,
		handlers: r.handlers,
	}

	return nr
}

/*************************************************************
 * Route params
 *************************************************************/

// Params for current route
type Params map[string]string

func (p Params) String(key string) (val string) {
	if val, ok := p[key]; ok {
		return val
	}

	return
}

func (p Params) Int(key string) (val int) {
	if str, ok := p[key]; ok {
		val, err := strconv.Atoi(str)
		if err != nil {
			return val
		}
	}

	return
}
