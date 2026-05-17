package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestContext_Param_ReadsFromInlineParams(t *testing.T) {
	c := &Context{}
	c.params.append("id", "42")
	assert.Eq(t, "42", c.Param("id"))
	assert.Eq(t, "", c.Param("missing"))
}

func TestContext_SetGet_LazyMap(t *testing.T) {
	c := &Context{}
	_, ok := c.Get("missing")
	assert.False(t, ok)
	assert.Nil(t, c.data, "data map should not be allocated until Set is called")

	c.Set("k", 42)
	v, ok := c.Get("k")
	assert.True(t, ok)
	assert.Eq(t, 42, v)
}

func TestContext_Init_AssignsRequestAndResponse(t *testing.T) {
	c := &Context{}
	req := httptest.NewRequest("GET", "/x", nil)
	w := httptest.NewRecorder()
	c.Init(w, req)
	assert.Same(t, req, c.Req)
	assert.NotNil(t, c.Resp)
}

func TestContext_SetCookie(t *testing.T) {
	c := &Context{}
	w := httptest.NewRecorder()
	c.Init(w, httptest.NewRequest("GET", "/x", nil))
	c.SetCookie("session", "abc123", 3600, "/", "", false, true)

	setCookie := w.Header().Get("Set-Cookie")
	assert.True(t, strings.Contains(setCookie, "session=abc123"))
	assert.True(t, strings.Contains(setCookie, "HttpOnly"))
}

func TestContext_FastSetCookie(t *testing.T) {
	c := &Context{}
	w := httptest.NewRecorder()
	c.Init(w, httptest.NewRequest("GET", "/x", nil))
	c.FastSetCookie("token", "xyz", 60)

	setCookie := w.Header().Get("Set-Cookie")
	assert.True(t, strings.Contains(setCookie, "token=xyz"))
	assert.True(t, strings.Contains(setCookie, "Path=/"))
	assert.True(t, strings.Contains(setCookie, "HttpOnly"))
	// Secure off by default — see FastSetCookie godoc.
	assert.False(t, strings.Contains(setCookie, "Secure"))
}

func TestContext_FastSetCookie_WithOpts(t *testing.T) {
	c := &Context{}
	w := httptest.NewRecorder()
	c.Init(w, httptest.NewRequest("GET", "/x", nil))
	c.FastSetCookie("session", "v", 3600,
		func(ck *http.Cookie) { ck.Secure = true },
		func(ck *http.Cookie) { ck.SameSite = http.SameSiteStrictMode },
		func(ck *http.Cookie) { ck.Domain = "example.com" },
	)

	setCookie := w.Header().Get("Set-Cookie")
	assert.True(t, strings.Contains(setCookie, "session=v"))
	assert.True(t, strings.Contains(setCookie, "Secure"))
	assert.True(t, strings.Contains(setCookie, "SameSite=Strict"))
	assert.True(t, strings.Contains(setCookie, "Domain=example.com"))
	// Defaults still preserved unless an opt overrode them.
	assert.True(t, strings.Contains(setCookie, "Path=/"))
	assert.True(t, strings.Contains(setCookie, "HttpOnly"))
}

func TestContext_DelCookie(t *testing.T) {
	c := &Context{}
	w := httptest.NewRecorder()
	c.Init(w, httptest.NewRequest("GET", "/x", nil))
	c.DelCookie("a", "b")

	cookies := w.Header().Values("Set-Cookie")
	assert.Len(t, cookies, 2)
	for _, ck := range cookies {
		assert.True(t, strings.Contains(ck, "Max-Age=0"))
	}
}

func TestContext_Cookie(t *testing.T) {
	c := &Context{}
	req := httptest.NewRequest("GET", "/x", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "xyz"})
	c.Init(httptest.NewRecorder(), req)
	assert.Eq(t, "xyz", c.Cookie("token"))
	assert.Eq(t, "", c.Cookie("missing"))
}
