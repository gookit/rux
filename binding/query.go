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
func (QueryBinder) Bind(r *http.Request, obj interface{}) error {
	return json.NewDecoder(r.Body).Decode(obj)
}
