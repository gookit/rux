package render

import (
	"net/http"
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

func writeContentType(w http.ResponseWriter, value string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{value}
	}
}
