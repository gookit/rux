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
	return decodeJSON(r.Body, ptr, r)
}

// BindBytes raw JSON data to struct
func (JSONBinder) BindBytes(bts []byte, ptr any) error {
	return decodeJSON(strings.NewReader(string(bts)), ptr, nil)
}

func decodeJSON(rd io.Reader, ptr any, r *http.Request) error {
	err := json.NewDecoder(rd).Decode(ptr)
	if err != nil {
		return err
	}

	return ValidateRequest(r, ptr)
}
