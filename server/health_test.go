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

// MountHealthChecks registers routes, and Use must precede any registration, so
// the documented order is Use -> MountHealthChecks -> business routes. The
// middleware must then cover the health endpoints as well.
func TestMountHealthChecks_MustFollowUse(t *testing.T) {
	s := New(false)
	s.Use(func(c *rux.Context) {
		c.SetHeader("X-Global", "1")
		c.Next()
	})
	s.MountHealthChecks()

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "1", w.Header().Get("X-Global"))

	// The reverse order panics, and the message says helpers count as registration.
	bad := New(false)
	bad.MountHealthChecks()

	defer func() {
		rec := recover()
		assert.NotNil(t, rec, "Use after MountHealthChecks must panic")
		assert.StrContains(t, fmt.Sprint(rec), "before any route registration")
		assert.StrContains(t, fmt.Sprint(rec), "health checks")
	}()
	bad.Use(func(c *rux.Context) { c.Next() })
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
