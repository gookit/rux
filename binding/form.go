package binding

import (
	"net/http"
	"net/url"

	"github.com/monoculum/formam"
)

// TagName for decode url.Values(form,query) data
var TagName = "form"

// FormBinder binding Form/url.Values data to struct
type FormBinder struct{}

// Name get name
func (FormBinder) Name() string {
	return "form"
}

// Bind Form data from http.Request
func (FormBinder) Bind(r *http.Request, ptr interface{}) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	return DecodeUrlValues(r.Form, ptr)
}

// BindValues data from url.Values
func (FormBinder) BindValues(values url.Values, ptr interface{}) error {
	return DecodeUrlValues(values, ptr)
}

// DecodeUrlValues data to struct
func DecodeUrlValues(values map[string][]string, ptr interface{}) error {
	dec := formam.NewDecoder(&formam.DecoderOptions{
		TagName: TagName,
	})

	if err := dec.Decode(values, ptr); err != nil {
		return err
	}

	return validating(ptr)
}
