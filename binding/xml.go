package binding

import (
	"encoding/xml"
	"net/http"
)

// JSONBinder JSON data binder
var XMLBinder = BinderFunc(func(ptr interface{}, r *http.Request) error {
	return xml.NewDecoder(r.Body).Decode(ptr)
})

// XML parse request XML data to an ptr
func XML(ptr interface{}, r *http.Request) error {
	return XMLBinder.Bind(ptr, r)
}
