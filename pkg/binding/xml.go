package binding

import (
	"encoding/xml"
	"io"
	"net/http"
	"strings"
)

// XMLBinder Xml data binder
// struct binding-tag default is "xml". eg: `xml:"field"`
type XMLBinder struct{}

// Name get name
func (XMLBinder) Name() string {
	return "xml"
}

// Bind XML data binder
func (XMLBinder) Bind(r *http.Request, obj any) error {
	return decodeXML(r.Body, obj, r)
}

// BindBytes raw JSON data to struct
func (XMLBinder) BindBytes(bts []byte, ptr any) error {
	return decodeXML(strings.NewReader(string(bts)), ptr, nil)
}

func decodeXML(rd io.Reader, obj any, r *http.Request) error {
	err := xml.NewDecoder(rd).Decode(obj)
	if err != nil {
		return err
	}

	return ValidateRequest(r, obj)
}
