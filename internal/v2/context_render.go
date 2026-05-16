package v2

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gookit/goutil/netutil/httpctype"
	"github.com/gookit/goutil/x/basefn"
	"github.com/gookit/rux/pkg/render"
)

// Response header constants used by render helpers.
const (
	// ContentType is the standard HTTP Content-Type header key.
	ContentType = "Content-Type"
	// ContentBinary is the application/octet-stream content type.
	ContentBinary = "application/octet-stream"
	// ContentDisposition is the standard HTTP Content-Disposition header key.
	ContentDisposition = "Content-Disposition"

	dispositionInline     = "inline"
	dispositionAttachment = "attachment"
)

// Render html template via the registered Renderer.
//
// Prefer ShouldRender for callers supplying their own renderer.
func (c *Context) Render(status int, name string, data any) (err error) {
	if c.Renderer == nil {
		return errors.New("rux: renderer not registered")
	}

	var buf = new(bytes.Buffer)
	if err = c.Renderer.Render(buf, name, data, c); err != nil {
		return err
	}

	c.HTML(status, buf.Bytes())
	return
}

// ShouldRender renders obj with the given renderer and writes status.
func (c *Context) ShouldRender(status int, obj any, renderer render.Renderer) error {
	c.SetStatus(status)
	return renderer.Render(c.Resp, obj)
}

// MustRender is Respond's panic-free alias retained for v1 compatibility.
func (c *Context) MustRender(status int, obj any, renderer render.Renderer) {
	c.Respond(status, obj, renderer)
}

// Respond renders obj and records render errors on the Context.
func (c *Context) Respond(status int, obj any, renderer render.Renderer) {
	c.SetStatus(status)
	err := renderer.Render(c.Resp, obj)
	if err != nil {
		c.AddError(err)
	}
}

/*************************************************************
 * data response render
 *************************************************************/

// HTTPError writes a plain-text error response.
func (c *Context) HTTPError(msg string, status int) {
	http.Error(c.Resp, msg, status)
}

// NoContent writes an HTTP 204 response with no body.
func (c *Context) NoContent() {
	c.Resp.WriteHeader(http.StatusNoContent)
}

// Redirect issues an HTTP redirect. Default code is 302 Found.
func (c *Context) Redirect(path string, optionalCode ...int) {
	code := http.StatusFound
	if len(optionalCode) > 0 {
		code = optionalCode[0]
	}
	http.Redirect(c.Resp, c.Req, path, code)
}

// Back redirects to the request's Referer header (302 by default).
func (c *Context) Back(optionalCode ...int) {
	code := basefn.FirstOr(optionalCode, http.StatusFound)
	c.Redirect(c.Req.Referer(), code)
}

// Text writes str as plain text.
func (c *Context) Text(status int, str string) {
	c.Blob(status, httpctype.Text, []byte(str))
}

// HTML writes data as text/html. Empty data still flushes the headers.
func (c *Context) HTML(status int, data []byte) {
	c.Blob(status, httpctype.HTML, data)
}

// HTMLString writes data as text/html.
func (c *Context) HTMLString(status int, data string) {
	c.Blob(status, httpctype.HTML, []byte(data))
}

// Blob writes raw bytes with the given contentType.
func (c *Context) Blob(status int, contentType string, data []byte) {
	c.Resp.WriteHeader(status)
	c.Resp.Header().Set(ContentType, contentType)
	if len(data) > 0 {
		c.WriteBytes(data)
	}
}

// Stream copies an io.Reader to the response.
func (c *Context) Stream(status int, contentType string, r io.Reader) {
	c.Resp.WriteHeader(status)
	c.Resp.Header().Set(ContentType, contentType)
	_, err := io.Copy(c.Resp, r)
	if err != nil {
		c.AddError(err)
	}
}

// JSON writes obj as a JSON response.
func (c *Context) JSON(status int, obj any) {
	c.Respond(status, obj, render.JSONRenderer{})
}

// JSONBytes writes pre-encoded JSON bytes.
func (c *Context) JSONBytes(status int, bs []byte) {
	c.Blob(status, httpctype.JSON, bs)
}

// XML writes obj as an XML response. The first indent arg, if any, is applied.
func (c *Context) XML(status int, obj any, indents ...string) {
	var indent string
	if len(indents) > 0 && indents[0] != "" {
		indent = indents[0]
	}
	c.Respond(status, obj, render.XMLRenderer{Indent: indent})
}

// JSONP writes obj as a JSONP response with the given callback name.
func (c *Context) JSONP(status int, callback string, obj any) {
	c.Respond(status, obj, render.JSONPRenderer{Callback: callback})
}

// File serves a single file via http.ServeFile.
func (c *Context) File(filePath string) {
	http.ServeFile(c.Resp, c.Req, filePath)
}

// FileContent serves the given file as text content.
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
	defer f.Close() //nolint:errcheck

	c.setRawContentHeader(c.Resp, false)
	http.ServeContent(c.Resp, c.Req, name, time.Now(), f)
}

// Attachment serves srcFile as a downloadable attachment named outName.
func (c *Context) Attachment(srcFile, outName string) {
	c.dispositionContent(c.Resp, http.StatusOK, outName, false)
	c.FileContent(srcFile)
}

// Inline serves srcFile inline (rendered in browser) with name outName.
func (c *Context) Inline(srcFile, outName string) {
	c.dispositionContent(c.Resp, http.StatusOK, outName, true)
	c.FileContent(srcFile)
}

// Binary writes the contents of in as a binary attachment (or inline).
func (c *Context) Binary(status int, in io.ReadSeeker, outName string, inline bool) {
	c.dispositionContent(c.Resp, status, outName, inline)
	http.ServeContent(c.Resp, c.Req, outName, time.Now(), in)
}

func (c *Context) dispositionContent(w http.ResponseWriter, status int, outName string, inline bool) {
	dispositionType := dispositionAttachment
	if inline {
		dispositionType = dispositionInline
	}

	w.Header().Set(httpctype.Key, httpctype.Binary)
	w.Header().Set(ContentDisposition, fmt.Sprintf("%s; filename=%s", dispositionType, outName))
	w.WriteHeader(status)
}

func (c *Context) setRawContentHeader(w http.ResponseWriter, addType bool) {
	w.Header().Set("Content-Description", "Raw content")
	if addType {
		w.Header().Set(httpctype.Key, "text/plain")
	}
	w.Header().Set("Expires", "0")
	w.Header().Set("Cache-Control", "must-revalidate")
	w.Header().Set("Pragma", "public")
}
