// Package binding provide some common binder for binding http.Request data to strcut
package binding

import (
	"errors"
	"net/http"
	"strings"
)

// DefaultMaxMemory for parse data form
var DefaultMaxMemory int64 = 32 << 20 // 32 MB

// MustBind auto bind request data to an struct ptr
func MustBind(r *http.Request, obj interface{}) {
	err := Auto(r, obj)
	if err != nil {
		panic(err)
	}
}

// Bind auto bind request data to an struct ptr
func Bind(r *http.Request, obj interface{}) error {
	return Auto(r, obj)
}

// Auto bind request data to an struct ptr
//
//		body, err := ioutil.ReadAll(c.Request().Body)
//		if err != nil {
//			c.Logger().Errorf("could not read request body: %v", err)
//		}
//		c.Set("request_body", body)
//		// fix: can not read request body multiple times
//		c.Request().Body = ioutil.NopCloser(bytes.NewReader(body))
//
func Auto(r *http.Request, obj interface{}) (err error) {
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

	// contains file uploaded form: "multipart/form-data" "multipart/mixed"
	// strings.HasPrefix(mediaType, "multipart/")
	if strings.Contains(cType, "/form-data") {
		err = r.ParseMultipartForm(DefaultMaxMemory)
		if err != nil {
			return err
		}

		return Form.BindValues(r.PostForm, obj)
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
