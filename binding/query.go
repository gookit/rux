package binding

import (
	"net/http"
	"net/url"
)

// QueryBinder binding URL query data to struct
type QueryBinder struct{}

// Name get name
func (QueryBinder) Name() string {
	return "url-query"
}

// Bind Query data binder
func (QueryBinder) Bind(r *http.Request, ptr interface{}) error {
	return DecodeUrlValues(r.URL.Query(), ptr)
}

// BindValues data from url.Values
func (QueryBinder) BindValues(values url.Values, ptr interface{}) error {
	return DecodeUrlValues(values, ptr)
}
