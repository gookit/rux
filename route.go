package souter

import "net/http"

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
type Params map[string]string

// Route in the router
type Route struct {
	Name string
	// path/pattern definition for the route. eg "/users" "/users/{id}"
	Pattern string
	// handler for the route. eg. myFunc, &MyController.SomeAction
	Handler interface{}
	// middleware list
	mds []middleware
	// some options data for the route
	Opts map[string]interface{}
	// metadata
	meta map[string]string

	// domains
	// defaults
}

func (r *Route) Chain(h http.Handler) http.Handler {
	for i := range r.mds {
		next := len(r.mds)-1-i
		h = r.mds[next].Middleware(h)
	}

	return h
}
