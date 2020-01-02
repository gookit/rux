package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gookit/rux"
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

type SkiperAllowURLConfig struct {
	Skipper Skipper
}

func (au *SkiperAllowURLConfig) Check() rux.HandlerFunc {
	return func(c *rux.Context) {
		if au.Skipper != nil {
			if !au.Skipper(c) {
				c.AbortWithStatus(403, "url error")
			}
		}

		c.Next()
	}
}

func TestSkiperHandler(t *testing.T) {
	art := assert.New(t)
	r := rux.New()

	var allowURL = &SkiperAllowURLConfig{}

	allowURL.Skipper = func(c *rux.Context) bool {
		if c.URL().Path == "/test4" {
			return true
		}

		return false
	}

	r.GET("/test1", DumpRoutesHandler(), allowURL.Check())
	r.GET("/test2", DumpRoutesHandler(), allowURL.Check())
	r.GET("/test3", DumpRoutesHandler(), allowURL.Check())
	r.GET("/test4", DumpRoutesHandler(), allowURL.Check())

	w1 := mockRequest(r, "GET", "/test1", nil)
	w2 := mockRequest(r, "GET", "/test2", nil)
	w3 := mockRequest(r, "GET", "/test3", nil)
	w4 := mockRequest(r, "GET", "/test4", nil)

	art.Equal(403, w1.Code)
	art.Equal(403, w2.Code)
	art.Equal(403, w3.Code)
	art.Equal(200, w4.Code)
}
