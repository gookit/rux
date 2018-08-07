package sux

import "net/http"

/*************************************************************
 * middleware definition
 *************************************************************/

// HandlerFunc a handler definition
type HandlerFunc func(c *Context)

// WarpHttpHandler warp an generic http.Handler as an middleware HandlerFunc
func WarpHttpHandler(gh http.Handler) HandlerFunc {
	return func(c *Context) {
		gh.ServeHTTP(c.Resp, c.Req)
	}
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

// LastName get the main handler name
func (c HandlersChain) LastName() string {
	return nameOfFunction(c.Last())
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
	if finalSize >= int(abortIndex) {
		panic("router: too many handlers")
	}

	mergedHandlers := make(HandlersChain, finalSize)

	copy(mergedHandlers, oldHandlers)
	copy(mergedHandlers[len(oldHandlers):], newHandlers)

	return mergedHandlers
}
