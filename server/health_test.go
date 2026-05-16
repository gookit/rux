package server

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
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
