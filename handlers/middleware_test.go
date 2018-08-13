package handlers

import (
	"github.com/gookit/sux"
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
	r := sux.New()
	art := assert.New(t)

	// add reqID to context
	r.GET("/rid", func(c *sux.Context) {
		rid := c.Get("reqID").(string)
		art.Len(rid, 32)
	}).Use(GenRequestID())

	w := mockRequest(r, "GET", "/rid", nil)
	art.Equal(200, w.Code)

	// ignore /favicon.ico request
	r.GET("/favicon.ico", func(c *sux.Context) {}, SkipFavIcon())
	w = mockRequest(r, "GET", "/favicon.ico", nil)
	art.Equal(204, w.Code)

	// catch panic
	r.GET("/panic", func(c *sux.Context) {
		panic("error msg")
	}, PanicsHandler())
	w = mockRequest(r, "GET", "/panic", nil)
	art.Equal(500, w.Code)

}

func TestRequestLogger(t *testing.T) {
	r := sux.New()
	art := assert.New(t)

	// log req
	rewriteStdout()
	r.GET("/req-log", func(c *sux.Context) {
		c.Text(200, "hello")
	}, RequestLogger())

	w := mockRequest(r, "GET", "/req-log", nil)
	art.Equal(200, w.Code)
	art.Equal("hello", w.Body.String())

	out := restoreStdout()
	art.Contains(out, "/req-log")

	// skip log
	rewriteStdout()
	r.GET("/status", func(c *sux.Context) {
		c.WriteString("hello")
	}, RequestLogger())

	w = mockRequest(r, "GET", "/status", nil)
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

// usage:
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

// usage:
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
