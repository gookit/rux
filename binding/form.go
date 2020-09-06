package binding

import (
	"net/http"
	"net/url"

	"github.com/monoculum/formam"
)

var TagName = "form"

// FormBinder binding Form/url.Values data to struct
type FormBinder struct {}

// Name get name
func (FormBinder) Name() string {
	return "form"
}

// Bind Form data from http.Request
func (FormBinder) Bind(r *http.Request, ptr interface{}) error {
	_ = r.ParseForm()
	return decodeForm(r.Form, ptr)
}

// Bind Form data from raw data
func (FormBinder) BindValues( val url.Values, ptr interface{}) error {
	return decodeForm(val, ptr)
}

func decodeForm(form url.Values, ptr interface{}) error  {
	dec := formam.NewDecoder(&formam.DecoderOptions{
		TagName: TagName,
	})

	err := dec.Decode(form, ptr)
	if err != nil {
		return err
	}

	return validating(ptr)
}

