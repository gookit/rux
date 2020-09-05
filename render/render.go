package render

import (
	"net/http"
)

// PrettyIndent indent string for  render JSON or XML
var PrettyIndent = "  "

// Renderer interface
type Renderer interface {
	Render(w http.ResponseWriter) error
}

func writeContentType(w http.ResponseWriter, value string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{value}
	}
}
