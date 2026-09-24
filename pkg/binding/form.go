package binding

import (
	"mime/multipart"
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
//
// A multipart/form-data request binds its uploaded files as well, see
// DecodeMultipart.
func (b FormBinder) Bind(r *http.Request, ptr any) error {
	if isMultipartForm(r) {
		if err := r.ParseMultipartForm(DefaultMaxMemory); err != nil {
			return err
		}

		return DecodeMultipart(r.PostForm, multipartFiles(r), ptr, b.TagName)
	}

	err := r.ParseForm()
	if err != nil {
		return err
	}

	return DecodeUrlValues(r.Form, ptr, b.TagName)
}

// BindValues data from url.Values
func (b FormBinder) BindValues(values url.Values, ptr any) error {
	return DecodeUrlValues(values, ptr, b.TagName)
}

// DecodeValues data to struct, without running the validator.
func DecodeValues(values map[string][]string, ptr any, tagName string) error {
	dec := formam.NewDecoder(&formam.DecoderOptions{
		TagName: tagName,
	})

	return dec.Decode(values, ptr)
}

// DecodeUrlValues data to struct
func DecodeUrlValues(values map[string][]string, ptr any, tagName string) error {
	if err := DecodeValues(values, ptr, tagName); err != nil {
		return err
	}

	return Validate(ptr)
}

// DecodeMultipart binds the form values and the uploaded files of a
// multipart/form-data request into ptr.
//
// Values and files are decoded first and the validator runs once afterwards, so
// rules that look at an upload (required, image, mime) see the bound file.
func DecodeMultipart(
	values map[string][]string,
	files map[string][]*multipart.FileHeader,
	ptr any,
	tagName string,
) error {
	if err := DecodeValues(values, ptr, tagName); err != nil {
		return err
	}

	if err := DecodeFiles(files, ptr, tagName); err != nil {
		return err
	}

	return Validate(ptr)
}
