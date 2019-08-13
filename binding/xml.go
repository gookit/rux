package binding

import (
	"encoding/xml"
	"net/http"
)

// JSONBinder JSON data binder
var XMLBinder = BinderFunc(func(i interface{}, r *http.Request) error {
	return xml.NewDecoder(r.Body).Decode(i)
})

// XML parse request XML data to an ptr
func XML(i interface{}, r *http.Request) error {
	return XMLBinder.Bind(i, r)
}
