package pprof

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/rux"
	"github.com/stretchr/testify/assert"
)

func TestRouter_PProf(t *testing.T) {
	r := rux.New(UsePProf)
	is := assert.New(t)

	w := mockRequest(r, "GET", "/debug/pprof/", nil)
	is.Equal(200, w.Code)
	w = mockRequest(r, "GET", "/debug/pprof/heap", nil)
	is.Equal(200, w.Code)
	w = mockRequest(r, "GET", "/debug/pprof/goroutine", nil)
	is.Equal(200, w.Code)
	w = mockRequest(r, "GET", "/debug/pprof/block", nil)
	is.Equal(200, w.Code)
	w = mockRequest(r, "GET", "/debug/pprof/threadcreate", nil)
	is.Equal(200, w.Code)
	w = mockRequest(r, "GET", "/debug/pprof/cmdline", nil)
	is.Equal(200, w.Code)
	w = mockRequest(r, "GET", "/debug/pprof/profile", nil)
	is.Equal(200, w.Code)
	w = mockRequest(r, "GET", "/debug/pprof/symbol", nil)
	is.Equal(200, w.Code)
	w = mockRequest(r, "GET", "/debug/pprof/mutex", nil)
	is.Equal(200, w.Code)
	w = mockRequest(r, "GET", "/debug/pprof/trace", nil)
	is.Equal(200, w.Code)
	w = mockRequest(r, "GET", "/debug/pprof/404", nil)
	is.Equal(404, w.Code)
}

/*************************************************************
 * helper methods(ref the gin framework)
 *************************************************************/

type m map[string]string
type md struct {
	B string
	H m
}

// Usage:
// 	handler := router.New()
// 	res := mockRequest(handler, "GET", "/path", nil)
// 	// with data
// 	res := mockRequest(handler, "GET", "/path", &md{B: "data", H:{"x-head": "val"}})
func mockRequest(h http.Handler, method, path string, data *md) *httptest.ResponseRecorder {
	var body io.Reader
	if data != nil && len(data.B) > 0 {
		body = strings.NewReader(data.B)
	}

	// create fake request
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		panic(err)
	}
	req.RequestURI = req.URL.String()
	if data != nil && len(data.H) > 0 {
		// req.Header.Set("Content-Type", "text/plain")
		for k, v := range data.H {
			req.Header.Set(k, v)
		}
	}

	w := httptest.NewRecorder()
	// s := httptest.NewServer()
	h.ServeHTTP(w, req)

	// return w.Result()
	return w
}
