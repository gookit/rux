package handlers

import (
	"context"
	"net/http"
)

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
				// record old method to context
				r = r.WithContext(context.WithValue(r.Context(), "originalMethod", "POST"))
			}
		}

		h.ServeHTTP(w, r)
	})
}
