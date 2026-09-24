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
		return decodeUrlValues(r.URL.Query(), obj, Query.TagName, r)
	}

	// binding body data by content type.
	cType := r.Header.Get("Content-Type")
	_, subType := mediaTypes(cType)

	switch {
	// basic POST form data binding. content type: "application/x-www-form-urlencoded"
	case isSubType(subType, "x-www-form-urlencoded"):
		if err = r.ParseForm(); err != nil {
			return err
		}

		return decodeUrlValues(r.PostForm, obj, Form.TagName, r)

	// contains file uploaded form: "multipart/form-data"
	case isSubType(subType, "form-data"):
		err = r.ParseMultipartForm(DefaultMaxMemory)
		if err != nil {
			return err
		}

		// bind the form values and the uploaded files
		return decodeMultipart(r.PostForm, multipartFiles(r), obj, Form.TagName, r)

	// JSON body request: "application/json", a "+json" structured suffix such
	// as "application/vnd.api+json", or a dash variant like "json-patch"
	case isSubType(subType, "json"):
		return JSON.Bind(r, obj)

	// XML body request: "text/xml" / "application/xml", or a "+xml" suffix such
	// as "application/atom+xml"
	case isSubType(subType, "xml"):
		return XML.Bind(r, obj)
	}

	return errors.New("cannot auto binding request data, content-type: " + cType)
}

// isMultipartForm reports whether r carries a multipart/form-data body, the
// only multipart type net/http parses into a form.
func isMultipartForm(r *http.Request) bool {
	_, subType := mediaTypes(r.Header.Get("Content-Type"))
	return isSubType(subType, "form-data")
}

// mediaTypes splits a Content-Type header into its lowercased media type and
// subtype, with any parameters removed: "multipart/form-data; boundary=x" gives
// ("multipart/form-data", "form-data").
//
// A header that cannot be parsed falls back to a plain split, so a malformed
// value like "multipart/form-data; boundary" still reaches the multipart
// branch, which reports the real problem.
func mediaTypes(cType string) (mediaType, subType string) {
	mediaType, _, err := mime.ParseMediaType(cType)
	if err != nil {
		mediaType = strings.ToLower(cType)
		if p := strings.IndexByte(mediaType, ';'); p != -1 {
			mediaType = mediaType[:p]
		}
	}

	if p := strings.IndexByte(mediaType, '/'); p != -1 {
		return mediaType, mediaType[p+1:]
	}
	return mediaType, ""
}

// isSubType reports whether a media subtype names want: the token itself
// ("json"), a structured suffix ("vnd.api+json") or a dash variant
// ("json-patch", accepted long before this check was explicit).
func isSubType(subType, want string) bool {
	return strings.HasPrefix(subType, want) || strings.HasSuffix(subType, "+"+want)
}
