package binding

import (
	"net/http"
	"strings"
)

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
func Auto(r *http.Request, obj interface{}) (err error) {
	method := r.Method

	// no body. like GET DELETE OPTION ....
	if method != "POST" && method != "PUT" && method != "PATCH" {
		// TODO query data binding
		return
	}

	cType := r.Header.Get("Content-Type")

	// contains file uploaded form: multipart/form-data
	// strings.HasPrefix(mediaType, "multipart/")
	if strings.Contains(cType, "/form-data") {
		// TODO form data binding
	}

	// basic POST form. content type: application/x-www-form-urlencoded
	if strings.Contains(cType, "/x-www-form-urlencoded") {
		if err = r.ParseForm(); err != nil {
			return err
		}

		// TODO form binding
	}

	// JSON body request: application/json
	if strings.Contains(cType, "/json") {
		return JSON.Bind(r, obj)
	}

	// XML body request: text/xml
	if strings.Contains(cType, "/xml") {
		return XML.Bind(r, obj)
	}
	return
}
