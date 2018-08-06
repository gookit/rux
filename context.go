package sux

import (
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
)

/*************************************************************
 * Context
 *************************************************************/

const (
	abortIndex int8 = math.MaxInt8 / 2
)

// IContext interface for http context
type IContext interface {
	Req() *http.Request
	Res() http.ResponseWriter
	Init(http.ResponseWriter, *http.Request, HandlersChain)

	Next()
	Reset()
	Params() Params
	SetParams(Params)

	HandlerName() string
}

// DefContext for http server
type DefContext struct {
}

// Context for http server
type Context struct {
	req *http.Request
	res http.ResponseWriter

	index int8
	// current route params, if route has var params
	params Params
	// context data, you can save some custom data.
	values map[string]interface{}
	// all handlers for current request
	handlers HandlersChain
}

// Init a context
func (c *Context) Init(res http.ResponseWriter, req *http.Request, handlers HandlersChain) {
	c.res = res
	c.req = req
	c.values = make(map[string]interface{})
	c.handlers = handlers
}

// HandlerName get the main handler name
func (c *Context) HandlerName() string {
	return nameOfFunction(c.handlers.Last())
}

// Handler returns the main handler.
func (c *Context) Handler() HandlerFunc {
	return c.handlers.Last()
}

// Values get all values
func (c *Context) Values() map[string]interface{} {
	return c.values
}

// Set a value to context by key
func (c *Context) Set(key string, val interface{}) {
	c.values[key] = val
}

// Get a value from context
func (c *Context) Get(key string) interface{} {
	return c.values[key]
}

// Next call next handler
func (c *Context) Next() {
	c.index++
	s := int8(len(c.handlers))

	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

// AppendHandlers to the context
func (c *Context) AppendHandlers(handlers ...HandlerFunc) {
	c.handlers = append(c.handlers, handlers...)
}

// Reset context data
func (c *Context) Reset() {
	// c.Writer = &c.writermem
	c.params = nil
	c.handlers = nil
	c.index = -1
	c.values = nil
	// c.Errors = c.Errors[0:0]
	// c.Accepted = nil
}

// Copy a new context
func (c *Context) Copy() *Context {
	var ctx = *c
	ctx.handlers = nil
	ctx.index = abortIndex

	return &ctx
}

/*************************************************************
 * getter/setter methods
 *************************************************************/

// Req get request instance
func (c *Context) Req() *http.Request {
	return c.req
}

// Res get response instance
func (c *Context) Res() http.ResponseWriter {
	return c.res
}

// Params get current route params
func (c *Context) Params() Params {
	return c.params
}

// SetParams to the context
func (c *Context) SetParams(params Params) {
	c.params = params
}

/*************************************************************
 * Context: input data
 *************************************************************/

// Param returns the value of the URL param.
//		router.GET("/user/{id}", func(c *gin.Context) {
//			// a GET request to /user/john
//			id := c.Param("id") // id == "john"
//		})
func (c *Context) Param(key string) string {
	return c.params.String(key)
}

// URL get URL instance from request
func (c *Context) URL() *url.URL {
	return c.req.URL
}

// URLQuery return query Values
func (c *Context) URLQuery() url.Values {
	return c.req.URL.Query()
}

// Query return query value by key
func (c *Context) Query(key string) string {
	if vs, ok := c.req.URL.Query()[key]; ok && len(vs) > 0 {
		return vs[0]
	}

	return ""
}

// RawData return stream data
func (c *Context) RawData() ([]byte, error) {
	return ioutil.ReadAll(c.req.Body)
}

/*************************************************************
 * Context: response data
 *************************************************************/

// Write byte data to response
func (c *Context) Write(bt []byte) (n int, err error) {
	return c.res.Write(bt)
}

// WriteString to response
func (c *Context) WriteString(str string) (n int, err error) {
	return c.res.Write([]byte(str))
}

// Text writes out a string as plain text.
func (c *Context) Text(status int, str string) (n int, err error) {
	c.res.WriteHeader(status)
	c.res.Header().Set("Content-Type", "text/plain; charset=UTF-8")

	return c.res.Write([]byte(str))
}

// NoContent serve success but no content response
func (c *Context) NoContent() error {
	c.res.WriteHeader(http.StatusNoContent)
	return nil
}
