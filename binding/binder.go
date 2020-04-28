package binding

import (
	"net/http"
	"strings"
)

// Binder interface
type Binder interface {
	Bind(ptr interface{}, r *http.Request) error
}

// BinderFunc bind func
type BinderFunc func(interface{}, *http.Request) error

// BinderFunc implements the Binder interface
func (fn BinderFunc) Bind(ptr interface{}, r *http.Request) error {
	return fn(ptr, r)
}

var binders = map[string]Binder{
	"xml":  BinderFunc(XML),
	"json": JSONBinder,
	// "form": , TODO
}

// Add register new binder with name
func Add(name string, b Binder) {
	if name != "" && b != nil {
		binders[name] = b
	}
}

// Remove a exist binder
func Remove(name string) {
	if _, ok := binders[name]; ok {
		delete(binders, name)
	}
}

// Auto auto bind request data to an ptr
//
//		body, err := ioutil.ReadAll(c.Request().Body)
//		if err != nil {
//			c.Logger().Errorf("could not read request body: %v", err)
//		}
//		c.Set("request_body", body)
//		// fix: can not read request body multiple times
//		c.Request().Body = ioutil.NopCloser(bytes.NewReader(body))
//
func Auto(ptr interface{}, r *http.Request) (err error) {
	method := r.Method

	// no body. like GET DELETE OPTION ....
	if method != "POST" && method != "PUT" && method != "PATCH" {
		return
	}

	cType := r.Header.Get("Content-Type")

	// contains file uploaded form: multipart/form-data
	// strings.HasPrefix(mediaType, "multipart/")
	if strings.Contains(cType, "/form-data") {

	}

	// basic POST form. content type: application/x-www-form-urlencoded
	if strings.Contains(cType, "/x-www-form-urlencoded") {
		if err = r.ParseForm(); err != nil {
			return err
		}

		// TODO
	}

	// JSON body request: application/json
	if strings.Contains(cType, "/json") {
		return JSON(ptr, r)
	}

	// XML body request: text/xml
	if strings.Contains(cType, "/xml") {
		return XML(ptr, r)
	}
	return
}

// Validator interface
type Validator interface {
	Validate(i interface{}) error
	Rules() map[string]string
}
