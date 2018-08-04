package souter


/*************************************************************
 * middleware definition
 *************************************************************/

type HandlerFunc func(ctx *Context)
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
 * global middleware
 *************************************************************/

func (r *Router) Use(handlers ...HandlerFunc) {
	r.Handlers = append(r.Handlers, handlers...)
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