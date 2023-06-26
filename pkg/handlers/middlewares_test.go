package handlers

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/testutil"
	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/rux"
)

func TestSomeMiddleware(t *testing.T) {
	r := rux.New()
	art := assert.New(t)

	// add reqID to context
	r.GET("/rid", func(c *rux.Context) {
		rid, ok := c.Get("req_id")
		art.True(ok)
		art.NotEmpty(rid.(string))
		dump.P(rid)
	}).Use(GenRequestID("req_id"))

	w := mockRequest(r, "GET", "/rid", nil)
	art.Eq(200, w.Code)

	// ignore /favicon.ico request
	r.GET("/favicon.ico", func(c *rux.Context) {}, IgnoreFavIcon())
	w = mockRequest(r, "GET", "/favicon.ico", nil)
	art.Eq(204, w.Code)

	// catch panic
	r.GET("/panic", func(c *rux.Context) {
		panic("error msg")
	}, PanicsHandler())
	w = mockRequest(r, "GET", "/panic", nil)
	art.Eq(500, w.Code)
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
	is.Eq(401, w.Code)

	req, _ = http.NewRequest("GET", "/auth", nil)
	req.SetBasicAuth("test", "123err")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	is.Eq(403, w.Code)

	req, _ = http.NewRequest("GET", "/auth", nil)
	req.SetBasicAuth("test", "123")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	is.Eq(200, w.Code)
	is.Eq("hello", w.Body.String())
}

func TestRequestLogger(t *testing.T) {
	r := rux.New()
	ris := assert.New(t)

	m2code := map[string]int{
		"GET":   200,
		"PUT":   301,
		"PATCH": 401,
		"HEAD":  501,
	}

	// log req
	// rewriteStdout()
	r.Any("/req-log", func(c *rux.Context) {
		code, err := strconv.Atoi(c.Query("code", "200"))
		c.Text(code, "hello")
		ris.NoErr(err)
	}, RequestLogger())

	for _, m := range rux.AnyMethods() {
		code := m2code[m]
		if code == 0 {
			code = 200
		}

		uri := fmt.Sprintf("/req-log?code=%d", code)
		w := mockRequest(r, m, uri, nil)
		ris.Eq(code, w.Code)
		ris.Eq("hello", w.Body.String())
	}

	// out := restoreStdout()
	// ris.Contains(out, "/req-log")

	// skip log
	rewriteStdout()
	r.GET("/status", func(c *rux.Context) {
		c.WriteString("hello")
	}, RequestLogger())

	w := testutil.MockRequest(r, "GET", "/status", nil)
	ris.Eq(200, w.Code)
	ris.Eq("hello", w.Body.String())

	// out = restoreStdout()
	// ris.Eq(out, "")
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
//
//	handler := router.New()
//	res := mockRequest(handler, "GET", "/path", nil)
//	// with data
//	res := mockRequest(handler, "GET", "/path", &md{B: "data", H:{"x-head": "val"}})
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
	_ = os.Stdout.Close()
	// restore
	os.Stdout = oldStdout
	oldStdout = nil

	// read data
	out, _ := ioutil.ReadAll(newReader)

	// close reader
	_ = newReader.Close()
	newReader = nil

	return string(out)
}
