// Package binding provide some common binder for binding http.Request data to strcut
//
// Auto picks a binder by content type: query data for a body-less method, form
// values, JSON or XML. A multipart/form-data request binds its form values and
// its uploaded files, so a struct field of type *multipart.FileHeader (or a
// slice of them) is filled by the same call. Use Form, Query, Header, JSON, XML
// or File to bind from one source only.
package binding

import (
	"errors"
	"mime"
	"net/http"
	"strings"
)

// DefaultMaxMemory for parse data form
var DefaultMaxMemory int64 = 32 << 20 // 32 MB

// MustBind auto bind request data to an struct ptr
func MustBind(r *http.Request, obj any) {
	err := Auto(r, obj)
	if err != nil {
		panic(err)
	}
}

// Bind auto bind request data to an struct ptr
func Bind(r *http.Request, obj any) error {
	return Auto(r, obj)
}

// Auto bind request data to a ptr value
//
//	body, err := ioutil.ReadAll(c.Request().Body)
//	if err != nil {
//		c.Logger().Errorf("could not read request body: %v", err)
//	}
//	c.Set("request_body", body)
//	// fix: can not read request body multiple times
//	c.Request().Body = ioutil.NopCloser(bytes.NewReader(body))
func Auto(r *http.Request, obj any) (err error) {
	method := r.Method

	// no body, query data binding. like GET DELETE OPTION ....
	if method != "POST" && method != "PUT" && method != "PATCH" {
		return Query.BindValues(r.URL.Query(), obj)
	}

	// binding body data by content type.
	cType := r.Header.Get("Content-Type")

	// basic POST form data binding. content type: "application/x-www-form-urlencoded"
	if strings.Contains(cType, "/x-www-form-urlencoded") {
		if err = r.ParseForm(); err != nil {
			return err
		}

		return Form.BindValues(r.PostForm, obj)
	}

	// contains file uploaded form: "multipart/form-data"
	if isMultipartForm(r) {
		err = r.ParseMultipartForm(DefaultMaxMemory)
		if err != nil {
			return err
		}

		// bind the form values and the uploaded files
		return DecodeMultipart(r.PostForm, multipartFiles(r), obj, Form.TagName)
	}

	// JSON body request: "application/json"
	if strings.Contains(cType, "/json") {
		return JSON.Bind(r, obj)
	}

	// XML body request: "text/xml"
	if strings.Contains(cType, "/xml") {
		return XML.Bind(r, obj)
	}

	return errors.New("cannot auto binding request data, content-type: " + cType)
}

// isMultipartForm reports whether r carries a multipart/form-data body, the
// only multipart type net/http parses into a form.
func isMultipartForm(r *http.Request) bool {
	cType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(cType)
	if err != nil {
		// malformed header: keep accepting the historical loose match
		return strings.Contains(cType, "/form-data")
	}

	return mediaType == "multipart/form-data"
}
