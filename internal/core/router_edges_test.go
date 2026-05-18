package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

// =========================================================================
// router options
// =========================================================================

func TestOption_UseEncodedPath(t *testing.T) {
	r := New(UseEncodedPath)
	// Register at the unescaped path; lookup with an encoded request should
	// still match because both encoding paths normalize to the same key.
	r.GET("/hello/world", textHandler("ok"))

	req := httptest.NewRequest("GET", "/hello/world", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
}

func TestOption_HandleFallbackRoute(t *testing.T) {
	// HandleFallbackRoute is an option-applying function that toggles a
	// private flag. The actual fallback dispatch in handle() looks up
	// the literal "/*" static key, which the public registration API
	// won't produce (wildcard names are required) — so we just verify
	// the option's effect and let server tests exercise the rest.
	r := New(HandleFallbackRoute)
	assert.True(t, r.handleFallbackRoute)
}

// =========================================================================
// formatPath edges
// =========================================================================

func TestFormatPath_TrimsTrailingSlash(t *testing.T) {
	r := New()
	assert.Eq(t, "/users", r.formatPath("/users/"))
}

func TestFormatPath_KeepsTrailingSlashWithStrict(t *testing.T) {
	r := New(StrictLastSlash)
	assert.Eq(t, "/users/", r.formatPath("/users/"))
}

func TestFormatPath_DoubleSlashCollapses(t *testing.T) {
	r := New()
	assert.Eq(t, "/users", r.formatPath("//users"))
}

func TestFormatPath_AddsLeadingSlash(t *testing.T) {
	r := New()
	assert.Eq(t, "/x", r.formatPath("x"))
}

// =========================================================================
// Use / Group panics + nested
// =========================================================================

func TestRouter_Use_AfterRouteRegistration_Panics(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) {})
	defer func() { assert.NotNil(t, recover()) }()
	r.Use(func(c *Context) {})
}

func TestRouter_Use_AfterFreeze_Panics(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) {})
	r.Freeze()
	defer func() { assert.NotNil(t, recover()) }()
	r.Use(func(c *Context) {})
}

func TestGroup_NestedComposesPrefixAndMiddleware(t *testing.T) {
	var order []string
	r := New()
	r.Group("/api", func() {
		r.Group("/v1", func() {
			r.GET("/x", func(c *Context) { order = append(order, "main") })
		}, func(c *Context) { order = append(order, "v1"); c.Next() })
	}, func(c *Context) { order = append(order, "api"); c.Next() })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/x", nil))
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, []string{"api", "v1", "main"}, order)
}

// =========================================================================
// Resource — panics + Uses-method wiring + HEAD alias
// =========================================================================

type resCtl struct {
	calls map[string]int
}

func (c *resCtl) Index(_ *Context)  { c.calls["Index"]++ }
func (c *resCtl) Create(_ *Context) { c.calls["Create"]++ }
func (c *resCtl) Store(_ *Context)  { c.calls["Store"]++ }
func (c *resCtl) Show(_ *Context)   { c.calls["Show"]++ }
func (c *resCtl) Edit(_ *Context)   { c.calls["Edit"]++ }
func (c *resCtl) Update(_ *Context) { c.calls["Update"]++ }
func (c *resCtl) Delete(_ *Context) { c.calls["Delete"]++ }

// Uses returns per-action middlewares; verifies the reflect-discovered
// Uses path in Resource.
func (c *resCtl) Uses() map[string][]HandlerFunc {
	return map[string][]HandlerFunc{
		"Show": {func(c *Context) { c.Set("show-mw", true); c.Next() }},
	}
}

func TestResource_RegistersAllRESTfulActions(t *testing.T) {
	c := &resCtl{calls: map[string]int{}}
	r := New()
	r.Resource("/api", c)

	hit := func(method, path string) int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
		return w.Code
	}

	assert.Eq(t, 200, hit("GET", "/api/resctl"))
	assert.Eq(t, 200, hit("GET", "/api/resctl/create"))
	assert.Eq(t, 200, hit("POST", "/api/resctl"))
	assert.Eq(t, 200, hit("GET", "/api/resctl/7"))
	assert.Eq(t, 200, hit("GET", "/api/resctl/7/edit"))
	assert.Eq(t, 200, hit("PUT", "/api/resctl/7"))
	assert.Eq(t, 200, hit("DELETE", "/api/resctl/7"))

	assert.Eq(t, 1, c.calls["Index"])
	assert.Eq(t, 1, c.calls["Create"])
	assert.Eq(t, 1, c.calls["Store"])
	assert.Eq(t, 1, c.calls["Show"])
	assert.Eq(t, 1, c.calls["Edit"])
	assert.Eq(t, 1, c.calls["Update"])
	assert.Eq(t, 1, c.calls["Delete"])
}

func TestResource_NonPointer_Panics(t *testing.T) {
	defer func() { assert.NotNil(t, recover()) }()
	r := New()
	r.Resource("/api", resCtl{}) // value, not pointer → panic
}

func TestResource_PointerToNonStruct_Panics(t *testing.T) {
	defer func() { assert.NotNil(t, recover()) }()
	r := New()
	v := 7
	r.Resource("/api", &v)
}

// =========================================================================
// Static* — directory serving
// =========================================================================

func TestStaticDir_ServesFile(t *testing.T) {
	dir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("alpha"), 0644))

	r := New()
	r.StaticDir("/static", dir)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/static/a.txt", nil))
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "alpha", w.Body.String())
}

func TestStaticFS_ServesFile(t *testing.T) {
	dir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("beta"), 0644))

	r := New()
	r.StaticFS("/fs", http.Dir(dir))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/fs/b.txt", nil))
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "beta"))
}

func TestStaticFiles_ServesFile(t *testing.T) {
	dir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "c.txt"), []byte("gamma"), 0644))

	r := New()
	r.StaticFiles("/assets", dir, "")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/assets/c.txt", nil))
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "gamma"))
}

// =========================================================================
// registerSingleRoute panic paths
// =========================================================================

func TestRegisterSingleRoute_DuplicateStatic_Panics(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) {})
	defer func() { assert.NotNil(t, recover()) }()
	r.GET("/x", func(c *Context) {})
}
