package binding

import (
	"net/http"
)

// HeaderBinder binding URL query data to struct
type HeaderBinder struct{}

// Name get name
func (HeaderBinder) Name() string {
	return "header"
}

// Bind Header data binding
func (HeaderBinder) Bind(r *http.Request, ptr interface{}) error {
	return DecodeUrlValues(r.Header, ptr)
}

// BindValues data from headers
func (HeaderBinder) BindValues(headers map[string][]string, ptr interface{}) error {
	return DecodeUrlValues(headers, ptr)
}
