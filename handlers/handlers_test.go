package handlers

import (
	"net/http"
	"testing"

	"github.com/gookit/rux"
	"github.com/stretchr/testify/assert"
)

func ExampleHTTPMethodOverrideHandler() {
	r := rux.New()

	h := HTTPMethodOverrideHandler(r)
	http.ListenAndServe(":8080", h)

	// can also:
	h1 := r.WrapHTTPHandlers(HTTPMethodOverrideHandler)
	http.ListenAndServe(":8080", h1)
}

func TestDumpRoutesHandler(t *testing.T) {
	art := assert.New(t)
	r := rux.New()

	r.GET("/routes", DumpRoutesHandler())

	w := mockRequest(r, "GET", "/routes", nil)
	art.Contains(w.Body.String(), "Routes Count: 1")
}

func TestHTTPMethodOverrideHandler(t *testing.T) {
	art := assert.New(t)

	r := rux.New()
	h := HTTPMethodOverrideHandler(r)

	r.PUT("/put", func(c *rux.Context) {
		// real method save on the request.Context
		art.Equal("POST", c.ReqCtxValue(OriginalMethodContextKey))
		c.Text(200, "put")
	})

	// send POST as PUT
	w := mockRequest(h, "POST", "/put", &md{
		H: m{"X-HTTP-Method-Override": "PUT"},
	})
	art.Equal(200, w.Code)
	art.Equal("put", w.Body.String())

	w = mockRequest(h, "POST", "/put", &md{
		H: m{"Content-Type": "application/x-www-form-urlencoded"},
		B: "_method=put",
	})
	art.Equal(200, w.Code)
	art.Equal("put", w.Body.String())
}
