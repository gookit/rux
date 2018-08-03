package souter

import (
	"net/http"
)

/*************************************************************
 * HandlerFunc and HandlersChain
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

func (c *Context) Values() map[string]interface{} {
	return c.values
}

func (c *Context) SetValue(key string, val interface{}) {
	c.values[key] = val
}

func (c *Context) Value(key string) interface{} {
	return c.values[key]
}

func (c *Context) Next() {
	c.index++
	s := int8(len(c.handlers))

	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}
