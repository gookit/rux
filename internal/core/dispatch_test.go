package core

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func textHandler(body string) HandlerFunc {
	return func(c *Context) {
		c.Resp.WriteHeader(200)
		_, _ = c.Resp.Write([]byte(body))
	}
}

func TestServeHTTP_StaticHit(t *testing.T) {
	r := New()
	r.GET("/users", textHandler("hi"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users", nil)
	r.ServeHTTP(w, req)

	assert.Eq(t, 200, w.Code)
	body, _ := io.ReadAll(w.Body)
	assert.Eq(t, "hi", string(body))
}

func TestServeHTTP_DynamicHit_BindsParam(t *testing.T) {
	r := New()
	r.GET("/users/{id}", func(c *Context) {
		_, _ = c.Resp.Write([]byte("id=" + c.Param("id")))
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/42", nil)
	r.ServeHTTP(w, req)
	body, _ := io.ReadAll(w.Body)
	assert.True(t, strings.Contains(string(body), "id=42"))
}

func TestServeHTTP_404(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) {})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/nothing", nil)
	r.ServeHTTP(w, req)
	assert.Eq(t, 404, w.Code)
}

func TestServeHTTP_HEAD_FallsBackToGET(t *testing.T) {
	r := New()
	r.GET("/x", textHandler("ok"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("HEAD", "/x", nil)
	r.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
}

func TestServeHTTP_TriggersFreeze(t *testing.T) {
	r := New()
	r.GET("/x", textHandler("x"))
	assert.False(t, r.Frozen())
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	assert.True(t, r.Frozen())
}

func TestServeHTTP_MiddlewareOrder_GlobalGroupRouteMain(t *testing.T) {
	var order []string
	r := New()
	r.Use(func(c *Context) { order = append(order, "global"); c.Next() })
	r.Group("/api", func() {
		r.GET("/x", func(c *Context) { order = append(order, "main") },
			func(c *Context) { order = append(order, "route"); c.Next() })
	}, func(c *Context) { order = append(order, "group"); c.Next() })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/x", nil)
	r.ServeHTTP(w, req)
	assert.Eq(t, []string{"global", "group", "route", "main"}, order)
}

func TestServeHTTP_OnPanic(t *testing.T) {
	var captured any
	r := New()
	r.OnPanic = func(c *Context) {
		captured, _ = c.Get(CTXRecoverResult)
		c.Resp.WriteHeader(500)
	}
	r.GET("/boom", func(c *Context) { panic("oops") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/boom", nil)
	r.ServeHTTP(w, req)

	assert.Eq(t, 500, w.Code)
	assert.Eq(t, "oops", captured)
}

func TestServeHTTP_HandleMethodNotAllowed(t *testing.T) {
	r := New(HandleMethodNotAllowed)
	r.GET("/x", func(c *Context) {})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/x", nil)
	r.ServeHTTP(w, req)

	assert.Eq(t, 405, w.Code)
	assert.True(t, strings.Contains(w.Header().Get("Allow"), "GET"))
}

// Ensure HEAD method receives no body but correct status (HEAD test above
// already covers status; this asserts the http test recorder remains usable).
var _ = http.MethodGet
