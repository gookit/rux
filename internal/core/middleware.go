package core

import (
	"net/http"
)

/*************************************************************
 * middleware definition
 *************************************************************/

// ServeHTTP adapts a HandlerFunc to the http.Handler interface.
// Used when a single HandlerFunc is mounted into a generic http.Handler stack
// outside the Router's dispatch path; flushes the deferred status code.
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := &Context{}
	c.Init(w, r)
	f(c)
	c.writer.ensureWriteHeader()
}

// Last returns the last handler in the chain (i.e. the main handler).
func (c HandlersChain) Last() HandlerFunc {
	length := len(c)
	if length > 0 {
		return c[length-1]
	}
	return nil
}

// WrapH wraps a generic http.Handler as a rux HandlerFunc.
func WrapH(hh http.Handler) HandlerFunc {
	return WrapHTTPHandler(hh)
}

// HTTPHandler wraps a generic http.Handler as a rux HandlerFunc.
func HTTPHandler(gh http.Handler) HandlerFunc {
	return WrapHTTPHandler(gh)
}

// WrapHTTPHandler wraps a generic http.Handler as a rux HandlerFunc.
func WrapHTTPHandler(gh http.Handler) HandlerFunc {
	return func(c *Context) {
		gh.ServeHTTP(c.Resp, c.Req)
	}
}

// WrapHF wraps a generic http.HandlerFunc as a rux HandlerFunc.
func WrapHF(hf http.HandlerFunc) HandlerFunc {
	return WrapHTTPHandlerFunc(hf)
}

// HTTPHandlerFunc wraps a generic http.HandlerFunc as a rux HandlerFunc.
func HTTPHandlerFunc(hf http.HandlerFunc) HandlerFunc {
	return WrapHTTPHandlerFunc(hf)
}

// WrapHTTPHandlerFunc wraps a generic http.HandlerFunc as a rux HandlerFunc.
func WrapHTTPHandlerFunc(hf http.HandlerFunc) HandlerFunc {
	return func(c *Context) {
		hf(c.Resp, c.Req)
	}
}

// combineHandlers concatenates two handler chains into a fresh slice.
func combineHandlers(oldHandlers, newHandlers HandlersChain) HandlersChain {
	finalSize := len(oldHandlers) + len(newHandlers)
	mergedHandlers := make(HandlersChain, finalSize)
	copy(mergedHandlers, oldHandlers)
	copy(mergedHandlers[len(oldHandlers):], newHandlers)
	return mergedHandlers
}
