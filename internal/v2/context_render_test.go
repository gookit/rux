package v2

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestContext_Text(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) { c.Text(200, "hello") })
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	body, _ := io.ReadAll(w.Body)
	assert.Eq(t, "hello", string(body))
}

func TestContext_JSON_WritesContentType(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) { c.JSON(200, map[string]string{"k": "v"}) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Header().Get("Content-Type"), "json"))
}

func TestContext_Redirect_DefaultsTo302(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) { c.Redirect("/y") })
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	assert.Eq(t, 302, w.Code)
}

func TestContext_NoContent(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) { c.NoContent() })
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	assert.Eq(t, 204, w.Code)
}
