package souter

import (
	"net/http"
	"strconv"
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
	Name string
	method string

	Params Params

	// path/pattern definition for the route. eg "/users" "/users/{id}"
	pattern string
	// handler for the route. eg. myFunc, &MyController.SomeAction
	Handler HandlerFunc
	// some options data for the route
	Opts map[string]interface{}
	// metadata
	meta map[string]string

	// middleware list
	Handlers HandlersChain
	// domains
	// defaults
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
