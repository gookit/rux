package binding

import (
	"encoding/json"
	"net/http"
)

// QueryBinder URL query data binder
type QueryBinder struct {

}

// Name get name
func (QueryBinder) Name() string {
	return "url-query"
}

// Bind Query data binder
func (QueryBinder) Bind(ptr interface{}, r *http.Request) error {
	return json.NewDecoder(r.Body).Decode(ptr)
}
