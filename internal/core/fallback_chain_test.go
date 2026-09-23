package core

import (
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// globalMW returns a global middleware that records how often it ran and sets
// an observable header, then continues the chain.
func globalMW(hits *int32, header string) HandlerFunc {
	return func(c *Context) {
		atomic.AddInt32(hits, 1)
		if header != "" {
			c.Resp.Header().Set(header, "1")
		}
		c.Next()
	}
}

func TestFallback_404RunsGlobalMiddleware(t *testing.T) {
	r := New()
	var hits int32
	r.Use(globalMW(&hits, "X-Global"))
	r.NotFound(func(c *Context) { c.Text(404, "custom nf") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/some/unknown/path", nil))

	assert.Eq(t, 404, w.Code)
	assert.Eq(t, "1", w.Header().Get("X-Global"))
	assert.Eq(t, int32(1), atomic.LoadInt32(&hits))
	assert.Eq(t, "custom nf", w.Body.String())
}

func TestFallback_404DefaultRunsGlobalMiddleware(t *testing.T) {
	r := New()
	var hits int32
	r.Use(globalMW(&hits, "X-Global"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/nope", nil))

	assert.Eq(t, 404, w.Code)
	assert.Eq(t, "1", w.Header().Get("X-Global"))
	assert.Eq(t, int32(1), atomic.LoadInt32(&hits))
}

func TestFallback_405RunsGlobalMiddleware(t *testing.T) {
	r := New(HandleMethodNotAllowed)
	var hits int32
	r.Use(globalMW(&hits, "X-Global"))
	r.POST("/x", func(c *Context) { c.Text(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))

	assert.Eq(t, 405, w.Code)
	assert.Eq(t, "POST", w.Header().Get("Allow"))
	assert.Eq(t, "1", w.Header().Get("X-Global"))
	assert.Eq(t, int32(1), atomic.LoadInt32(&hits))
}

// A global auth middleware must be able to reject an unknown path before the
// fallback handler runs (the eget scenario: 401, not 404).
func TestFallback_GlobalMiddlewareCanReject(t *testing.T) {
	r := New()
	fallbackRan := false
	r.Use(func(c *Context) {
		if c.Header("Authorization") == "" {
			c.AbortWithStatus(401)
			return
		}
		c.Next()
	})
	r.NotFound(func(c *Context) {
		fallbackRan = true
		c.Text(404, "nf")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/nope", nil))
	assert.Eq(t, 401, w.Code)
	assert.False(t, fallbackRan)

	w2 := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/nope", nil)
	req.Header.Set("Authorization", "Bearer t")
	r.ServeHTTP(w2, req)
	assert.Eq(t, 404, w2.Code)
	assert.True(t, fallbackRan)
}

// Serving must never write to the Router: the fallback chains are composed once
// in Freeze, so concurrent requests can't race on noRoute/noAllowed.
func TestFallback_ChainsComposedInFreezeNotWhileServing(t *testing.T) {
	r := New(HandleMethodNotAllowed)
	var hits int32
	r.Use(globalMW(&hits, ""))
	r.POST("/x", func(c *Context) { c.Text(200, "ok") })

	assert.Eq(t, 0, len(r.noRoute))
	assert.Eq(t, 0, len(r.noAllowed))

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/nope", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))

	// Still untouched by the serving path.
	assert.Eq(t, 0, len(r.noRoute))
	assert.Eq(t, 0, len(r.noAllowed))
	// The composed chains exist and carry the global middleware.
	assert.Eq(t, 2, len(r.noRouteChain))
	assert.Eq(t, 2, len(r.noAllowedChain))
	assert.Eq(t, int32(2), atomic.LoadInt32(&hits))
}

func TestFallback_ComposedWithoutGlobalChain(t *testing.T) {
	r := New()
	r.NotFound(func(c *Context) { c.Text(404, "nf") })
	r.Freeze()

	assert.Eq(t, 1, len(r.noRouteChain))
	assert.Eq(t, 1, len(r.noAllowedChain)) // internal 405 default
}

func TestFallback_RegistrationAfterFreezePanics(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) { c.Text(200, "ok") })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))

	assert.Panics(t, func() { r.NotFound(func(c *Context) {}) })
	assert.Panics(t, func() { r.NotAllowed(func(c *Context) {}) })
}
