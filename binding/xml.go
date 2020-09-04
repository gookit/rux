package binding

import (
	"encoding/xml"
	"io"
	"net/http"
)

// XMLBinder Xml data binder
type XMLBinder struct {}

// Name get name
func (XMLBinder) Name() string {
	return "xml"
}

// Bind XML data binder
func (XMLBinder) Bind(ptr interface{}, r *http.Request) error {
	return decodeXML(r.Body, ptr)
}

func decodeXML(r io.Reader, ptr interface{}) error {
	err := xml.NewDecoder(r).Decode(ptr)
	if err != nil {
		return err
	}

	return validating(ptr)
}
