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

	w := mockRequest(r, "GET", "/routes", "")
	art.Contains(w.buf.String(), "Routes Count: 1")
}

func TestHTTPMethodOverrideHandler(t *testing.T) {
	art := assert.New(t)

	r := sux.New()
	h := HTTPMethodOverrideHandler(r)

	r.PUT("/put", func(c *sux.Context) {
		// real method save on the request.Context
		art.Equal("POST", c.Req.Context().Value("originalMethod"))
		c.Text(200, "put")
	})

	// send POST as PUT
	w := requestWithData(h, "POST", "/put", &mockData{
		Heads: m{"X-HTTP-Method-Override": "PUT"},
	})
	art.Equal(200, w.status)
	art.Equal("put", w.buf.String())

	w = requestWithData(h, "POST", "/put", &mockData{
		Heads: m{"Content-Type": "application/x-www-form-urlencoded"},
		Body:  "_method=put",
	})
	art.Equal(200, w.status)
	art.Equal("put", w.buf.String())
}
