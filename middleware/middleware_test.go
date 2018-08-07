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
	art.Equal(w.Status(), 200)

	r.GET("/favicon.ico", func(c *sux.Context) {}, SkipFavIcon())
	w = mockRequest(r, "GET", "/favicon.ico", "")
	art.Equal(w.Status(), 204)

	r.GET("/panic", func(c *sux.Context) {
		panic("error msg")
	}, PanicsHandler())
	w = mockRequest(r, "GET", "/panic", "")
	art.Equal(w.Status(), 500)
}

type mockWriter struct {
	headers http.Header
	buf     *bytes.Buffer
	status  int
}

func (m *mockWriter) Status() int {
	return m.status
}

func newMockWriter() *mockWriter {
	return &mockWriter{
		http.Header{},
		&bytes.Buffer{},
		200,
	}
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

	w := newMockWriter()
	r.ServeHTTP(w, req)

	return w
}
