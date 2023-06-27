package rux

import (
	"context"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gookit/goutil"
	"github.com/gookit/goutil/netutil/httpctype"
)

/*************************************************************
 * Context
 *************************************************************/

const (
	defaultMaxMemory = 32 << 20 // 32 MB
	// abortIndex int8 = math.MaxInt8 / 2
	abortIndex int8 = 63
)

// M a short name for `map[string]any`
type M map[string]any

// Context for http server
type Context struct {
	Req  *http.Request
	Resp http.ResponseWriter
	// extended ResponseWriter
	writer responseWriter
	// current route Params, if route has var Params
	Params Params
	Errors []error

	index int8
	// current router instance
	router *Router
	// context data, you can save some custom data.
	data map[string]any
	// all handlers for current request.
	// call priority: global -> group -> route -> main handler
	// Notice: last always is main handler of the matched route.
	handlers HandlersChain
	// components for context
}

// Init a context
func (c *Context) Init(w http.ResponseWriter, r *http.Request) {
	c.writer.reset(w)
	c.Req = r
	c.Reset()
}

// RawWriter get raw http.ResponseWriter instance
func (c *Context) RawWriter() http.ResponseWriter {
	return c.writer.Writer
}

// Abort will abort at the end of this middleware run
func (c *Context) Abort() {
	c.index = abortIndex
}

// IsAborted returns true if the current context was aborted.
func (c *Context) IsAborted() bool {
	return c.index >= abortIndex
}

// AbortThen will abort at the end of this middleware run, and return context to continue.
func (c *Context) AbortThen() *Context {
	c.index = abortIndex
	return c
}

// AbortWithStatus calls `Abort()` and writes the headers with the specified status code.
func (c *Context) AbortWithStatus(code int, msg ...string) {
	if len(msg) == 0 {
		c.Resp.WriteHeader(code)
	} else {
		http.Error(c.Resp, msg[0], code)
	}

	c.Abort()
}

// Next processing, run all handlers
func (c *Context) Next() {
	c.index++
	s := int8(len(c.handlers))
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

// Reset context data
func (c *Context) Reset() {
	c.index = -1
	c.data = nil
	c.Resp = &c.writer
	c.Params = nil
	c.handlers = c.handlers[:0]
	c.Errors = c.Errors[:0]
	// c.Accepted = nil
}

// Copy a new context
func (c *Context) Copy() *Context {
	var ctx = *c
	ctx.writer.Writer = nil
	ctx.Resp = &ctx.writer
	ctx.handlers = nil
	ctx.index = abortIndex
	return &ctx
}

// Set a value to context by key.
// Usage:
//
//	c.Set("key", "value")
//	// ...
//	val := c.Get("key") // "value"
func (c *Context) Set(key string, val any) {
	if c.data == nil {
		c.data = make(map[string]any)
	}
	c.data[key] = val
}

// Get a value from context data
func (c *Context) Get(key string) (v any, ok bool) {
	v, ok = c.data[key]
	return
}

// SafeGet a value from context data
func (c *Context) SafeGet(key string) any { return c.data[key] }

// Data get all context data
func (c *Context) Data() map[string]any { return c.data }

// Handler returns the main handler.
func (c *Context) Handler() HandlerFunc { return c.handlers.Last() }

// HandlerName get the main handler name
func (c *Context) HandlerName() string { return goutil.FuncName(c.handlers.Last()) }

// SetHandlers set handlers
func (c *Context) SetHandlers(handlers HandlersChain) { c.handlers = handlers }

// Router get router instance
func (c *Context) Router() *Router { return c.router }

// AddError add a error to context
func (c *Context) AddError(err error) {
	if err != nil {
		c.Errors = append(c.Errors, err)
	}
}

// FirstError get first error
func (c *Context) FirstError() error {
	if len(c.Errors) > 0 {
		return c.Errors[0]
	}
	return nil
}

/*************************************************************
 * Context: request data
 *************************************************************/

// Param returns the value of the URL param.
//
//	router.GET("/user/{id}", func(c *rux.Context) {
//		// a GET request to /user/john
//		id := c.Param("id") // id == "john"
//	})
func (c *Context) Param(key string) string { return c.Params.String(key) }

// Header return header value by key
func (c *Context) Header(key string) string {
	if values, _ := c.Req.Header[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}

// URL get URL instance from request
func (c *Context) URL() *url.URL { return c.Req.URL }

// Query return query value by key, and allow with default value
func (c *Context) Query(key string, defVal ...string) string {
	val, has := c.QueryParam(key)
	if has {
		return val
	}

	if len(defVal) > 0 {
		return defVal[0]
	}
	return ""
}

// QueryParam return query value by key
func (c *Context) QueryParam(key string) (string, bool) {
	if vs, ok := c.QueryParams(key); ok {
		return vs[0], true
	}
	return "", false
}

// QueryParams return query values by key
func (c *Context) QueryParams(key string) ([]string, bool) {
	if vs, ok := c.Req.URL.Query()[key]; ok && len(vs) > 0 {
		return vs, ok
	}
	return []string{}, false
}

// QueryValues get URL query data
func (c *Context) QueryValues() url.Values { return c.Req.URL.Query() }

// Post return body value by key, and allow with default value
func (c *Context) Post(key string, defVal ...string) string {
	val, has := c.PostParam(key)
	if has {
		return val
	}

	if len(defVal) > 0 {
		return defVal[0]
	}
	return ""
}

// PostParam return body value by key
func (c *Context) PostParam(key string) (string, bool) {
	if vs, ok := c.PostParams(key); ok {
		return vs[0], true
	}
	return "", false
}

// PostParams return body values by key
func (c *Context) PostParams(key string) ([]string, bool) {
	// parse body data
	req := c.Req
	_ = req.ParseForm()
	_ = req.ParseMultipartForm(defaultMaxMemory)

	if vs := req.PostForm[key]; len(vs) > 0 {
		return vs, true
	}
	return []string{}, false
}

// FormParams return body values
func (c *Context) FormParams(excepts ...[]string) (url.Values, error) {
	if strings.HasPrefix(c.Req.Header.Get(httpctype.Key), "multipart/form-data") {
		if err := c.ParseMultipartForm(defaultMaxMemory); err != nil {
			return nil, err
		}
	} else if err := c.Req.ParseForm(); err != nil {
		return nil, err
	}

	if len(excepts) > 0 {
		for _, k := range excepts[0] {
			c.Req.Form.Del(k)
		}
	}

	return c.Req.Form, nil
}

// ParseMultipartForm parse multipart forms.
//
// Tips:
//
//	c.Req.PostForm = POST(PUT,PATCH) body data
//	c.Req.Form = c.Req.PostForm + GET queries data
//	c.Req.MultipartForm = uploaded files data + other body fields data(will append to Req.Form and Req.PostForm)
func (c *Context) ParseMultipartForm(maxMemory ...int) error {
	max := defaultMaxMemory
	if len(maxMemory) > 0 {
		max = maxMemory[0]
	}

	return c.Req.ParseMultipartForm(int64(max))
}

// FormFile returns the first file for the provided form key.
func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	_, fh, err := c.Req.FormFile(name)
	return fh, err
}

// UploadFile handle upload file and save as local file
func (c *Context) UploadFile(name string, saveAs string) error {
	_, fh, err := c.Req.FormFile(name)
	if err != nil {
		return err
	}

	return c.SaveFile(fh, saveAs)
}

// SaveFile uploads the form file to specific dst.
func (c *Context) SaveFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		_ = src.Close()
		return err
	}
	//noinspection GoUnhandledErrorResult
	defer out.Close()

	_, err = io.Copy(out, src)
	_ = src.Close()

	return err
}

// ReqCtxValue get context value from http.Request.ctx
//
// Example:
//
//	// record value to Request.ctx
//	r := c.Req
//	c.Req = r.WithContext(context.WithValue(r.Context(), "key", "value"))
//	// ...
//	val := c.ReqCtxValue("key") // "value"
func (c *Context) ReqCtxValue(key any) any {
	return c.Req.Context().Value(key)
}

// WithReqCtxValue with request ctx Value.
//
// Usage:
//
//	ctx.WithReqCtxValue()
func (c *Context) WithReqCtxValue(key, val any) {
	r := c.Req
	c.Req = r.WithContext(context.WithValue(r.Context(), key, val))
}

// RawBodyData get raw body data
func (c *Context) RawBodyData() ([]byte, error) { return io.ReadAll(c.Req.Body) }

/*************************************************************
 * Context: request extra info
 *************************************************************/

// IsTLS request check
func (c *Context) IsTLS() bool { return c.Req.TLS != nil }

// IsAjax check request is ajax request
func (c *Context) IsAjax() bool {
	return c.Header("X-Requested-With") == "XMLHttpRequest"
}

// IsGet check request is post request
func (c *Context) IsGet() bool { return c.Req.Method == GET }

// IsPost check request is post request
func (c *Context) IsPost() bool { return c.Req.Method == POST }

// IsMethod returns true if current is equal to input method name
func (c *Context) IsMethod(method string) bool { return c.Req.Method == method }

// IsWebSocket returns true if the request headers indicate that a webSocket
// handshake is being initiated by the client.
func (c *Context) IsWebSocket() bool {
	if strings.Contains(strings.ToLower(c.Header("Connection")), "upgrade") &&
		strings.ToLower(c.Header("Upgrade")) == "websocket" {
		return true
	}
	return false
}

// ContentType get content type.
func (c *Context) ContentType() string { return c.Req.Header.Get(ContentType) }

// AcceptedTypes get Accepted Types.
func (c *Context) AcceptedTypes() []string {
	return parseAccept(c.Req.Header.Get("Accept"))
}

// ClientIP implements a best effort algorithm to return the real client IP
func (c *Context) ClientIP() string {
	clientIP := c.Header("X-Forwarded-For")
	if index := strings.IndexByte(clientIP, ','); index >= 0 {
		clientIP = clientIP[0:index]
	}

	clientIP = strings.TrimSpace(clientIP)
	if len(clientIP) > 0 {
		return clientIP
	}

	clientIP = strings.TrimSpace(c.Header("X-Real-Ip"))
	if len(clientIP) > 0 {
		return clientIP
	}

	// if c.AppEngine {
	// 	if addr := c.Req.Header.Get("X-Appengine-Remote-Addr"); addr != "" {
	// 		return addr
	// 	}
	// }

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Req.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

/*************************************************************
 * Context: cookies data
 *************************************************************/

// SetCookie adds a Set-Cookie header to the ResponseWriter's headers.
// The provided cookie must have a valid Name. Invalid cookies may be
// silently dropped.
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	if path == "" {
		path = "/"
	}

	http.SetCookie(c.Resp, &http.Cookie{
		Name:     name,
		Value:    url.QueryEscape(value),
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		Secure:   secure,
		HttpOnly: httpOnly,
	})
}

// FastSetCookie Quick Set Cookie
func (c *Context) FastSetCookie(name, value string, maxAge int) {
	scheme := c.Req.URL.Scheme
	isHttp := scheme == "" || scheme == "http"

	c.SetCookie(name, value, maxAge, "/", c.Req.URL.Host, !isHttp, isHttp)
}

// DelCookie by given names
func (c *Context) DelCookie(names ...string) {
	for _, name := range names {
		c.FastSetCookie(name, "", -1)
	}
}

// Cookie returns the named cookie provided in the request or
// ErrNoCookie if not found. And return the named cookie is unescaped.
// If multiple cookies match the given name, only one cookie will
// be returned.
func (c *Context) Cookie(name string) string {
	cookie, err := c.Req.Cookie(name)
	if err != nil {
		return ""
	}

	val, _ := url.QueryUnescape(cookie.Value)
	return val
}

/*************************************************************
 * Context: response data
 * - more please see context_render.go
 *************************************************************/

// SetStatus code for the response
func (c *Context) SetStatus(status int) { c.writer.WriteHeader(status) }

// SetStatusCode code for the response. alias of the SetStatus()
func (c *Context) SetStatusCode(status int) { c.writer.WriteHeader(status) }

// StatusCode get status code from the response
func (c *Context) StatusCode() int { return c.writer.Status() }

// Length get length from the response
func (c *Context) Length() int { return c.writer.Length() }

// SetHeader for the response
func (c *Context) SetHeader(key, value string) {
	c.Resp.Header().Set(key, value)
}

// WriteBytes write byte data to response, will panic on error.
func (c *Context) WriteBytes(bt []byte) {
	_, err := c.Resp.Write(bt)
	if err != nil {
		panic(err)
	}
}

// WriteString write string to response
func (c *Context) WriteString(str string) { c.WriteBytes([]byte(str)) }

/*************************************************************
 * Context: implement the context.Context
 *************************************************************/

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
func (c *Context) Deadline() (deadline time.Time, ok bool) { return }

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled. Done may return nil if this context can
// never be canceled. Successive calls to Done return the same value.
func (c *Context) Done() <-chan struct{} { return nil }

// Err returns a non-nil error value after Done is closed,
// successive calls to Err return the same error.
// If Done is not yet closed, Err returns nil.
// If Done is closed, Err returns a non-nil error explaining why:
// Canceled if the context was canceled
// or DeadlineExceeded if the context's deadline passed.
func (c *Context) Err() error { return nil }

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
func (c *Context) Value(key any) any {
	if key == 0 || key == nil {
		return c.Req
	}

	if keyAsString, ok := key.(string); ok {
		return c.SafeGet(keyAsString)
	}
	return nil
}
