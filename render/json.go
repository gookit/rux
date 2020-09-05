package render

import (
	"encoding/json"
	"net/http"
)

// JSONRenderer for response JSON content to client
type JSONRenderer struct {
	Data interface{}
	Indent string
	// NotEscape HTML string
	NotEscape bool
}

// Render JSON to client
func (r JSONRenderer) Render(w http.ResponseWriter) error {
	writeContentType(w, JSONContentType)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(!r.NotEscape)

	if r.Indent != "" {
		enc.SetIndent("", r.Indent)
	}

	return enc.Encode(r.Data)
}

// JSON response rendering
func JSON(obj interface{}, w http.ResponseWriter) error {
	return JSONRenderer{Data: obj}.Render(w)
}

// JSONPretty response rendering with indent
func JSONPretty(obj interface{}, w http.ResponseWriter) error {
	return JSONRenderer{Data: obj, Indent: PrettyIndent}.Render(w)
}

// JSONPRenderer for response JSONP content to client
type JSONPRenderer struct {
	Data interface{}
	Callback string
}

// Render JSONP to client
func (r JSONPRenderer) Render(w http.ResponseWriter) (err error) {
	writeContentType(w, JSContentType)

	if _, err = w.Write([]byte(r.Callback + "(")); err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	if err = enc.Encode(r.Data); err != nil {
		return err
	}

	_, err = w.Write([]byte(");"))
	return err
}

// JSONP response rendering
func JSONP(callback string, obj interface{}, w http.ResponseWriter) error {
	r := JSONPRenderer{
		Data:     obj,
		Callback: callback,
	}

	return r.Render(w)
}
