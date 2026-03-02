package fastrux_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gookit/rux/fastrux"
)

/*************************************************************
 * Comparison benchmarks with original rux
 *************************************************************/

// These tests compare fastrux performance against common patterns

func BenchmarkFastrux_ParseGithubAPI(b *testing.B) {
	r := loadGithubAPI()
	runRequest(b, r, "GET", "/repos/gookit/rux")
}

func BenchmarkFastrux_ParseGithubAPI_Param(b *testing.B) {
	r := loadGithubAPI()
	runRequest(b, r, "GET", "/repos/gookit/rux/pulls/123")
}

func BenchmarkFastrux_ParseGithubAPI_5Params(b *testing.B) {
	r := loadGithubAPI()
	runRequest(b, r, "GET", "/repos/gookit/rux/pulls/123/comments/456/reactions")
}

// loadGithubAPI creates a router with GitHub-like routes
func loadGithubAPI() *fastrux.Router {
	r := fastrux.New()

	// User endpoints
	r.GET("/users/{user}", emptyHandler)
	r.GET("/users/{user}/repos", emptyHandler)
	r.GET("/users/{user}/following", emptyHandler)
	r.GET("/users/{user}/followers", emptyHandler)

	// Repository endpoints
	r.GET("/repos/{owner}/{repo}", emptyHandler)
	r.GET("/repos/{owner}/{repo}/issues", emptyHandler)
	r.GET("/repos/{owner}/{repo}/issues/{number}", emptyHandler)
	r.GET("/repos/{owner}/{repo}/pulls", emptyHandler)
	r.GET("/repos/{owner}/{repo}/pulls/{number}", emptyHandler)
	r.GET("/repos/{owner}/{repo}/pulls/{number}/comments", emptyHandler)
	r.GET("/repos/{owner}/{repo}/pulls/{number}/comments/{id}", emptyHandler)
	r.GET("/repos/{owner}/{repo}/pulls/{number}/comments/{id}/reactions", emptyHandler)
	r.GET("/repos/{owner}/{repo}/commits", emptyHandler)
	r.GET("/repos/{owner}/{repo}/commits/{sha}", emptyHandler)
	r.GET("/repos/{owner}/{repo}/branches", emptyHandler)
	r.GET("/repos/{owner}/{repo}/tags", emptyHandler)

	// Organization endpoints
	r.GET("/orgs/{org}", emptyHandler)
	r.GET("/orgs/{org}/repos", emptyHandler)
	r.GET("/orgs/{org}/members", emptyHandler)
	r.GET("/orgs/{org}/teams", emptyHandler)

	// Gist endpoints
	r.GET("/gists", emptyHandler)
	r.GET("/gists/{id}", emptyHandler)
	r.GET("/gists/{id}/comments", emptyHandler)

	return r
}

/*************************************************************
 * Memory pressure tests
 *************************************************************/

func BenchmarkFastrux_HighLoad_Static(b *testing.B) {
	r := newTestRouter()
	req, _ := http.NewRequest("GET", "/user", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.ServeHTTP(w, req)
		}
	})
}

func BenchmarkFastrux_HighLoad_Param(b *testing.B) {
	r := newTestRouter()
	req, _ := http.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.ServeHTTP(w, req)
		}
	})
}

func BenchmarkFastrux_HighLoad_5Params(b *testing.B) {
	r := newTestRouter()
	req, _ := http.NewRequest("GET", "/blog/2024/01/15/hello-world", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.ServeHTTP(w, req)
		}
	})
}

/*************************************************************
 * Large routing table tests
 *************************************************************/

func BenchmarkFastrux_LargeTable_First(b *testing.B) {
	r := createLargeRoutingTable()
	runRequest(b, r, "GET", "/route0")
}

func BenchmarkFastrux_LargeTable_Middle(b *testing.B) {
	r := createLargeRoutingTable()
	runRequest(b, r, "GET", "/route500")
}

func BenchmarkFastrux_LargeTable_Last(b *testing.B) {
	r := createLargeRoutingTable()
	runRequest(b, r, "GET", "/route999")
}

func BenchmarkFastrux_LargeTable_Param(b *testing.B) {
	r := createLargeRoutingTable()
	runRequest(b, r, "GET", "/param/500/data")
}

func createLargeRoutingTable() *fastrux.Router {
	r := fastrux.New()

	// Add 1000 static routes
	for i := 0; i < 1000; i++ {
		r.GET("/route"+string(rune(i)), emptyHandler)
	}

	// Add 100 parameterized routes
	for i := 0; i < 100; i++ {
		r.GET("/param/{id}/data", emptyHandler)
		r.GET("/param/{id}/meta", emptyHandler)
	}

	return r
}

/*************************************************************
 * Edge case tests
 *************************************************************/

func BenchmarkFastrux_DeepNesting(b *testing.B) {
	r := fastrux.New()
	r.GET("/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p", emptyHandler)
	runRequest(b, r, "GET", "/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p")
}

func BenchmarkFastrux_LongParam(b *testing.B) {
	r := fastrux.New()
	r.GET("/user/{id}", emptyHandler)
	// Create a long but valid parameter value
	longID := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789"
	runRequest(b, r, "GET", "/user/"+longID)
}

func BenchmarkFastrux_ManyMethods(b *testing.B) {
	r := fastrux.New()
	r.GET("/resource", emptyHandler)
	r.POST("/resource", emptyHandler)
	r.PUT("/resource", emptyHandler)
	r.PATCH("/resource", emptyHandler)
	r.DELETE("/resource", emptyHandler)
	r.OPTIONS("/resource", emptyHandler)
	runRequest(b, r, "GET", "/resource")
}

/*************************************************************
 * Specific optimization verification
 *************************************************************/

func BenchmarkFastrux_NoParams_ZeroAlloc(b *testing.B) {
	r := fastrux.New()
	r.GET("/static/path", emptyHandler)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := r.Match("GET", "/static/path")
		if result.Route == nil {
			b.Fatal("route not found")
		}
	}
}

func BenchmarkFastrux_Match_Reuse(b *testing.B) {
	r := fastrux.New()
	r.GET("/user/{id}", emptyHandler)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := r.Match("GET", "/user/123")
		if result.Route == nil {
			b.Fatal("route not found")
		}
		// Explicitly release for reuse
		r.ReleaseMatchResult(result)
	}
}

func BenchmarkFastrux_ContextPool_Efficiency(b *testing.B) {
	r := fastrux.New()
	r.GET("/test", emptyHandler)
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}
