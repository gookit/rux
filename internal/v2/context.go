package v2

import "net/http"

// Context carries per-request state through the handler chain.
// Full version (with renderers, binding, error chain) is built up in Phase 4/5.
// This skeleton is the minimum for Route/HandlerFunc compilation.
type Context struct {
	Req  *http.Request
	Resp http.ResponseWriter

	// writer is the embedded response wrapper; Resp aliases &writer after Init.
	writer responseWriter

	// Path params, inlined for zero-allocation parameter passing.
	params Params

	// Hot fields used by every request — typed, not in a map.
	matchedRoute *Route
	matchedPath  string

	// Handler chain — already merged at Freeze time, so no per-request append.
	handlers HandlersChain
	index    int8

	// Errors accumulated during handling.
	Errors []error

	// Lazy-init bag for arbitrary user data.
	data map[string]any

	// Renderer (optional) used by Context.Render for templated views.
	Renderer Renderer
}

// Init prepares c for a new request. Field-only — no slice reallocation.
func (c *Context) Init(w http.ResponseWriter, req *http.Request) {
	c.Req = req
	c.writer.reset(w)
	c.Resp = &c.writer
	c.params.Reset()
	c.matchedRoute = nil
	c.matchedPath = ""
	c.handlers = nil
	c.index = -1
	if c.Errors != nil {
		c.Errors = c.Errors[:0]
	}
	if c.data != nil {
		for k := range c.data {
			delete(c.data, k)
		}
	}
}

// SetStatus writes the HTTP status code to the response.
func (c *Context) SetStatus(status int) { c.Resp.WriteHeader(status) }

// SetHeader sets a response header value.
func (c *Context) SetHeader(key, value string) { c.Resp.Header().Set(key, value) }

// SetHandlers installs the (already merged) handler chain.
func (c *Context) SetHandlers(chain HandlersChain) {
	c.handlers = chain
	c.index = -1
}

// Next advances to the next handler in the chain.
func (c *Context) Next() {
	c.index++
	for c.index < int8(len(c.handlers)) {
		c.handlers[c.index](c)
		c.index++
	}
}

// Abort prevents subsequent handlers from running.
func (c *Context) Abort() { c.index = abortIndex }

// IsAborted reports whether Abort was called.
func (c *Context) IsAborted() bool { return c.index >= abortIndex }

// AbortWithStatus aborts and writes the given HTTP status.
func (c *Context) AbortWithStatus(status int) {
	c.Resp.WriteHeader(status)
	c.Abort()
}

// AddError records an error to be processed by Router.OnError.
func (c *Context) AddError(err error) {
	if err == nil {
		return
	}
	c.Errors = append(c.Errors, err)
}

// Err returns the most recent error or nil.
func (c *Context) Err() error {
	if len(c.Errors) == 0 {
		return nil
	}
	return c.Errors[len(c.Errors)-1]
}

// SafeGet returns the value or panics — for required keys.
func (c *Context) SafeGet(key string) any {
	v, ok := c.Get(key)
	if !ok {
		panic("rux: missing context key " + key)
	}
	return v
}

// Param returns the value of the named path parameter, or "" if absent.
func (c *Context) Param(name string) string { return c.params.Get(name) }

// Params returns a pointer to the inlined params (avoids 16-Param value copy).
func (c *Context) Params() *Params { return &c.params }

// Route returns the matched Route or nil.
func (c *Context) Route() *Route { return c.matchedRoute }

// MatchedPath returns the route's registered path with placeholders.
func (c *Context) MatchedPath() string { return c.matchedPath }

// Set stores arbitrary user data. Allocates the map on first call.
func (c *Context) Set(key string, value any) {
	if c.data == nil {
		c.data = make(map[string]any, 4)
	}
	c.data[key] = value
}

// Get retrieves user data set by Set.
func (c *Context) Get(key string) (any, bool) {
	if c.data == nil {
		return nil, false
	}
	v, ok := c.data[key]
	return v, ok
}

// StatusCode returns the HTTP status code that will be (or was) written.
func (c *Context) StatusCode() int { return c.writer.Status() }

// Length returns the number of body bytes written so far.
func (c *Context) Length() int { return c.writer.Length() }

// WriteBytes writes raw bytes to the response, panicking on I/O error.
func (c *Context) WriteBytes(bt []byte) {
	_, err := c.Resp.Write(bt)
	if err != nil {
		panic(err)
	}
}

// WriteString writes a string to the response.
func (c *Context) WriteString(str string) { c.WriteBytes([]byte(str)) }
