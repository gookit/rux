package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// namedMW records its name when the chain runs.
func namedMW(order *[]string, name string) HandlerFunc {
	return func(c *Context) {
		*order = append(*order, name)
		c.Next()
	}
}

func serveGet(r *Router, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	return w
}

func TestGroupValue_PrefixAndMiddleware(t *testing.T) {
	r := New()
	var order []string

	g := r.NewGroup("/api", namedMW(&order, "api"))
	g.GET("/users", func(c *Context) { c.Text(200, "list") })

	assert.Eq(t, "/api", g.Prefix())
	assert.Eq(t, r, g.Router())

	w := serveGet(r, "/api/users")
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "list", w.Body.String())
	assert.Eq(t, "api", strings.Join(order, ","))
}

func TestGroupValue_MiddlewareOrder(t *testing.T) {
	r := New()
	var order []string

	r.Use(namedMW(&order, "global"))
	api := r.NewGroup("/api", namedMW(&order, "group"))
	admin := api.NewGroup("/admin", namedMW(&order, "nested"))
	admin.GET("/stats", func(c *Context) {
		order = append(order, "handler")
		c.Text(200, "ok")
	}, namedMW(&order, "route"))

	w := serveGet(r, "/api/admin/stats")
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "global,group,nested,route,handler", strings.Join(order, ","))
}

func TestGroupValue_UseAppliesToLaterRoutes(t *testing.T) {
	r := New()
	var order []string

	g := r.NewGroup("/api")
	g.GET("/before", func(c *Context) { c.Text(200, "before") })
	g.Use(namedMW(&order, "late"))
	g.GET("/after", func(c *Context) { c.Text(200, "after") })

	serveGet(r, "/api/before")
	assert.Eq(t, "", strings.Join(order, ","), "Use must not retro-apply")

	serveGet(r, "/api/after")
	assert.Eq(t, "late", strings.Join(order, ","))
}

// A group value created inside a closure-style Group inherits its prefix and
// middleware.
func TestGroupValue_InheritsClosureGroup(t *testing.T) {
	r := New()
	var order []string

	r.Group("/api", func() {
		admin := r.NewGroup("/admin")
		assert.Eq(t, "/api/admin", admin.Prefix())
		admin.GET("/x", func(c *Context) { c.Text(200, "x") })
	}, namedMW(&order, "closure"))

	w := serveGet(r, "/api/admin/x")
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "closure", strings.Join(order, ","))
}

func TestGroupValue_Methods(t *testing.T) {
	r := New()
	g := r.NewGroup("/api")
	g.POST("/x", func(c *Context) { c.Text(200, "post") })
	g.PUT("/x", func(c *Context) { c.Text(200, "put") })
	g.Any("/any", func(c *Context) { c.Text(200, "any") })
	g.Add("/custom", func(c *Context) { c.Text(200, "custom") }, "GET", "DELETE")

	for _, tc := range []struct{ method, path, body string }{
		{"POST", "/api/x", "post"},
		{"PUT", "/api/x", "put"},
		{"GET", "/api/any", "any"},
		{"DELETE", "/api/any", "any"},
		{"GET", "/api/custom", "custom"},
		{"DELETE", "/api/custom", "custom"},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		assert.Eq(t, 200, w.Code, "%s %s", tc.method, tc.path)
		assert.Eq(t, tc.body, w.Body.String(), "%s %s", tc.method, tc.path)
	}
}

func TestGroupValue_AddNamed(t *testing.T) {
	r := New()
	g := r.NewGroup("/api")
	g.AddNamed("user_show", "/users/{id}", func(c *Context) { c.Text(200, "show") })

	route := r.GetRoute("user_show")
	assert.NotNil(t, route)
	assert.Eq(t, "/api/users/:id", route.Path())

	w := serveGet(r, "/api/users/7")
	assert.Eq(t, 200, w.Code)
}

func TestGroupValue_FreezePanics(t *testing.T) {
	r := New()
	g := r.NewGroup("/api")
	g.GET("/x", func(c *Context) { c.Text(200, "x") })
	serveGet(r, "/api/x")

	assert.Panics(t, func() { g.GET("/y", func(c *Context) {}) })
	assert.Panics(t, func() { g.Use(func(c *Context) { c.Next() }).GET("/z", func(c *Context) {}) })
}

func TestGroupValue_StaticDir(t *testing.T) {
	dir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o644))

	r := New()
	g := r.NewGroup("/api")
	g.StaticDir("/assets", dir)

	w := serveGet(r, "/api/assets/hello.txt")
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "hi", w.Body.String())
}

// The closure-style Group had the same stripping bug: the route was prefixed but
// the file server stripped the un-prefixed path.
func TestGroup_ClosureStyle_StaticDir(t *testing.T) {
	dir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o644))

	r := New()
	r.Group("/api", func() {
		r.StaticDir("/assets", dir)
	})

	w := serveGet(r, "/api/assets/hello.txt")
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "hi", w.Body.String())
}

func TestGroupValue_StaticFS(t *testing.T) {
	r := New()
	g := r.NewGroup("/api")
	g.StaticFS("/assets", http.Dir(t.TempDir()))

	// Route registered under the group prefix; a missing file is a 404 from the
	// file server (not from the router).
	w := serveGet(r, "/api/assets/missing.txt")
	assert.Eq(t, 404, w.Code)
}
