package server

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/rux/v2"
)

func TestHealthz_AlwaysOK(t *testing.T) {
	s := New(false)
	s.MountHealthChecks()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	s.Router.ServeHTTP(w, req)

	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "ok", w.Body.String())
}

// MountHealthChecks registers routes. Use is not order-restricted before the
// first request (global middleware is merged when the router freezes), so both
// orders work, and the global chain covers the health endpoints either way.
func TestMountHealthChecks_UseOrderIsFlexible(t *testing.T) {
	// the documented order: middleware first
	first := New(false)
	first.Use(setGlobalHeader)
	first.MountHealthChecks()
	assertHealthHasGlobalHeader(t, first)

	// mounting first is fine too
	mounted := New(false)
	mounted.MountHealthChecks()
	mounted.Use(setGlobalHeader)
	assertHealthHasGlobalHeader(t, mounted)

	// once the router has served a request it is frozen, and Use panics
	frozen := New(false)
	frozen.MountHealthChecks()
	frozen.Router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/healthz", nil))

	defer func() {
		rec := recover()
		assert.NotNil(t, rec, "Use after the first request must panic")
		assert.StrContains(t, fmt.Sprint(rec), "frozen")
	}()
	frozen.Use(func(c *rux.Context) { c.Next() })
}

func setGlobalHeader(c *rux.Context) {
	c.SetHeader("X-Global", "1")
	c.Next()
}

func assertHealthHasGlobalHeader(t *testing.T, s *Server) {
	t.Helper()

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "1", w.Header().Get("X-Global"))
}

func TestReadyz_NotReadyBeforeRun(t *testing.T) {
	s := New(false)
	s.MountHealthChecks()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/readyz", nil)
	s.Router.ServeHTTP(w, req)

	assert.Eq(t, 503, w.Code)
}

func TestReadyz_ReadyAfterStart(t *testing.T) {
	s := New(false)
	s.MountHealthChecks()
	s.ready.Store(true) // simulate after PostStart

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/readyz", nil)
	s.Router.ServeHTTP(w, req)

	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "ready", w.Body.String())
}

func TestReadyz_FailingCheck(t *testing.T) {
	s := New(false)
	s.ReadyChecks = append(s.ReadyChecks, func(ctx context.Context) error {
		return errors.New("db down")
	})
	s.MountHealthChecks()
	s.ready.Store(true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/readyz", nil)
	s.Router.ServeHTTP(w, req)

	assert.Eq(t, 503, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "db down"))
}

func TestReadyz_503DuringDrain(t *testing.T) {
	s := New(false)
	s.MountHealthChecks()
	s.ready.Store(true)
	s.draining.Store(true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/readyz", nil)
	s.Router.ServeHTTP(w, req)

	assert.Eq(t, 503, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "draining"))
}
