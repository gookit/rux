package render

import (
	"encoding/xml"
	"net/http"

	"github.com/gookit/goutil/netutil/httpctype"
)

// XMLRenderer for response XML content to client
type XMLRenderer struct {
	// Data interface{}
	Indent string
}

// Render XML to client
func (r XMLRenderer) Render(w http.ResponseWriter, obj interface{}) error {
	writeContentType(w, httpctype.XML)

	enc := xml.NewEncoder(w)
	if r.Indent != "" {
		enc.Indent("", r.Indent)
	}

	var err error
	if _, err = w.Write([]byte(xml.Header)); err != nil {
		return err
	}

	return enc.Encode(obj)
}

// XML response rendering
func XML(w http.ResponseWriter, obj interface{}) error {
	return XMLRenderer{}.Render(w, obj)
}

// XMLPretty response rendering with indent
func XMLPretty(w http.ResponseWriter, obj interface{}) error {
	return XMLRenderer{Indent: PrettyIndent}.Render(w, obj)
}
