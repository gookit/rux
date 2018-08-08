package middleware

import (
	"bytes"
	"github.com/gookit/sux"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMiddleware(t *testing.T) {
	art := assert.New(t)

	r := sux.New()
	r.GET("/rid", func(c *sux.Context) {
		rid := c.Get("reqID").(string)
		art.Len(rid, 32)
	}).Use(GenRequestID())

	w := mockRequest(r, "GET", "/rid", "")
	art.Equal(200, w.Status())

	r.GET("/favicon.ico", func(c *sux.Context) {}, SkipFavIcon())
	w = mockRequest(r, "GET", "/favicon.ico", "")
	art.Equal(204, w.Status())

	r.GET("/panic", func(c *sux.Context) {
		panic("error msg")
	}, PanicsHandler())
	w = mockRequest(r, "GET", "/panic", "")
	art.Equal(500, w.Status())

	r.GET("/req-log", func(c *sux.Context) {
		c.Text(200, "hello")
	}, RequestLogger())
	w = mockRequest(r, "GET", "/req-log", "")
	art.Equal(200, w.Status())
	art.Equal("hello", w.buf.String())
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
