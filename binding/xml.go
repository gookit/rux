package binding

import (
	"encoding/xml"
	"io"
	"net/http"
	"strings"
)

// XMLBinder Xml data binder
type XMLBinder struct{}

// Name get name
func (XMLBinder) Name() string {
	return "xml"
}

// Bind XML data binder
func (XMLBinder) Bind(r *http.Request, obj interface{}) error {
	return decodeXML(r.Body, obj)
}

// BindBytes raw JSON data to struct
func (XMLBinder) BindBytes(bts []byte, ptr interface{}) error {
	return decodeXML(strings.NewReader(string(bts)), ptr)
}

func decodeXML(r io.Reader, obj interface{}) error {
	err := xml.NewDecoder(r).Decode(obj)
	if err != nil {
		return err
	}

	return validating(obj)
}
