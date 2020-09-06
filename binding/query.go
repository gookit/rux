package binding

import (
	"net/http"
)

// QueryBinder binding URL query data to struct
type QueryBinder struct {}

// Name get name
func (QueryBinder) Name() string {
	return "url-query"
}

// Bind Query data binder
func (QueryBinder) Bind(r *http.Request, ptr interface{}) error {
	return decodeForm(r.URL.Query(), ptr)
}
