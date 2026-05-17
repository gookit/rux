package core

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

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

	// router is the back-pointer to the owning Router (set by the pool factory).
	router *Router

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
// If a message is supplied, it is written via http.Error (which sets the
// Content-Type to text/plain and writes the body).
func (c *Context) AbortWithStatus(status int, msg ...string) {
	if len(msg) > 0 {
		http.Error(c.Resp, msg[0], status)
	} else {
		c.Resp.WriteHeader(status)
	}
	c.Abort()
}

// AbortThen marks the context as aborted at the end of the current
// middleware run while still allowing the caller to chain further
// response-shaping calls (e.g. c.AbortThen().NoContent()).
func (c *Context) AbortThen() *Context {
	c.index = abortIndex
	return c
}

// Router returns the Router that dispatched this request.
func (c *Context) Router() *Router { return c.router }

// URL is a shortcut for c.Req.URL.
func (c *Context) URL() *url.URL { return c.Req.URL }

// Header returns the first request header value for key, or "" if absent.
func (c *Context) Header(key string) string {
	if values := c.Req.Header[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}

// Query returns the URL query value for key. If the key is absent and a
// default is supplied, the default is returned; otherwise "" is returned.
func (c *Context) Query(key string, defVal ...string) string {
	if vs, ok := c.Req.URL.Query()[key]; ok && len(vs) > 0 {
		return vs[0]
	}
	if len(defVal) > 0 {
		return defVal[0]
	}
	return ""
}

// ReqCtxValue returns the value associated with key on the request's
// context.Context, or nil if absent.
func (c *Context) ReqCtxValue(key any) any {
	return c.Req.Context().Value(key)
}

// ClientIP returns a best-effort client IP, consulting X-Forwarded-For,
// X-Real-Ip, and finally the connection RemoteAddr.
func (c *Context) ClientIP() string {
	clientIP := c.Header("X-Forwarded-For")
	if i := strings.IndexByte(clientIP, ','); i >= 0 {
		clientIP = clientIP[:i]
	}
	clientIP = strings.TrimSpace(clientIP)
	if clientIP != "" {
		return clientIP
	}
	if ip := strings.TrimSpace(c.Header("X-Real-Ip")); ip != "" {
		return ip
	}
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Req.RemoteAddr)); err == nil {
		return ip
	}
	return ""
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

// FirstError returns the first error recorded via AddError, or nil if none.
func (c *Context) FirstError() error {
	if len(c.Errors) == 0 {
		return nil
	}
	return c.Errors[0]
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

// Cookie reads a request cookie value, or "" if not set.
func (c *Context) Cookie(name string) string {
	if cookie, err := c.Req.Cookie(name); err == nil {
		return cookie.Value
	}
	return ""
}

// SetCookie sets a response cookie. Path defaults to "/" if empty.
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	if path == "" {
		path = "/"
	}
	http.SetCookie(c.Resp, &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		Secure:   secure,
		HttpOnly: httpOnly,
	})
}

// FastSetCookie sets a response cookie with developer-friendly defaults:
// path=/, httpOnly=true, secure=false. The Secure attribute is deliberately
// off so cookies work over plain HTTP during local development.
//
// SECURITY: for production HTTPS deployments use SetCookie directly with
// secure=true (and consider SameSite via http.SetCookie if you need it).
// This default is preserved for backward compatibility.
func (c *Context) FastSetCookie(name, value string, maxAge int) {
	c.SetCookie(name, value, maxAge, "/", "", false, true)
}

// DelCookie deletes one or more cookies by setting MaxAge=-1.
func (c *Context) DelCookie(names ...string) {
	for _, name := range names {
		c.SetCookie(name, "", -1, "/", "", false, false)
	}
}
