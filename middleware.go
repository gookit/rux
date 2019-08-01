package rux

import (
	"net/http"
)

/*************************************************************
 * middleware definition
 *************************************************************/

// HandlerFunc a handler definition
type HandlerFunc func(c *Context)

// HandlersChain middleware handlers chain definition
type HandlersChain []HandlerFunc

// ServeHTTP implement the http.Handler
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := &Context{}
	c.Init(w, r)
	f(c)
}

// Last returns the last handler in the chain. ie. the last handler is the main own.
func (c HandlersChain) Last() HandlerFunc {
	length := len(c)
	if length > 0 {
		return c[length-1]
	}
	return nil
}

// WrapHTTPHandler warp an generic http.Handler as an middleware HandlerFunc
func WrapHTTPHandler(gh http.Handler) HandlerFunc {
	return func(c *Context) {
		gh.ServeHTTP(c.Resp, c.Req)
	}
}

// WrapHTTPHandlerFunc warp an generic http.HandlerFunc as an middleware HandlerFunc
func WrapHTTPHandlerFunc(gh http.HandlerFunc) HandlerFunc {
	return func(c *Context) {
		gh(c.Resp, c.Req)
	}
}

/*************************************************************
 * global middleware
 *************************************************************/

// Use add handlers for the router
func (r *Router) Use(handlers ...HandlerFunc) {
	r.handlers = append(r.handlers, handlers...)
}

func combineHandlers(oldHandlers, newHandlers HandlersChain) HandlersChain {
	finalSize := len(oldHandlers) + len(newHandlers)
	mergedHandlers := make(HandlersChain, finalSize)

	copy(mergedHandlers, oldHandlers)
	copy(mergedHandlers[len(oldHandlers):], newHandlers)

	return mergedHandlers
}
