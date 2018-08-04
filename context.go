package souter

import (
	"net/http"
	"math"
	"io/ioutil"
)

/*************************************************************
 * Context
 *************************************************************/

const (
	abortIndex    int8 = math.MaxInt8 / 2
)

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

func newContext(res http.ResponseWriter, req *http.Request, handlers HandlersChain) *Context {
	return &Context{Res: res, Req: req, handlers: handlers}
}

func (c *Context) HandlerName() string {
	return nameOfFunction(c.handlers.Last())
}

// Handler returns the main handler.
func (c *Context) Handler() HandlerFunc {
	return c.handlers.Last()
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

func (c *Context) Copy() *Context {
	var ctx = *c
	ctx.handlers = nil
	ctx.index = abortIndex

	return &ctx
}

// appendHandlers
func (c *Context) appendHandlers(handlers ...HandlerFunc) {
	c.handlers = append(c.handlers, handlers...)
}

func (c *Context) reset() {
	// c.Writer = &c.writermem
	c.Params = nil
	c.handlers = nil
	c.index = -1
	c.values = nil
	// c.Errors = c.Errors[0:0]
	// c.Accepted = nil
}

/*************************************************************
 * Context helper methods
 *************************************************************/

// GetRawData return stream data
func (c *Context) GetRawData() ([]byte, error) {
	return ioutil.ReadAll(c.Req.Body)
}
