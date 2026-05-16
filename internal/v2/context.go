package v2

import "net/http"

// Context carries per-request state through the handler chain.
// Full version (with renderers, binding, error chain) is built up in Phase 4/5.
// This skeleton is the minimum for Route/HandlerFunc compilation.
type Context struct {
	Req  *http.Request
	Resp http.ResponseWriter

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
}

// Init prepares c for a new request. Field-only — no slice reallocation.
func (c *Context) Init(w http.ResponseWriter, req *http.Request) {
	c.Req = req
	c.Resp = w
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
