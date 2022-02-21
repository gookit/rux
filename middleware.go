package rux

import (
	"net/http"
)

/*************************************************************
 * middleware definition
 *************************************************************/

// HandlerFunc a handler definition
type HandlerFunc func(c *Context)

// ServeHTTP implement the http.Handler
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := &Context{}
	c.Init(w, r)
	f(c)
}

// HandlersChain middleware handlers chain definition
type HandlersChain []HandlerFunc

// Last returns the last handler in the chain. ie. the last handler is the main own.
func (c HandlersChain) Last() HandlerFunc {
	length := len(c)
	if length > 0 {
		return c[length-1]
	}
	return nil
}

// WrapH warp an generic http.Handler as an rux HandlerFunc
func WrapH(hh http.Handler) HandlerFunc {
	return WrapHTTPHandler(hh)
}

// HTTPHandler warp a generic http.Handler as an rux HandlerFunc
func HTTPHandler(gh http.Handler) HandlerFunc {
	return WrapHTTPHandler(gh)
}

// WrapHTTPHandler warp a generic http.Handler as an rux HandlerFunc
func WrapHTTPHandler(gh http.Handler) HandlerFunc {
	return func(c *Context) {
		gh.ServeHTTP(c.Resp, c.Req)
	}
}

// WrapHF warp a generic http.HandlerFunc as a rux HandlerFunc
func WrapHF(hf http.HandlerFunc) HandlerFunc {
	return WrapHTTPHandlerFunc(hf)
}

// HTTPHandlerFunc warp a generic http.HandlerFunc as a rux HandlerFunc
func HTTPHandlerFunc(hf http.HandlerFunc) HandlerFunc {
	return WrapHTTPHandlerFunc(hf)
}

// WrapHTTPHandlerFunc warp a generic http.HandlerFunc as a rux HandlerFunc
func WrapHTTPHandlerFunc(hf http.HandlerFunc) HandlerFunc {
	return func(c *Context) {
		hf(c.Resp, c.Req)
	}
}

/*************************************************************
 * global middleware
 *************************************************************/

// Use add handlers/middles for the router or group
func (r *Router) Use(middles ...HandlerFunc) {
	// use method in Group()
	if r.currentGroupPrefix != "" {
		r.currentGroupHandlers = append(r.currentGroupHandlers, middles...)
		return
	}

	// global middleware
	r.handlers = append(r.handlers, middles...)
}

func combineHandlers(oldHandlers, newHandlers HandlersChain) HandlersChain {
	finalSize := len(oldHandlers) + len(newHandlers)
	mergedHandlers := make(HandlersChain, finalSize)

	copy(mergedHandlers, oldHandlers)
	copy(mergedHandlers[len(oldHandlers):], newHandlers)

	return mergedHandlers
}
