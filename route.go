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

type CtxMdlFunc func(ctx *Context, next func())

type CtxHandle func(ctx *Context)
type Handle func(http.ResponseWriter, *http.Request, Params)
type Params map[string]string

// Route in the router
type Route struct {
	Name string
	method string

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

func (r *Route) Chain(h http.Handler) http.Handler {
	for i := range r.mds {
		next := len(r.mds)-1-i
		h = r.mds[next].Middleware(h)
	}

	return h
}

type HandlerFunc func(*Context)

// Context for http server
type Context struct {
	Req *http.Request
	Res http.ResponseWriter

	Params Params

	// context data
	Data map[string]interface{}

	index    int8
	handlers HandlersChain
}

func(c *Context) Next() {
	c.index++
	s := int8(len(c.handlers))

	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

type HandlersChain []HandlerFunc

// Last returns the last handler in the chain. ie. the last handler is the main own.
func (c HandlersChain) Last() HandlerFunc {
	length := len(c)
	if length > 0 {
		return c[length-1]
	}
	return nil
}