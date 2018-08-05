package souter

import (
	"net/http"
	"math"
	"io/ioutil"
	"context"
	"net/url"
)

/*************************************************************
 * Context
 *************************************************************/

const (
	abortIndex int8 = math.MaxInt8 / 2
)

type IContext interface {
	Req() *http.Request
	Res() http.ResponseWriter

	Next()
	Params() Params

	HandlerName() string
}

// Context for http server
type Context struct {
	req *http.Request
	res http.ResponseWriter

	index int8
	// current route params
	params Params
	// context data
	values map[string]interface{}
	//
	handlers HandlersChain
}

func newContext(res http.ResponseWriter, req *http.Request, handlers HandlersChain) *Context {
	return &Context{
		res: res,
		req: req,

		index:  -1,
		values: make(map[string]interface{}),

		handlers: handlers,
	}
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

func (c *Context) Set(key string, val interface{}) {
	c.values[key] = val
}

func (c *Context) Get(key string) interface{} {
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
	c.params = nil
	c.handlers = nil
	c.index = -1
	c.values = nil
	// c.Errors = c.Errors[0:0]
	// c.Accepted = nil
}

/*************************************************************
 * getter methods
 *************************************************************/

func (c *Context) Req() *http.Request {
	return c.req
}

func (c *Context) Res() http.ResponseWriter {
	return c.res
}

func (c *Context) Params() Params {
	return c.params
}

/*************************************************************
 * Context helper methods
 *************************************************************/

// GetRawData return stream data
func (c *Context) GetRawData() ([]byte, error) {
	return ioutil.ReadAll(c.req.Body)
}

func (c *Context) WriteString(str string) (n int, err error) {
	return c.res.Write([]byte(str))
}

func (c *Context) Write(bt []byte) (n int, err error) {
	return c.res.Write(bt)
}

func (c *Context) WriteBytes(bt []byte) (n int, err error) {
	return c.res.Write(bt)
}

func (c *Context) URL() *url.URL {
	return c.req.URL
}

func contextGet(r *http.Request, key interface{}) interface{} {
	return r.Context().Value(key)
}

func contextSet(r *http.Request, key, val interface{}) *http.Request {
	if val == nil {
		return r
	}

	return r.WithContext(context.WithValue(r.Context(), key, val))
}
