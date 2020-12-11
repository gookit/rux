package render

import (
	"encoding/json"
	"net/http"

	"github.com/gookit/goutil/netutil/httpctype"
)

// JSONRenderer for response JSON content to client
type JSONRenderer struct {
	// Data interface{}
	// Indent string for encode
	Indent string
	// NotEscape HTML string
	NotEscape bool
}

// Render JSON to client
func (r JSONRenderer) Render(w http.ResponseWriter, obj interface{}) (err error) {
	writeContentType(w, httpctype.JSON)

	enc := json.NewEncoder(w)
	if r.Indent != "" {
		enc.SetIndent("", r.Indent)
	}

	if r.NotEscape {
		enc.SetEscapeHTML(false)
	}

	return enc.Encode(obj)
}

// JSON response rendering
func JSON(w http.ResponseWriter, obj interface{}) error {
	return JSONRenderer{}.Render(w, obj)
}

// JSONIndented response rendering with indent
func JSONIndented(w http.ResponseWriter, obj interface{}) error {
	return JSONRenderer{Indent: PrettyIndent}.Render(w, obj)
}

// JSONPRenderer for response JSONP content to client
type JSONPRenderer struct {
	Callback string
}

// Render JSONP to client
func (r JSONPRenderer) Render(w http.ResponseWriter, obj interface{}) (err error) {
	writeContentType(w, httpctype.JSONP)

	if _, err = w.Write([]byte(r.Callback + "(")); err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	if err = enc.Encode(obj); err != nil {
		return err
	}

	_, err = w.Write([]byte(");"))
	return err
}

// JSONP response rendering
func JSONP(callback string, obj interface{}, w http.ResponseWriter) error {
	return JSONPRenderer{Callback: callback}.Render(w, obj)
}
