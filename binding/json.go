package binding

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// JSONBinder JSON data binder
type JSONBinder struct {}

// Name get name
func (JSONBinder) Name() string {
	return "json"
}

// Bind JSON data from http.Request
func (JSONBinder) Bind(ptr interface{}, r *http.Request) error {
	return json.NewDecoder(r.Body).Decode(ptr)
}

// Bind JSON data from raw data
func (JSONBinder) BindRaw(ptr interface{}, bts []byte) error {
	return decodeJSON(strings.NewReader(string(bts)), ptr)
}

func decodeJSON(r io.Reader, ptr interface{}) error  {
	err := json.NewDecoder(r).Decode(ptr)
	if err != nil {
		return err
	}

	return validating(ptr)
}
