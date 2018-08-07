package handlers

import "net/http"

const (
	// HTTPMethodOverrideHeader is a commonly used
	// http header to override a request method.
	HTTPMethodOverrideHeader = "X-HTTP-Method-Override"
	// HTTPMethodOverrideFormKey is a commonly used
	// HTML form key to override a request method.
	HTTPMethodOverrideFormKey = "_method"
)

// HTTPMethodOverrideHandler wraps and returns a http.Handler which checks for
// the X-HTTP-Method-Override header or the _method form key, and overrides (if
// valid) request.Method with its value.
//
// This is especially useful for HTTP clients that don't support many http verbs.
// It isn't secure to override e.g a GET to a POST, so only POST requests are
// considered.  Likewise, the override method can only be a "write" method: PUT,
// PATCH or DELETE.
//
// Form method takes precedence over header method.
//
// It is from the https://github.com/gorilla/handlers
func HTTPMethodOverrideHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			om := r.FormValue(HTTPMethodOverrideFormKey)
			if om == "" {
				om = r.Header.Get(HTTPMethodOverrideHeader)
			}

			// only allow: PUT, PATCH or DELETE.
			if om == "PUT" || om == "PATCH" || om == "DELETE" {
				r.Method = om
			}
		}

		h.ServeHTTP(w, r)
	})
}
