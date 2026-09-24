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

		return decodeMultipart(r.PostForm, multipartFiles(r), ptr, b.TagName, r)
	}

	err := r.ParseForm()
	if err != nil {
		return err
	}

	return decodeUrlValues(r.Form, ptr, b.TagName, r)
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
	return decodeUrlValues(values, ptr, tagName, nil)
}

// decodeUrlValues decodes url.Values and validates the result, handing r to a
// RequestValidator when one is installed.
func decodeUrlValues(values map[string][]string, ptr any, tagName string, r *http.Request) error {
	if err := DecodeValues(values, ptr, tagName); err != nil {
		return err
	}

	return ValidateRequest(r, ptr)
}

// DecodeMultipart binds the form values and the uploaded files of a
// multipart/form-data request into ptr.
//
// Values and files are decoded first and the validator runs once afterwards, so
// rules that look at an upload (required, image, mime) see the bound file.
// Binders that hold the request (Auto, Form.Bind) also hand it to a
// RequestValidator; this entry point has no request and passes nil.
func DecodeMultipart(
	values map[string][]string,
	files map[string][]*multipart.FileHeader,
	ptr any,
	tagName string,
) error {
	return decodeMultipart(values, files, ptr, tagName, nil)
}

// decodeMultipart is DecodeMultipart with the request that produced the data.
func decodeMultipart(
	values map[string][]string,
	files map[string][]*multipart.FileHeader,
	ptr any,
	tagName string,
	r *http.Request,
) error {
	if err := DecodeValues(values, ptr, tagName); err != nil {
		return err
	}

	if err := DecodeFiles(files, ptr, tagName); err != nil {
		return err
	}

	return ValidateRequest(r, ptr)
}
