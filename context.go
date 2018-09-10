package sux

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

/*************************************************************
 * Context
 *************************************************************/

const (
	defaultMaxMemory = 32 << 20 // 32 MB
	// abortIndex int8 = math.MaxInt8 / 2
	abortIndex int8 = 63
)

const (
	// ContentType header key
	ContentType = "Content-Type"
	// ContentBinary represents content type application/octet-stream
	ContentBinary = "application/octet-stream"

	// ContentDisposition describes contentDisposition
	ContentDisposition = "Content-Disposition"
	// describes content disposition type
	dispositionInline = "inline"
	// describes content disposition type
	dispositionAttachment = "attachment"
)

// M a short name for `map[string]interface{}`
type M map[string]interface{}

// type responseWriter struct {
// 	http.ResponseWriter
// 	size   int
// 	status int
// }

// Context for http server
type Context struct {
	Req  *http.Request
	Resp http.ResponseWriter
	// current route Params, if route has var Params
	Params Params
	Errors []*error

	index int8
	// current router instance
	router *Router
	// context data, you can save some custom data.
	data map[string]interface{}
	// all handlers for current request.
	// call priority: global -> group -> route -> main handler
	// Notice: last always is main handler of the matched route.
	handlers HandlersChain
}

// Init a context
func (c *Context) Init(res http.ResponseWriter, req *http.Request) {
	c.Req = req
	c.Resp = res
	c.data = make(map[string]interface{})
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
	// c.Writer = &c.writermem
	c.index = -1
	c.data = nil
	c.Params = nil
	c.handlers = nil
	c.Errors = c.Errors[0:0]
	// c.Accepted = nil
}

// Copy a new context
func (c *Context) Copy() *Context {
	var ctx = *c
	ctx.handlers = nil
	ctx.index = abortIndex

	return &ctx
}

// Set a value to context by key.
// usage:
// 		c.Set("key", "value")
// 		// ...
// 		val := c.Get("key") // "value"
func (c *Context) Set(key string, val interface{}) {
	c.data[key] = val
}

// Get a value from context
func (c *Context) Get(key string) (v interface{}, ok bool) {
	v, ok = c.data[key]
	return
}

// Get a value from context
func (c *Context) MustGet(key string) interface{} {
	return c.data[key]
}

// Data get all context data
func (c *Context) Data() map[string]interface{} {
	return c.data
}

// Handler returns the main handler.
func (c *Context) Handler() HandlerFunc {
	return c.handlers.Last()
}

// HandlerName get the main handler name
func (c *Context) HandlerName() string {
	return nameOfFunction(c.handlers.Last())
}

// SetHandlers set handlers
func (c *Context) SetHandlers(handlers HandlersChain) {
	c.handlers = handlers
}

// Router get router instance
func (c *Context) Router() *Router {
	return c.router
}

// Error add a error to context
func (c *Context) Error(err error) {
	c.Errors = append(c.Errors, &err)
}

/*************************************************************
 * Context: request data
 *************************************************************/

// Param returns the value of the URL param.
// 		router.GET("/user/{id}", func(c *sux.Context) {
// 			// a GET request to /user/john
// 			id := c.Param("id") // id == "john"
// 		})
func (c *Context) Param(key string) string {
	return c.Params.String(key)
}

// Header return header value by key
func (c *Context) Header(key string) string {
	if values, _ := c.Req.Header[key]; len(values) > 0 {
		return values[0]
	}

	return ""
}

// URL get URL instance from request
func (c *Context) URL() *url.URL {
	return c.Req.URL
}

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
func (c *Context) QueryValues() url.Values {
	return c.Req.URL.Query()
}

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

// BodyParams return body values by key
func (c *Context) PostParams(key string) ([]string, bool) {
	// parse body data
	req := c.Req
	req.ParseForm()
	req.ParseMultipartForm(defaultMaxMemory)

	if vs := req.PostForm[key]; len(vs) > 0 {
		return vs, true
	}

	return []string{}, false
}

// ParseMultipartForm parse multipart forms.
// Tips:
// 	c.Req.PostForm = POST(PUT,PATCH) body data
// 	c.Req.Form = c.Req.PostForm + GET queries data
// 	c.Req.MultipartForm = uploaded files data + other body fields data(will append to Req.Form and Req.PostForm)
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
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	io.Copy(out, src)
	return nil
}

// ReqCtxValue get context value from http.Request.ctx
// example:
// 		// record value to Request.ctx
// 		r := c.Req
// 		c.Req = r.WithContext(context.WithValue(r.Context(), "key", "value"))
// 		// ...
// 		val := c.ReqCtxValue("key") // "value"
func (c *Context) ReqCtxValue(key interface{}) interface{} {
	return c.Req.Context().Value(key)
}

// WithReqCtxValue with request ctx Value.
// usage:
// ctx.WithReqCtxValue()
func (c *Context) WithReqCtxValue(key, val interface{}) {
	r := c.Req
	c.Req = r.WithContext(context.WithValue(r.Context(), key, val))
}

// RawBody return stream data
func (c *Context) RawBody() ([]byte, error) {
	return ioutil.ReadAll(c.Req.Body)
}

// IsTLS request check
func (c *Context) IsTLS() bool {
	return c.Req.TLS != nil
}

// IsAjax check request is ajax request
func (c *Context) IsAjax() bool {
	return c.Header("X-Requested-With") == "XMLHttpRequest"
}

// IsMethod returns true if current is equal to input method name
func (c *Context) IsMethod(method string) bool {
	return c.Req.Method == method
}

// IsWebSocket returns true if the request headers indicate that a webSocket
// handshake is being initiated by the client.
func (c *Context) IsWebSocket() bool {
	if strings.Contains(strings.ToLower(c.Header("Connection")), "upgrade") &&
		strings.ToLower(c.Header("Upgrade")) == "websocket" {
		return true
	}
	return false
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
 * Context: response data
 *************************************************************/

// SetStatus code for the response
func (c *Context) SetStatus(status int) {
	c.Resp.WriteHeader(status)
}

// SetHeader for the response
func (c *Context) SetHeader(key, value string) {
	c.Resp.Header().Set(key, value)
}

// Write byte data to response
func (c *Context) Write(bt []byte) (n int, err error) {
	return c.Resp.Write(bt)
}

// WriteString to response
func (c *Context) WriteString(str string) (n int, err error) {
	return c.Resp.Write([]byte(str))
}

// HTTPError response
func (c *Context) HTTPError(msg string, status int) {
	http.Error(c.Resp, msg, status)
}

// Text writes out a string as plain text.
func (c *Context) Text(status int, str string) (err error) {
	c.Resp.WriteHeader(status)
	c.Resp.Header().Set(ContentType, "text/plain; charset=UTF-8")

	_, err = c.Resp.Write([]byte(str))
	return
}

// HTML writes out as html text. if data is empty, only write headers
func (c *Context) HTML(status int, data []byte) (err error) {
	c.Resp.WriteHeader(status)
	c.Resp.Header().Set(ContentType, "text/html; charset=UTF-8")

	if len(data) > 0 {
		_, err = c.Resp.Write(data)
	}

	return
}

// JSON writes out a JSON response.
func (c *Context) JSON(status int, v interface{}) (err error) {
	bs, err := json.Marshal(v)
	if err != nil {
		return
	}

	return c.JSONBytes(status, bs)
}

// JSONBytes writes out a string as JSON response.
func (c *Context) JSONBytes(status int, bs []byte) (err error) {
	c.Resp.WriteHeader(status)
	c.Resp.Header().Set(ContentType, "application/json; charset=UTF-8")

	_, err = c.Resp.Write(bs)
	return
}

// NoContent serve success but no content response
func (c *Context) NoContent() error {
	c.Resp.WriteHeader(http.StatusNoContent)
	return nil
}

// Redirect other URL with status code(3xx e.g 301, 302).
func (c *Context) Redirect(path string, optionalCode ...int) {
	// default is http.StatusMovedPermanently
	code := 301
	if len(optionalCode) > 0 {
		code = optionalCode[0]
	}

	http.Redirect(c.Resp, c.Req, path, code)
}

// File writes the specified file into the body stream in a efficient way.
func (c *Context) File(filePath string) {
	http.ServeFile(c.Resp, c.Req, filePath)
}

// FileContent serves given file as text content to response.
func (c *Context) FileContent(file string, names ...string) {
	var name string
	if len(names) > 0 {
		name = names[0]
	} else {
		name = path.Base(file)
	}

	f, err := os.Open(file)
	if err != nil {
		http.Error(c.Resp, "Internal Server Error", 500)
		return
	}
	defer f.Close()

	c.setRawContentHeader(false)
	http.ServeContent(c.Resp, c.Req, name, time.Now(), f)
}

// Attachment a file to response.
// Usage:
// 	c.Attachment("path/to/some.zip", "new-name.zip")
func (c *Context) Attachment(srcFile, outName string) {
	c.dispositionContent(http.StatusOK, outName, false)
	c.FileContent(srcFile)
}

// Inline file content.
// Usage:
// 	c.Inline("testdata/site.md", "new-name.md")
func (c *Context) Inline(srcFile, outName string) {
	c.dispositionContent(http.StatusOK, outName, true)
	c.FileContent(srcFile)
}

// Binary serve data as Binary response.
// Usage:
// 	in, _ := os.Open("./README.md")
// 	r.Binary(http.StatusOK, in, "readme.md", true)
func (c *Context) Binary(status int, in io.ReadSeeker, outName string, inline bool) error {
	c.dispositionContent(http.StatusOK, outName, true)

	// _, err := io.Copy(c.Resp, in)
	http.ServeContent(c.Resp, c.Req, outName, time.Now(), in)
	return nil
}

func (c *Context) dispositionContent(status int, outName string, inline bool) {
	dispositionType := dispositionAttachment
	if inline {
		dispositionType = dispositionInline
	}

	c.Resp.Header().Set(ContentType, ContentBinary)
	c.Resp.Header().Set(ContentDisposition, fmt.Sprintf("%s; filename=%s", dispositionType, outName))
	c.Resp.WriteHeader(status)
}

func (c *Context) setRawContentHeader(addType bool) {
	c.Resp.Header().Set("Content-Description", "Raw content")

	if addType {
		c.Resp.Header().Set("Content-Type", "text/plain")
	}

	c.Resp.Header().Set("Expires", "0")
	c.Resp.Header().Set("Cache-Control", "must-revalidate")
	c.Resp.Header().Set("Pragma", "public")
}

/*************************************************************
 * Context: implement the context.Context
 *************************************************************/

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled. Done may return nil if this context can
// never be canceled. Successive calls to Done return the same value.
func (c *Context) Done() <-chan struct{} {
	return nil
}

// Err returns a non-nil error value after Done is closed,
// successive calls to Err return the same error.
// If Done is not yet closed, Err returns nil.
// If Done is closed, Err returns a non-nil error explaining why:
// Canceled if the context was canceled
// or DeadlineExceeded if the context's deadline passed.
func (c *Context) Err() error {
	return nil
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
func (c *Context) Value(key interface{}) interface{} {
	if key == 0 || key == nil {
		return c.Req
	}

	if keyAsString, ok := key.(string); ok {
		return c.MustGet(keyAsString)
	}

	return nil
}
