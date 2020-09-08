package render

import (
	"net/http"

	"github.com/gookit/goutil/netutil/httpctype"
)

// PrettyIndent indent string for  render JSON or XML
var PrettyIndent = "  "

// Renderer interface
type Renderer interface {
	Render(w http.ResponseWriter, obj interface{}) error
}

// RendererFunc definition
type RendererFunc func(w http.ResponseWriter, obj interface{}) error

// Render to http.ResponseWriter
func (fn RendererFunc) Render(w http.ResponseWriter, obj interface{}) error {
	return fn(w, obj)
}

// Text writes out a string as plain text.
func Text(w http.ResponseWriter, str string) error {
	return Blob(w, httpctype.Text, []byte(str))
}

// Plain writes out a string as plain text. alias of the Text()
func Plain(w http.ResponseWriter, str string) error {
	return Blob(w, httpctype.Text, []byte(str))
}

// TextBytes writes out a string as plain text.
func TextBytes(w http.ResponseWriter, data []byte) error {
	return Blob(w, httpctype.Text, data)
}

// HTML writes out as html text. if data is empty, only write headers
func HTML(w http.ResponseWriter, data string) error {
	return Blob(w, httpctype.HTML, []byte(data))
}

// HTMLBytes writes out as html text. if data is empty, only write headers
func HTMLBytes(w http.ResponseWriter, data []byte) error {
	return Blob(w, httpctype.HTML, data)
}

// Blob writes out []byte
func Blob(w http.ResponseWriter, contentType string, data []byte) (err error) {
	writeContentType(w, contentType)

	if len(data) > 0 {
		_, err = w.Write(data)
	}
	return
}

func writeContentType(w http.ResponseWriter, value string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{value}
	}
}
