package binding

import (
	"net/http"
)

// HeaderTagName for binding data
var HeaderTagName = "header"

// HeaderBinder binding URL query data to struct
type HeaderBinder struct {
	TagName string
}

// Name get name
func (HeaderBinder) Name() string {
	return "header"
}

// Bind Header data binding
func (b HeaderBinder) Bind(r *http.Request, ptr any) error {
	return DecodeUrlValues(r.Header, ptr, b.TagName)
}

// BindValues data from headers
func (b HeaderBinder) BindValues(headers map[string][]string, ptr any) error {
	return DecodeUrlValues(headers, ptr, b.TagName)
}
