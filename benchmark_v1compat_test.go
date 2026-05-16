package rux

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// v1-comparable benchmarks. Function names + bodies mirror v1.x-final's
// benchmark_test.go so benchstat can directly diff v1 vs v2 numbers.
//
// Only the ServeHTTP-based benchmarks are mirrored — v1's Hybrid*/Legacy*/
// StableRoutes_* benchmarks depend on removed APIs (QuickMatch, EnableHybridMode,
// HybridCacheSize, NewRoute as a package-level constructor).

var compatEmptyHandler = func(c *Context) {}

func runCompatRequest(B *testing.B, r *Router, method, path string) {
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

func BenchmarkOneRoute(B *testing.B) {
	router := New()
	router.GET("/ping", compatEmptyHandler)
	runCompatRequest(B, router, "GET", "/ping")
}

func BenchmarkManyHandlers(B *testing.B) {
	router := New()
	router.Use(func(c *Context) {})
	router.Use(func(c *Context) {})
	router.GET("/ping", compatEmptyHandler)
	runCompatRequest(B, router, "GET", "/ping")
}

func Benchmark5Params(B *testing.B) {
	router := New()
	router.Use(func(c *Context) {})
	router.GET("/param/{param1}/{params2}/{param3}/{param4}/{param5}", compatEmptyHandler)
	runCompatRequest(B, router, "GET", "/param/path/to/parameter/john/12345")
}

func BenchmarkManyRoutesFist(B *testing.B) {
	router := New()
	router.Any("/ping", compatEmptyHandler)
	runCompatRequest(B, router, "GET", "/ping")
}

func Benchmark404(B *testing.B) {
	router := New()
	router.Any("/something", compatEmptyHandler)
	router.NotFound(func(c *Context) {})
	runCompatRequest(B, router, "GET", "/ping")
}

func Benchmark404Many(B *testing.B) {
	router := New()
	router.GET("/", compatEmptyHandler)
	router.GET("/path/to/something", compatEmptyHandler)
	router.GET("/post/{id}", compatEmptyHandler)
	router.GET("/view/{id}", compatEmptyHandler)
	router.GET("/favicon.ico", compatEmptyHandler)
	router.GET("/robots.txt", compatEmptyHandler)
	router.GET("/delete/{id}", compatEmptyHandler)
	router.GET("/user/{id}/{mode}", compatEmptyHandler)

	router.NotFound(func(c *Context) {})
	runCompatRequest(B, router, "GET", "/viewfake")
}

// Silence unused-import warning if fmt drops out as helpers evolve.
var _ = fmt.Sprintf
