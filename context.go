package souter

import (
	"net/http"
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
 * Middleware(HandlerFunc and HandlersChain)
 *************************************************************/

type HandlerFunc func(ctx *Context)
type ContextFunc func(ctx *Context)

type HandlersChain []HandlerFunc

// Last returns the last handler in the chain. ie. the last handler is the main own.
func (c HandlersChain) Last() HandlerFunc {
	length := len(c)
	if length > 0 {
		return c[length-1]
	}

	return nil
}

/*************************************************************
 * Context
 *************************************************************/

// Context for http server
type Context struct {
	Req *http.Request
	Res http.ResponseWriter

	// current route params
	Params Params

	// context data
	values map[string]interface{}

	index    int8
	handlers HandlersChain
}

func NewContext(res http.ResponseWriter, req *http.Request, handlers HandlersChain) *Context {
	return &Context{Res: res, Req: req, handlers: handlers}
}

func (c *Context) Values() map[string]interface{} {
	return c.values
}

func (c *Context) SetValue(key string, val interface{}) {
	c.values[key] = val
}

func (c *Context) Value(key string) interface{} {
	return c.values[key]
}

// AppendHandlers
func (c *Context) AppendHandlers(handlers ...HandlerFunc) {
	c.handlers = append(c.handlers, handlers...)
}

func (c *Context) Next() {
	c.index++
	s := int8(len(c.handlers))

	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}
