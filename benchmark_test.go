package rux

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
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
	router.GET("/post/{id}", func(c *Context) {})
	router.GET("/view/{id}", func(c *Context) {})
	router.GET("/favicon.ico", func(c *Context) {})
	router.GET("/robots.txt", func(c *Context) {})
	router.GET("/delete/{id}", func(c *Context) {})
	router.GET("/user/{id}/{mode}", func(c *Context) {})

	router.NotFound(func(c *Context) {})
	runRequest(B, router, "GET", "/viewfake")
}

var (
	srsHasMethod = map[string]*Route{}
	srsNoMethod  = map[string]*Route{}
)

func BenchmarkStableRoutes_hasMethod(B *testing.B) {
	srsHasMethod["GET/"] = NewRoute("/", emptyHandler, http.MethodGet)
	srsHasMethod["GET/home"] = NewRoute("/home", emptyHandler, http.MethodGet)

	B.ReportAllocs()
	B.ResetTimer()

	path := "/"
	method := http.MethodGet
	for i := 0; i < B.N; i++ {
		key := method + path
		if _, ok := srsHasMethod[key]; ok {
			// match ok
		}
	}
}

func BenchmarkStableRoutes_noMethod(B *testing.B) {
	srsNoMethod["/"] = NewRoute("/", emptyHandler, http.MethodGet)
	srsNoMethod["/home"] = NewRoute("/home", emptyHandler, http.MethodGet)

	B.ReportAllocs()
	B.ResetTimer()

	path := "/"
	method := http.MethodGet
	for i := 0; i < B.N; i++ {
		route, ok := srsNoMethod[path]
		if ok && strings.Contains(route.MethodString("|")+"|", method+"|") {
			// match ok
		}
	}
}

func TestMultiMatchAtOnce(t *testing.T) {
	t.Skip("skip testing this")

	// route: /user/{arg1}/{arg2}
	regexS := `^(?|/user/([^/]+)/([^/]+)|/blog/([^/]+)/([^/]+)|/order/([^/]+)/([^/]+)|/goods/([^/]+)/([^/]+))$`

	// tests := []struct{}
	rgp := regexp.MustCompilePOSIX(regexS)
	ret := rgp.FindAllStringSubmatch("/user/test/123", -1)
	fmt.Println(ret)
}

/*************************************************************
 * helper methods(ref the gin framework)
 *************************************************************/

type (
	m  map[string]string
	md struct {
		// body
		B string
		// headers
		H m
	}
)

// mock an HTTP Request
// Usage:
// 	handler := router.New()
// 	res := mockRequest(handler, "GET", "/path", nil)
// 	// with data
// 	res := mockRequest(handler, "GET", "/path", &md{B: "data", H: m{"x-head": "val"}})
func mockRequest(h http.Handler, method, path string, data *md, beforeSend ...func(req *http.Request)) *httptest.ResponseRecorder {
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

	if len(beforeSend) > 0 {
		beforeSend[0](req)
	}

	w := httptest.NewRecorder()
	// s := httptest.NewServer()
	h.ServeHTTP(w, req)

	// return w.Result() will return http.Response
	return w
}

// will store old env value, set new val. will restore old value on end.
func mockEnvValue(key, val string, fn func()) {
	old := os.Getenv(key)
	_ = os.Setenv(key, val)

	fn()

	if old != "" {
		_ = os.Setenv(key, old)
	}
}

func runRequest(B *testing.B, r *Router, method, path string) {
	// create fake request
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	B.ReportAllocs()
	B.ResetTimer()

	for i := 0; i < B.N; i++ {
		r.ServeHTTP(w, req)
	}
}
