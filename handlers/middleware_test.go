package handlers

import (
	"github.com/gookit/rux"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSomeMiddleware(t *testing.T) {
	r := rux.New()
	art := assert.New(t)

	// add reqID to context
	r.GET("/rid", func(c *rux.Context) {
		rid, ok := c.Get("reqID")
		art.True(ok)
		art.Len(rid.(string), 32)
	}).Use(GenRequestID())

	w := mockRequest(r, "GET", "/rid", nil)
	art.Equal(200, w.Code)

	// ignore /favicon.ico request
	r.GET("/favicon.ico", func(c *rux.Context) {}, IgnoreFavIcon())
	w = mockRequest(r, "GET", "/favicon.ico", nil)
	art.Equal(204, w.Code)

	// catch panic
	r.GET("/panic", func(c *rux.Context) {
		panic("error msg")
	}, PanicsHandler())
	w = mockRequest(r, "GET", "/panic", nil)
	art.Equal(500, w.Code)
}

func TestHTTPBasicAuth(t *testing.T) {
	r := rux.New()
	is := assert.New(t)

	// basic auth
	r.GET("/auth", func(c *rux.Context) {
		c.WriteString("hello")
	}, HTTPBasicAuth(map[string]string{"test": "123"}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth", nil)
	r.ServeHTTP(w, req)
	is.Equal(401, w.Code)

	req, _ = http.NewRequest("GET", "/auth", nil)
	req.SetBasicAuth("test", "123err")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	is.Equal(403, w.Code)

	req, _ = http.NewRequest("GET", "/auth", nil)
	req.SetBasicAuth("test", "123")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	is.Equal(200, w.Code)
	is.Equal("hello", w.Body.String())
}

func TestRequestLogger(t *testing.T) {
	r := rux.New()
	art := assert.New(t)

	// log req
	rewriteStdout()
	r.Any("/req-log", func(c *rux.Context) {
		c.Text(200, "hello")
	}, RequestLogger())

	for _, m := range rux.AnyMethods() {
		w := mockRequest(r, m, "/req-log", nil)
		art.Equal(200, w.Code)
		art.Equal("hello", w.Body.String())
	}

	out := restoreStdout()
	art.Contains(out, "/req-log")

	// skip log
	rewriteStdout()
	r.GET("/status", func(c *rux.Context) {
		c.WriteString("hello")
	}, RequestLogger())

	w := mockRequest(r, "GET", "/status", nil)
	art.Equal(200, w.Code)
	art.Equal("hello", w.Body.String())

	out = restoreStdout()
	art.Equal(out, "")
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

var oldStdout, newReader *os.File

// Usage:
// rewriteStdout()
// fmt.Println("Hello, playground")
// msg := restoreStdout()
func rewriteStdout() {
	oldStdout = os.Stdout
	r, w, _ := os.Pipe()
	newReader = r
	os.Stdout = w
}

func restoreStdout() string {
	if newReader == nil {
		return ""
	}

	// Notice: must close writer before read data
	// close now reader
	os.Stdout.Close()
	// restore
	os.Stdout = oldStdout
	oldStdout = nil

	// read data
	out, _ := ioutil.ReadAll(newReader)

	// close reader
	newReader.Close()
	newReader = nil

	return string(out)
}
