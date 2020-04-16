package binding

import (
	"encoding/json"
	"net/http"
)

// JSONBinder JSON data binder
var JSONBinder = BinderFunc(func(i interface{}, r *http.Request) error {
	return json.NewDecoder(r.Body).Decode(i)
})

// JSON parse request JSON data to an ptr
func JSON(i interface{}, r *http.Request) error {
	return JSONBinder.Bind(i, r)
}
