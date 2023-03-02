package rux

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gookit/goutil/netutil/httpctype"
	"github.com/gookit/rux/render"
)

// ShouldRender render and response to client
func (c *Context) ShouldRender(status int, obj any, renderer render.Renderer) error {
	c.SetStatus(status)
	return renderer.Render(c.Resp, obj)
}

// MustRender render and response to client
func (c *Context) MustRender(status int, obj any, renderer render.Renderer) {
	c.Respond(status, obj, renderer)
}

// Respond render and response to client
func (c *Context) Respond(status int, obj any, renderer render.Renderer) {
	c.SetStatus(status)

	err := renderer.Render(c.Resp, obj)
	if err != nil {
		// panic(err) // TODO or use AddError()
		c.AddError(err)
	}
}

/*************************************************************
 * data response render
 *************************************************************/

// HTTPError response
func (c *Context) HTTPError(msg string, status int) {
	http.Error(c.Resp, msg, status)
}

// NoContent serve success but no content response
func (c *Context) NoContent() {
	c.Resp.WriteHeader(http.StatusNoContent)
}

// Redirect other URL with status code(3xx e.g 301, 302).
func (c *Context) Redirect(path string, optionalCode ...int) {
	// default is 301
	code := http.StatusMovedPermanently
	if len(optionalCode) > 0 {
		code = optionalCode[0]
	}

	http.Redirect(c.Resp, c.Req, path, code)
}

// Back Redirect back url
func (c *Context) Back(optionalCode ...int) {
	// default is 302
	code := http.StatusFound
	if len(optionalCode) > 0 {
		code = optionalCode[0]
	}

	c.Redirect(c.Req.Referer(), code)
}

// Text writes out a string as plain text.
func (c *Context) Text(status int, str string) {
	c.Blob(status, httpctype.Text, []byte(str))
}

// HTML writes out as html text. if data is empty, only write headers
func (c *Context) HTML(status int, data []byte) {
	c.Blob(status, httpctype.HTML, data)
}

// Blob writes out []byte
func (c *Context) Blob(status int, contentType string, data []byte) {
	c.Resp.WriteHeader(status)
	c.Resp.Header().Set(ContentType, contentType)

	if len(data) > 0 {
		c.WriteBytes(data)
	}
}

// Stream writes out io.Reader
func (c *Context) Stream(status int, contentType string, r io.Reader) {
	c.Resp.WriteHeader(status)
	c.Resp.Header().Set(ContentType, contentType)
	_, err := io.Copy(c.Resp, r)

	if err != nil {
		// TODO use AddError()
		// panic(err)
		c.AddError(err)
	}
}

// JSON writes out a JSON response.
func (c *Context) JSON(status int, obj any) {
	c.Respond(status, obj, render.JSONRenderer{})
}

// JSONBytes writes out a string as JSON response.
func (c *Context) JSONBytes(status int, bs []byte) {
	c.Blob(status, httpctype.JSON, bs)
}

// XML output xml response.
func (c *Context) XML(status int, obj any, indents ...string) {
	var indent string
	if len(indents) > 0 && indents[0] != "" {
		indent = indents[0]
	}

	c.Respond(status, obj, render.XMLRenderer{Indent: indent})
}

// JSONP is JSONP response.
func (c *Context) JSONP(status int, callback string, obj any) {
	c.Respond(status, obj, render.JSONPRenderer{Callback: callback})
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
	//noinspection GoUnhandledErrorResult
	defer f.Close()

	c.setRawContentHeader(c.Resp, false)
	http.ServeContent(c.Resp, c.Req, name, time.Now(), f)
}

// Attachment a file to response.
// Usage:
//
//	c.Attachment("path/to/some.zip", "new-name.zip")
func (c *Context) Attachment(srcFile, outName string) {
	c.dispositionContent(c.Resp, http.StatusOK, outName, false)
	c.FileContent(srcFile)
}

// Inline file content.
// Usage:
//
//	c.Inline("testdata/site.md", "new-name.md")
func (c *Context) Inline(srcFile, outName string) {
	c.dispositionContent(c.Resp, http.StatusOK, outName, true)
	c.FileContent(srcFile)
}

// Binary serve data as Binary response.
// Usage:
//
//	in, _ := os.Open("./README.md")
//	r.Binary(http.StatusOK, in, "readme.md", true)
func (c *Context) Binary(status int, in io.ReadSeeker, outName string, inline bool) {
	c.dispositionContent(c.Resp, status, outName, inline)

	// _, err := io.Copy(c.Resp, in)
	http.ServeContent(c.Resp, c.Req, outName, time.Now(), in)
}

// Stream read
// func (c *Context) Stream(step func(w io.Writer) bool) {
// 	w := c.Resp
// 	clientGone := w.(http.CloseNotifier).CloseNotify()
// 	for {
// 		select {
// 		case <-clientGone:
// 			return
// 		default:
// 			keepOpen := step(w)
// 			w.(http.Flusher).Flush()
// 			if !keepOpen {
// 				return
// 			}
// 		}
// 	}
// }

func (c *Context) dispositionContent(w http.ResponseWriter, status int, outName string, inline bool) {
	dispositionType := dispositionAttachment
	if inline {
		dispositionType = dispositionInline
	}

	// "application/octet-stream"
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
