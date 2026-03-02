package fastrux_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gookit/rux/fastrux"
)

/*************************************************************
 * helpers
 *************************************************************/

var emptyHandler = func(c *fastrux.Context) {}

func newTestRouter() *fastrux.Router {
	r := fastrux.New()
	r.GET("/", emptyHandler)
	r.GET("/user", emptyHandler)
	r.GET("/user/{id}", emptyHandler)
	r.GET("/user/{id}/profile", emptyHandler)
	r.GET("/user/{id}/posts/{postId}", emptyHandler)
	r.POST("/user", emptyHandler)
	r.PUT("/user/{id}", emptyHandler)
	r.DELETE("/user/{id}", emptyHandler)
	r.GET("/blog/{year}/{month}/{day}/{slug}", emptyHandler)
	r.GET("/assets/*file", emptyHandler)
	r.GET("/api/v1/resource", emptyHandler)
	r.GET("/api/v1/resource/{id}", emptyHandler)
	r.GET("/api/v2/resource", emptyHandler)
	r.GET("/api/v2/resource/{id}", emptyHandler)
	return r
}

func runRequest(b *testing.B, r *fastrux.Router, method, path string) {
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

/*************************************************************
 * ServeHTTP benchmarks (full request cycle)
 *************************************************************/

func BenchmarkServeHTTP_Static(b *testing.B) {
	r := newTestRouter()
	runRequest(b, r, "GET", "/user")
}

func BenchmarkServeHTTP_Root(b *testing.B) {
	r := newTestRouter()
	runRequest(b, r, "GET", "/")
}

func BenchmarkServeHTTP_Param1(b *testing.B) {
	r := newTestRouter()
	runRequest(b, r, "GET", "/user/123")
}

func BenchmarkServeHTTP_Param2(b *testing.B) {
	r := newTestRouter()
	runRequest(b, r, "GET", "/user/123/posts/456")
}

func BenchmarkServeHTTP_Param5(b *testing.B) {
	r := newTestRouter()
	runRequest(b, r, "GET", "/blog/2024/01/15/hello-world")
}

func BenchmarkServeHTTP_Wildcard(b *testing.B) {
	r := newTestRouter()
	runRequest(b, r, "GET", "/assets/css/main.css")
}

func BenchmarkServeHTTP_404(b *testing.B) {
	r := newTestRouter()
	r.NotFound(emptyHandler)
	runRequest(b, r, "GET", "/nonexistent/path")
}

func BenchmarkServeHTTP_ManyHandlers(b *testing.B) {
	r := fastrux.New()
	r.Use(emptyHandler)
	r.Use(emptyHandler)
	r.GET("/ping", emptyHandler)
	runRequest(b, r, "GET", "/ping")
}

/*************************************************************
 * Match/QuickMatch benchmarks (pure route matching, no HTTP overhead)
 *************************************************************/

func BenchmarkMatch_Static(b *testing.B) {
	r := newTestRouter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/user")
	}
}

func BenchmarkMatch_Param1(b *testing.B) {
	r := newTestRouter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/user/123")
	}
}

func BenchmarkMatch_Param2(b *testing.B) {
	r := newTestRouter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/user/123/posts/456")
	}
}

func BenchmarkMatch_Param5(b *testing.B) {
	r := newTestRouter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/blog/2024/01/15/hello-world")
	}
}

func BenchmarkMatch_Wildcard(b *testing.B) {
	r := newTestRouter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/assets/css/main.css")
	}
}

func BenchmarkMatch_404(b *testing.B) {
	r := newTestRouter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/nonexistent/path")
	}
}

/*************************************************************
 * Allocation analysis
 *************************************************************/

func TestAlloc_Match_Static(t *testing.T) {
	r := newTestRouter()
	allocs := int(testing.AllocsPerRun(100, func() {
		r.Match("GET", "/user")
	}))
	t.Logf("Static match allocs: %d", allocs)
}

func TestAlloc_Match_Param1(t *testing.T) {
	r := newTestRouter()
	allocs := int(testing.AllocsPerRun(100, func() {
		r.Match("GET", "/user/123")
	}))
	t.Logf("1-param match allocs: %d", allocs)
}

func TestAlloc_Match_Param2(t *testing.T) {
	r := newTestRouter()
	allocs := int(testing.AllocsPerRun(100, func() {
		r.Match("GET", "/user/123/posts/456")
	}))
	t.Logf("2-param match allocs: %d", allocs)
}

func TestAlloc_Match_Wildcard(t *testing.T) {
	r := newTestRouter()
	allocs := int(testing.AllocsPerRun(100, func() {
		r.Match("GET", "/assets/css/main.css")
	}))
	t.Logf("Wildcard match allocs: %d", allocs)
}

func TestAlloc_Match_404(t *testing.T) {
	r := newTestRouter()
	allocs := int(testing.AllocsPerRun(100, func() {
		r.Match("GET", "/nonexistent")
	}))
	t.Logf("404 match allocs: %d", allocs)
}

func TestAlloc_ServeHTTP_Static(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest("GET", "/user", nil)
	w := httptest.NewRecorder()
	allocs := int(testing.AllocsPerRun(100, func() {
		r.ServeHTTP(w, req)
	}))
	t.Logf("ServeHTTP static allocs: %d", allocs)
}

func TestAlloc_ServeHTTP_Param(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()
	allocs := int(testing.AllocsPerRun(100, func() {
		r.ServeHTTP(w, req)
	}))
	t.Logf("ServeHTTP param allocs: %d", allocs)
}
