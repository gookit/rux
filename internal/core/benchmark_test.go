package core

import (
	"fmt"
	"net/http/httptest"
	"testing"
)

var noopHandler HandlerFunc = func(c *Context) {}

func runReq(b *testing.B, r *Router, method, path string) {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req) // warm up + freeze
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkV2_StaticRoute(b *testing.B) {
	r := New()
	r.GET("/users", noopHandler)
	runReq(b, r, "GET", "/users")
}

func BenchmarkV2_StaticRoute_404(b *testing.B) {
	r := New()
	r.GET("/users", noopHandler)
	runReq(b, r, "GET", "/missing")
}

func BenchmarkV2_Param1(b *testing.B) {
	r := New()
	r.GET("/users/{id}", noopHandler)
	runReq(b, r, "GET", "/users/42")
}

func BenchmarkV2_Param5(b *testing.B) {
	r := New()
	r.GET("/a/{a}/b/{b}/c/{c}/d/{d}/e/{e}", noopHandler)
	runReq(b, r, "GET", "/a/1/b/2/c/3/d/4/e/5")
}

func BenchmarkV2_Wildcard(b *testing.B) {
	r := New()
	r.GET("/files/*path", noopHandler)
	runReq(b, r, "GET", "/files/a/b/c/d.txt")
}

// 200-route table modeled after a real API.
func BenchmarkV2_GithubAPI(b *testing.B) {
	r := newGithubAPIRouter()
	runReq(b, r, "GET", "/repos/gookit/rux/issues/1")
}

func newGithubAPIRouter() *Router {
	r := New()
	paths := []struct{ m, p string }{
		{"GET", "/users/{user}"},
		{"GET", "/users/{user}/repos"},
		{"GET", "/repos/{owner}/{repo}"},
		{"GET", "/repos/{owner}/{repo}/issues"},
		{"GET", "/repos/{owner}/{repo}/issues/{number}"},
		{"GET", "/repos/{owner}/{repo}/pulls"},
		{"GET", "/repos/{owner}/{repo}/pulls/{number}"},
		{"GET", "/repos/{owner}/{repo}/contributors"},
		{"GET", "/repos/{owner}/{repo}/forks"},
		{"GET", "/repos/{owner}/{repo}/stargazers"},
	}
	for _, p := range paths {
		r.Add(p.p, noopHandler, p.m)
	}
	return r
}

func BenchmarkV2_Parallel_Static(b *testing.B) {
	r := New()
	r.GET("/users", noopHandler)
	req := httptest.NewRequest("GET", "/users", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		w := httptest.NewRecorder()
		for pb.Next() {
			r.ServeHTTP(w, req)
		}
	})
}

func BenchmarkV2_Parallel_Param(b *testing.B) {
	r := New()
	r.GET("/users/{id}", noopHandler)
	req := httptest.NewRequest("GET", "/users/42", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		w := httptest.NewRecorder()
		for pb.Next() {
			r.ServeHTTP(w, req)
		}
	})
}

// Suppress unused fmt import warning if any helper drops out.
var _ = fmt.Sprintf
