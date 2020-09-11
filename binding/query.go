package binding

import (
	"net/http"
	"net/url"
)

// TagName for decode url.Values(form,query) data
var QueryTagName = "query"

// QueryBinder binding URL query data to struct
type QueryBinder struct {
	TagName string
}

// Name get name
func (QueryBinder) Name() string {
	return "query"
}

// Bind Query data binder
func (b QueryBinder) Bind(r *http.Request, ptr interface{}) error {
	return DecodeUrlValues(r.URL.Query(), ptr, b.TagName)
}

// BindValues data from url.Values
func (b QueryBinder) BindValues(values url.Values, ptr interface{}) error {
	return DecodeUrlValues(values, ptr, b.TagName)
}
