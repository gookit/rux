package binding

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// JSONBinder binding JSON data to struct
type JSONBinder struct{}

// Name get name
func (JSONBinder) Name() string {
	return "json"
}

// Bind JSON data from http.Request
func (JSONBinder) Bind(r *http.Request, ptr any) error {
	return decodeJSON(r.Body, ptr)
}

// BindBytes raw JSON data to struct
func (JSONBinder) BindBytes(bts []byte, ptr any) error {
	return decodeJSON(strings.NewReader(string(bts)), ptr)
}

func decodeJSON(r io.Reader, ptr any) error {
	err := json.NewDecoder(r).Decode(ptr)
	if err != nil {
		return err
	}

	return validating(ptr)
}
