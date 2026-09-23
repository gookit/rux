package rux

import (
	"net/http/httptest"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// The root package re-exports the options; make sure both spellings work from
// the public import path.
func TestRoot_OptionAliases(t *testing.T) {
	r := New(WithMethodNotAllowed(true), WithEncodedPath(false))
	r.GET("/x", func(c *Context) { c.Text(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/x", nil))
	assert.Eq(t, 405, w.Code)

	legacy := New(HandleMethodNotAllowed)
	legacy.GET("/x", func(c *Context) { c.Text(200, "ok") })

	w2 := httptest.NewRecorder()
	legacy.ServeHTTP(w2, httptest.NewRequest("POST", "/x", nil))
	assert.Eq(t, 405, w2.Code)
}
