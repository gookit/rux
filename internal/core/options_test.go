package core

import (
	"net/http/httptest"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestOptions_WithFormsSetValue(t *testing.T) {
	r := New(
		WithStrictLastSlash(true),
		WithEncodedPath(true),
		WithMethodNotAllowed(true),
		WithFallbackRoute(true),
	)

	assert.True(t, r.strictLastSlash)
	assert.True(t, r.useEncodedPath)
	assert.True(t, r.handleMethodNotAllowed)
	assert.True(t, r.handleFallbackRoute)
}

// The value form is the point: a setting can be switched off explicitly.
func TestOptions_WithFormsCanDisable(t *testing.T) {
	r := New(
		WithStrictLastSlash(false),
		WithEncodedPath(false),
		WithMethodNotAllowed(false),
		WithFallbackRoute(false),
	)

	assert.False(t, r.strictLastSlash)
	assert.False(t, r.useEncodedPath)
	assert.False(t, r.handleMethodNotAllowed)
	assert.False(t, r.handleFallbackRoute)

	// Later options win, so an override works.
	r2 := New(WithMethodNotAllowed(true), WithMethodNotAllowed(false))
	assert.False(t, r2.handleMethodNotAllowed)
}

func TestOptions_LegacyAliasesStillEnable(t *testing.T) {
	r := New(StrictLastSlash, UseEncodedPath, HandleMethodNotAllowed, HandleFallbackRoute)

	assert.True(t, r.strictLastSlash)
	assert.True(t, r.useEncodedPath)
	assert.True(t, r.handleMethodNotAllowed)
	assert.True(t, r.handleFallbackRoute)
}

// The flag must actually change dispatch behavior, not just the field.
func TestOptions_MethodNotAllowedFlagControls405(t *testing.T) {
	off := New(WithMethodNotAllowed(false))
	off.GET("/x", func(c *Context) { c.Text(200, "ok") })
	w := httptest.NewRecorder()
	off.ServeHTTP(w, httptest.NewRequest("POST", "/x", nil))
	assert.Eq(t, 404, w.Code)

	on := New(WithMethodNotAllowed(true))
	on.GET("/x", func(c *Context) { c.Text(200, "ok") })
	w2 := httptest.NewRecorder()
	on.ServeHTTP(w2, httptest.NewRequest("POST", "/x", nil))
	assert.Eq(t, 405, w2.Code)
}
