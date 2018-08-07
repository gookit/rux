package sux

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func BenchmarkOneRoute(B *testing.B) {
	router := New()
	router.GET("/ping", func(c *Context) {})
	runRequest(B, router, "GET", "/ping")
}

func BenchmarkManyHandlers(B *testing.B) {
	router := New()
	// router.Use(Recovery(), LoggerWithWriter(newMockWriter()))
	router.Use(func(c *Context) {})
	router.Use(func(c *Context) {})
	router.GET("/ping", func(c *Context) {})
	runRequest(B, router, "GET", "/ping")
}

func Benchmark5Params(B *testing.B) {
	// DefaultWriter = os.Stdout
	router := New()
	router.Use(func(c *Context) {})
	router.GET("/param/{param1}/{params2}/{param3}/{param4}/{param5}", func(c *Context) {})
	runRequest(B, router, "GET", "/param/path/to/parameter/john/12345")
}

func BenchmarkManyRoutesFist(B *testing.B) {
	router := New()
	router.Any("/ping", func(c *Context) {})
	runRequest(B, router, "GET", "/ping")
}

func Benchmark404(B *testing.B) {
	router := New()
	router.Any("/something", func(c *Context) {})
	router.NotFound(func(c *Context) {})
	runRequest(B, router, "GET", "/ping")
}

func Benchmark404Many(B *testing.B) {
	router := New()
	router.GET("/", func(c *Context) {})
	router.GET("/path/to/something", func(c *Context) {})
	router.GET("/post/:id", func(c *Context) {})
	router.GET("/view/:id", func(c *Context) {})
	router.GET("/favicon.ico", func(c *Context) {})
	router.GET("/robots.txt", func(c *Context) {})
	router.GET("/delete/:id", func(c *Context) {})
	router.GET("/user/:id/:mode", func(c *Context) {})

	router.NotFound(func(c *Context) {})
	runRequest(B, router, "GET", "/viewfake")
}

/*************************************************************
 * helper methods(ref the gin framework)
 *************************************************************/

type mockWriter struct {
	headers http.Header
	buf     *bytes.Buffer
}

func newMockWriter() *mockWriter {
	return &mockWriter{
		http.Header{},
		&bytes.Buffer{},
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
	m.headers.Set("Status", fmt.Sprintf("%d", code))
}

func runRequest(B *testing.B, r *Router, method, path string) {
	// create fake request
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}

	w := newMockWriter()
	B.ReportAllocs()
	B.ResetTimer()

	for i := 0; i < B.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func mockRequest(r *Router, method, path, bodyStr string) {
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
}
