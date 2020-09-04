package render

import (
	"io"
	"net/http"
)

var (
	TextContentType  = "text/plain; charset=UTF-8"
	HTMLContentType  = "text/html; charset=UTF-8"
	XMLContentType   = "application/xml; charset=UTF-8"
	JSONContentType  = "application/json; charset=utf-8"

	JSContentType = "application/javascript; charset=UTF-8"
)

// Renderer interface
type Renderer interface {
	Render(w io.Writer, obj interface{}) error
}

func writeContentType(w http.ResponseWriter, value string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{value}
	}
}
