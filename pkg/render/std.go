package render

import (
	"io"
	"net/http"
)

// std is the package-level default Responder. Initialized eagerly so
// the package-level helpers work without a New() call.
var std = New()

// Default returns the package-level Responder so callers can tweak it
// directly (e.g. render.Default().SetTemplateRenderer(...)).
func Default() *Responder { return std }

// Init applies OptionFns to the default Responder. Safe to call before
// first use; not safe to interleave with concurrent traffic.
func Init(fns ...OptionFn) {
	for _, fn := range fns {
		fn(std.opts)
	}
}

// SetTemplateRenderer on the default Responder.
func SetTemplateRenderer(t TemplateRenderer) { std.SetTemplateRenderer(t) }

// LoadTemplateGlob on the default Responder.
func LoadTemplateGlob(pattern string) error { return std.LoadTemplateGlob(pattern) }

// LoadTemplateFiles on the default Responder.
func LoadTemplateFiles(files ...string) error { return std.LoadTemplateFiles(files...) }

// The *Status helpers below are status-aware proxies on the default
// Responder. The names are intentionally suffixed so they don't collide
// with the existing stateless render.JSON / render.Text / render.XML /
// render.HTML helpers (which take no status code).

// EmptyStatus writes 204 No Content via the default Responder.
func EmptyStatus(w http.ResponseWriter) error { return std.Empty(w) }

// ContentStatus writes raw bytes with the given Content-Type and status.
func ContentStatus(w http.ResponseWriter, status int, body []byte, contentType string) error {
	return std.Content(w, status, body, contentType)
}

// TextStatus writes a text response with the given status.
func TextStatus(w http.ResponseWriter, status int, v string) error {
	return std.Text(w, status, v)
}

// JSONStatus encodes v as JSON via the default Responder.
func JSONStatus(w http.ResponseWriter, status int, v any) error {
	return std.JSON(w, status, v)
}

// JSONPStatus wraps the JSON-encoded v in a JS callback invocation.
func JSONPStatus(w http.ResponseWriter, status int, callback string, v any) error {
	return std.JSONP(w, status, callback, v)
}

// XMLStatus encodes v as XML via the default Responder.
func XMLStatus(w http.ResponseWriter, status int, v any) error {
	return std.XML(w, status, v)
}

// HTMLStatus renders a template via the default Responder's TemplateRenderer.
func HTMLStatus(w http.ResponseWriter, status int, name string, data any, layout ...string) error {
	return std.HTML(w, status, name, data, layout...)
}

// HTMLStringStatus parses and executes an inline template.
func HTMLStringStatus(w http.ResponseWriter, status int, tplContent string, data any) error {
	return std.HTMLString(w, status, tplContent, data)
}

// HTMLTextStatus writes html as the response body without templating.
func HTMLTextStatus(w http.ResponseWriter, status int, html string) error {
	return std.HTMLText(w, status, html)
}

// BinaryStatus streams in as an attachment (or inline).
func BinaryStatus(w http.ResponseWriter, status int, in io.Reader, outName string, inline bool) error {
	return std.Binary(w, status, in, outName, inline)
}

// AutoStatus picks an output format based on Accept (see Responder.Auto).
func AutoStatus(w http.ResponseWriter, req *http.Request, data any, tplName ...string) error {
	return std.Auto(w, req, data, tplName...)
}
