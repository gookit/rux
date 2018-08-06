// Package render is a simple renderer for render Text,JSON,HTML,... response
//
// ref the package: https://github.com/thedevsaddam/renderer
package render

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
)

const (
	// ContentType header key
	ContentType = "Content-Type"
	// ContentText represents content type text/plain
	ContentText = "text/plain"
	// ContentJSON represents content type application/json
	ContentJSON = "application/json"
	// ContentJSONP represents content type application/javascript
	ContentJSONP = "application/javascript"
	// ContentXML represents content type application/xml
	ContentXML = "application/xml"
	// ContentYAML represents content type application/x-yaml
	ContentYAML = "application/x-yaml"
	// ContentHTML represents content type text/html
	ContentHTML = "text/html"
	// ContentBinary represents content type application/octet-stream
	ContentBinary = "application/octet-stream"
)

const (
	defaultCharset            = "UTF-8"
	defaultXMLPrefix          = `<?xml version="1.0" encoding="ISO-8859-1" ?>\n`
	defaultJSONPrefix         = ""
	defaultTemplateExt        = "tpl"
	defaultTemplateLeftDelim  = "{{"
	defaultTemplateRightDelim = "}}"
)

type Handler interface {
	Render(w http.ResponseWriter, status int, data interface{}) error
}

// M describes handy type that represents data to send as response
type M map[string]interface{}
type TplDelims struct {
	Left  string
	Right string
}

// Options for the renderer
type Options struct {
	// supported content types
	ContentBinary, ContentHTML, ContentXML, ContentText, ContentJSON, ContentJSONP string

	Charset       string
	AppendCharset bool

	// template render
	TplDelims   TplDelims
	TplSuffixes []string
	TplFuncMap  []template.FuncMap
}

type HtmlTpl struct {
}

var tplEngine *template.Template
var opts = &Options{
	ContentXML:    ContentXML,
	ContentText:   ContentText,
	ContentHTML:   ContentHTML,
	ContentJSON:   ContentJSON,
	ContentJSONP:  ContentJSONP,
	ContentBinary: ContentBinary,

	// Charset content data charset
	Charset: defaultCharset,
	// AppendCharset on response content
	AppendCharset: true,

	TplDelims:   TplDelims{"{{", "}}"},
	TplSuffixes: []string{"tpl"},
}

// Config the render package
func Config(fn func(*Options)) {
	fn(opts)
}

// Init
func Init() {
	if opts.AppendCharset {

	}
}

func AppendCharset() {

}

// Empty serve success but no content response
func Empty(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// NoContent alias method of the Empty()
func NoContent(w http.ResponseWriter) error {
	return Empty(w)
}

// Text serve string content as text/plain response
func Text(w http.ResponseWriter, status int, v string) error {
	w.WriteHeader(status)
	w.Header().Set(ContentType, "text/plain; charset=UTF-8")
	_, err := w.Write([]byte(v))

	return err
}

// String alias method of the Text()
func String(w http.ResponseWriter, status int, v string) error {
	return Text(w, status, v)
}

// Data is the generic function called by XML, JSON, Data, HTML, and can be called by custom implementations.
func Data(w http.ResponseWriter, status int, v interface{}) error {
	w.WriteHeader(status)
	_, err := w.Write(v.([]byte))

	return err
}

func JSON(w http.ResponseWriter, status int, v interface{}) error {
	w.Header().Set(ContentType, opts.ContentJSON)
	w.WriteHeader(status)

	bs, err := jsonMarshal(v, false, false)
	if err != nil {
		return err
	}

	// if opts.JSONPrefix != "" {
	// 	w.Write([]byte(r.opts.JSONPrefix))
	// }

	_, err = w.Write(bs)
	return err
}

// JSONP serve data as JSONP response
func JSONP(w http.ResponseWriter, status int, callback string, v interface{}) error {
	w.Header().Set(ContentType, opts.ContentJSONP)
	w.WriteHeader(status)

	bs, err := jsonMarshal(v, false, false)
	if err != nil {
		return err
	}

	if callback == "" {
		return errors.New("renderer: callback can not bet empty")
	}

	w.Write([]byte(callback + "("))
	_, err = w.Write(bs)
	w.Write([]byte(");"))

	return err
}

// json converts the data as bytes using json encoder
func jsonMarshal(v interface{}, indent, unEscapeHTML bool) ([]byte, error) {
	var bs []byte
	var err error
	if indent {
		bs, err = json.MarshalIndent(v, "", "  ")
	} else {
		bs, err = json.Marshal(v)
	}

	if err != nil {
		return bs, err
	}

	if unEscapeHTML {
		bs = bytes.Replace(bs, []byte("\\u003c"), []byte("<"), -1)
		bs = bytes.Replace(bs, []byte("\\u003e"), []byte(">"), -1)
		bs = bytes.Replace(bs, []byte("\\u0026"), []byte("&"), -1)
	}

	return bs, nil
}
