package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil"
	"github.com/gookit/goutil/testutil/assert"
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

	w := testutil.MockRequest(r, "GET", "/routes", nil)
	art.Contains(w.Body.String(), "Routes Count: 1")
}

func TestHTTPMethodOverrideHandler(t *testing.T) {
	art := assert.New(t)

	r := rux.New()
	h := HTTPMethodOverrideHandler(r)

	r.PUT("/put", func(c *rux.Context) {
		// real method save on the request.Context
		art.Eq("POST", c.ReqCtxValue(OriginalMethodContextKey))
		c.Text(200, "put")
	})

	// send POST as PUT
	w := testutil.MockRequest(h, "POST", "/put", &testutil.MD{
		Headers: testutil.M{"X-HTTP-Method-Override": "PUT"},
	})
	art.Eq(200, w.Code)
	art.Eq("put", w.Body.String())

	w = testutil.MockRequest(h, "POST", "/put", &testutil.MD{
		Headers: testutil.M{"Content-Type": "application/x-www-form-urlencoded"},
		Body:    strings.NewReader("_method=put"),
	})
	art.Eq(200, w.Code)
	art.Eq("put", w.Body.String())
}

type SkipperAllowURLConfig struct {
	Skipper Skipper
}

func (au *SkipperAllowURLConfig) Check() rux.HandlerFunc {
	return func(c *rux.Context) {
		if au.Skipper != nil {
			if !au.Skipper(c) {
				c.AbortWithStatus(403, "url error")
			}
		}

		c.Next()
	}
}

func TestSkipperHandler(t *testing.T) {
	art := assert.New(t)
	r := rux.New()

	var allowURL = &SkipperAllowURLConfig{}

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

	w1 := testutil.MockRequest(r, "GET", "/test1", nil)
	w2 := testutil.MockRequest(r, "GET", "/test2", nil)
	w3 := testutil.MockRequest(r, "GET", "/test3", nil)
	w4 := testutil.MockRequest(r, "GET", "/test4", nil)

	art.Eq(403, w1.Code)
	art.Eq(403, w2.Code)
	art.Eq(403, w3.Code)
	art.Eq(200, w4.Code)
}
