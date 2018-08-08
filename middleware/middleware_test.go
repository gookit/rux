package middleware

import (
	"bytes"
	"github.com/gookit/sux"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
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

	w := mockRequest(r, "GET", "/rid", "")
	art.Equal(200, w.Status())

	// ignore /favicon.ico request
	r.GET("/favicon.ico", func(c *sux.Context) {}, SkipFavIcon())
	w = mockRequest(r, "GET", "/favicon.ico", "")
	art.Equal(204, w.Status())

	// catch panic
	r.GET("/panic", func(c *sux.Context) {
		panic("error msg")
	}, PanicsHandler())
	w = mockRequest(r, "GET", "/panic", "")
	art.Equal(500, w.Status())

}

func TestRequestLogger(t *testing.T) {
	r := sux.New()
	art := assert.New(t)

	// log req
	rewriteStdout()
	r.GET("/req-log", func(c *sux.Context) {
		c.Text(200, "hello")
	}, RequestLogger())

	w := mockRequest(r, "GET", "/req-log", "")
	art.Equal(200, w.Status())
	art.Equal("hello", w.buf.String())

	out := restoreStdout()
	art.Contains(out, "/req-log")

	// skip log
	rewriteStdout()
	r.GET("/status", func(c *sux.Context) {
		c.WriteString("hello")
	}, RequestLogger())

	w = mockRequest(r, "GET", "/status", "")
	art.Equal(200, w.Status())
	art.Equal("hello", w.buf.String())

	out = restoreStdout()
	art.Equal(out, "")
}

/*************************************************************
 * helper methods(ref the gin framework)
 *************************************************************/

type mockWriter struct {
	buf     *bytes.Buffer
	status  int
	headers http.Header
}

func newMockWriter() *mockWriter {
	return &mockWriter{
		&bytes.Buffer{},
		200,
		http.Header{},
	}
}

func (m *mockWriter) Status() int {
	return m.status
}

func (m *mockWriter) Header() (h http.Header) {
	return m.headers
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	return m.buf.Write(p)
}

func (m *mockWriter) WriteString(s string) (n int, err error) {
	return m.buf.Write([]byte(s))
}

func (m *mockWriter) WriteHeader(code int) {
	m.status = code
}

func mockRequest(r *sux.Router, method, path, bodyStr string) *mockWriter {
	var body io.Reader
	if bodyStr != "" {
		body = strings.NewReader(bodyStr)
	}

	// create fake request
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		panic(err)
	}
	req.RequestURI = req.URL.String()

	w := newMockWriter()
	r.ServeHTTP(w, req)
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
