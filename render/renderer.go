// Package render is a simple renderer for render Text,JSON,HTML,... response
//
// ref the package: https://github.com/thedevsaddam/renderer
package render

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"html/template"
	"io"
	"net/http"
	"github.com/gookit/view"
)

const (
	defaultCharset            = "UTF-8"
	defaultXMLPrefix          = `<?xml version="1.0" encoding="ISO-8859-1" ?>\n`
	defaultJSONPrefix         = ""
	defaultTemplateExt        = "tpl"
	defaultTemplateLeftDelim  = "{{"
	defaultTemplateRightDelim = "}}"
)

// M describes handy type that represents data to send as response
type M map[string]interface{}

// Options for the renderer
type Options struct {
	Debug bool

	JSONIndent bool
	JSONPrefix string

	XMLIndent bool
	XMLPrefix string

	// template render
	TplLayout   string
	TplDelims   view.TplDelims
	TplSuffixes []string
	TplFuncMap  template.FuncMap
}

// Renderer definition
type Renderer struct {
	opts       Options
	HTMLRender *template.Template
}

func New(config ...func(*Options)) *Renderer {
	r := &Renderer{
		opts: Options{
			TplDelims:   view.TplDelims{"{{", "}}"},
			TplSuffixes: []string{"tpl"},
		},
	}

	// apply user config
	if len(config) > 0 {
		config[0](&r.opts)
	}

	return r
}

// LoadTemplateGlob
// usage:
// 		LoadTemplateGlob("views/*")
// 		LoadTemplateGlob("views/**/*")
func (r *HTTPRenderer) LoadTemplateGlob(pattern string) {
	r.HTMLRender = template.Must(template.New("").
		Delims(r.opts.TplDelims.Left, r.opts.TplDelims.Right).
		Funcs(r.opts.TplFuncMap).
		ParseGlob(pattern),
	)
}

// LoadTemplateFiles
// usage:
// 		LoadTemplateFiles("path/file1.tpl", "path/file2.tpl")
func (r *HTTPRenderer) LoadTemplateFiles(files ...string) {
	r.HTMLRender = template.Must(template.New("").
		Delims(r.opts.TplDelims.Left, r.opts.TplDelims.Right).
		Funcs(r.opts.TplFuncMap).
		ParseFiles(files...),
	)
}

func (r *Renderer) Auto(w io.Writer, accepted string, data interface{}) error {
	return nil
}

// func (r *Renderer) Render(w io.Writer, d Driver, data interface{}) error {
// 	err := d.Render(w, data)

	// if hw, ok := w.(http.ResponseWriter); err != nil && !r.opts.DisableHTTPErrorRendering && ok {
	// 	http.Error(hw, err.Error(), http.StatusInternalServerError)
	// }
//
// 	return err
// }

// Text write text to Writer
func (r *Renderer) Text(w io.Writer, status int, v string) error {
	_, err := w.Write([]byte(v))
	return err
}

// JSON serve string content as json response
func (r *Renderer) JSON(w io.Writer, v interface{}) error {
	bs, err := jsonMarshal(v, r.opts.JSONIndent, false)
	if err != nil {
		return err
	}

	if r.opts.JSONPrefix != "" {
		w.Write([]byte(r.opts.JSONPrefix))
	}

	_, err = w.Write(bs)
	return err
}

func (r *Renderer) HTML(w io.Writer, template string, v interface{}) error {
	if template == "" {
		return errors.New("renderer: template name not exist")
	}

	vr := view.NewRenderer()
	return vr.Render(w, template, v)
}

// XML serve data as XML response
func (r *Renderer) XML(w io.Writer, v interface{}) error {
	var bs []byte
	var err error

	if r.opts.XMLIndent {
		bs, err = xml.MarshalIndent(v, "", " ")
	} else {
		bs, err = xml.Marshal(v)
	}
	if err != nil {
		return err
	}

	if r.opts.XMLPrefix != "" {
		w.Write([]byte(r.opts.XMLPrefix))
	}

	_, err = w.Write(bs)
	return err
}

// Data is the generic function called by XML, JSON, Data, HTML, and can be called by custom implementations.
func Data(w http.ResponseWriter, status int, v interface{}) error {
	w.WriteHeader(status)
	_, err := w.Write(v.([]byte))

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
