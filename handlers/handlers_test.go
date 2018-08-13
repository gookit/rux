package handlers

import (
	"github.com/gookit/sux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func ExampleHTTPMethodOverrideHandler() {
	r := sux.New()

	h := HTTPMethodOverrideHandler(r)
	http.ListenAndServe(":8080", h)

	// can also:
	h1 := r.WrapHttpHandlers(HTTPMethodOverrideHandler)
	http.ListenAndServe(":8080", h1)
}

func TestDumpRoutesHandler(t *testing.T) {
	art := assert.New(t)
	r := sux.New()

	r.GET("/routes", DumpRoutesHandler())

	w := mockRequest(r, "GET", "/routes", nil)
	art.Contains(w.Body.String(), "Routes Count: 1")
}

func TestHTTPMethodOverrideHandler(t *testing.T) {
	art := assert.New(t)

	r := sux.New()
	h := HTTPMethodOverrideHandler(r)

	r.PUT("/put", func(c *sux.Context) {
		// real method save on the request.Context
		art.Equal("POST", c.ReqCtxValue("originalMethod"))
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
