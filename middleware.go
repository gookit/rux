package sux

/*************************************************************
 * middleware definition
 *************************************************************/

// HandlerFunc a handler definition
type HandlerFunc func(c *Context)

// HandlersChain a handlers chain
type HandlersChain []HandlerFunc

// Len get handles number
func (c HandlersChain) Len() int {
	return len(c)
}

// Last returns the last handler in the chain. ie. the last handler is the main own.
func (c HandlersChain) Last() HandlerFunc {
	length := len(c)
	if length > 0 {
		return c[length-1]
	}

	return nil
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
		panic("too many handlers")
	}

	mergedHandlers := make(HandlersChain, finalSize)

	copy(mergedHandlers, oldHandlers)
	copy(mergedHandlers[len(oldHandlers):], newHandlers)

	return mergedHandlers
}
