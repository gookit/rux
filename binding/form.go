package binding

import (
	"net/http"
	"net/url"

	"github.com/monoculum/formam"
)

// FormTagName for decode form data
var FormTagName = "form"

// FormBinder binding Form/url.Values data to struct
type FormBinder struct {
	TagName string
}

// Name get name
func (FormBinder) Name() string {
	return "form"
}

// Bind Form data from http.Request
func (b FormBinder) Bind(r *http.Request, ptr interface{}) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	return DecodeUrlValues(r.Form, ptr, b.TagName)
}

// BindValues data from url.Values
func (b FormBinder) BindValues(values url.Values, ptr interface{}) error {
	return DecodeUrlValues(values, ptr, b.TagName)
}

// DecodeUrlValues data to struct
func DecodeUrlValues(values map[string][]string, ptr interface{}, tagName string) error {
	dec := formam.NewDecoder(&formam.DecoderOptions{
		TagName: tagName,
	})

	if err := dec.Decode(values, ptr); err != nil {
		return err
	}

	return validating(ptr)
}
