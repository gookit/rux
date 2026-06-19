package core

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// newCtx is a small helper that produces a Context wired with a fresh
// request + recorder so each test gets a clean writer and request URL.
func newCtx(t *testing.T, method, target string) (*Context, *httptest.ResponseRecorder) {
	t.Helper()
	c := &Context{}
	w := httptest.NewRecorder()
	c.Init(w, httptest.NewRequest(method, target, nil))
	return c, w
}

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

// --- abort / control-flow ------------------------------------------

func TestContext_Abort_StopsHandlerChain(t *testing.T) {
	c, _ := newCtx(t, "GET", "/x")
	assert.False(t, c.IsAborted())
	c.Abort()
	assert.True(t, c.IsAborted())
}

func TestContext_AbortWithStatus_NoMessage(t *testing.T) {
	c, _ := newCtx(t, "GET", "/x")
	c.AbortWithStatus(http.StatusForbidden)
	// AbortWithStatus(noMsg) defers the actual WriteHeader to the wrapper's
	// ensureWriteHeader; assert via the API that reads the wrapper status.
	assert.Eq(t, http.StatusForbidden, c.StatusCode())
	assert.True(t, c.IsAborted())
}

func TestContext_AbortWithStatus_WithMessage(t *testing.T) {
	c, w := newCtx(t, "GET", "/x")
	c.AbortWithStatus(http.StatusBadRequest, "bad input")
	assert.Eq(t, http.StatusBadRequest, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "bad input"))
	assert.True(t, c.IsAborted())
}

func TestContext_AbortThen_ChainsCalls(t *testing.T) {
	c, _ := newCtx(t, "GET", "/x")
	got := c.AbortThen()
	assert.Same(t, c, got)
	assert.True(t, c.IsAborted())
}

// --- request introspection -----------------------------------------

func TestContext_URL(t *testing.T) {
	c, _ := newCtx(t, "GET", "/some/path?x=1")
	assert.Eq(t, "/some/path", c.URL().Path)
}

func TestContext_Header(t *testing.T) {
	c, _ := newCtx(t, "GET", "/x")
	c.Req.Header.Set("X-Custom", "v1")
	assert.Eq(t, "v1", c.Header("X-Custom"))
	assert.Eq(t, "", c.Header("X-Missing"))
}

func TestContext_Query(t *testing.T) {
	c, _ := newCtx(t, "GET", "/x?name=alice&empty=")
	assert.Eq(t, "alice", c.Query("name"))
	assert.Eq(t, "", c.Query("missing"))
	assert.Eq(t, "fallback", c.Query("missing", "fallback"))
	// Present-but-empty returns "" (not the default — present beats fallback).
	assert.Eq(t, "", c.Query("empty"))
}

func TestContext_ReqCtxValue(t *testing.T) {
	type k string
	c, _ := newCtx(t, "GET", "/x")
	c.Req = c.Req.WithContext(context.WithValue(c.Req.Context(), k("traceID"), "abc"))
	assert.Eq(t, "abc", c.ReqCtxValue(k("traceID")))
	assert.Nil(t, c.ReqCtxValue(k("missing")))
}

func TestContext_ClientIP(t *testing.T) {
	t.Run("X-Forwarded-For first value", func(t *testing.T) {
		c, _ := newCtx(t, "GET", "/x")
		c.Req.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
		assert.Eq(t, "203.0.113.7", c.ClientIP())
	})
	t.Run("X-Real-Ip fallback", func(t *testing.T) {
		c, _ := newCtx(t, "GET", "/x")
		c.Req.Header.Set("X-Real-Ip", "198.51.100.4")
		assert.Eq(t, "198.51.100.4", c.ClientIP())
	})
	t.Run("RemoteAddr fallback strips port", func(t *testing.T) {
		c, _ := newCtx(t, "GET", "/x")
		c.Req.RemoteAddr = "192.0.2.5:55432"
		assert.Eq(t, "192.0.2.5", c.ClientIP())
	})
	t.Run("malformed RemoteAddr → empty", func(t *testing.T) {
		c, _ := newCtx(t, "GET", "/x")
		c.Req.RemoteAddr = "not-a-host"
		assert.Eq(t, "", c.ClientIP())
	})
}

// --- error bag -----------------------------------------------------

func TestContext_AddErr_FirstErr_NilSafe(t *testing.T) {
	c, _ := newCtx(t, "GET", "/x")
	assert.Nil(t, c.Err())
	assert.Nil(t, c.FirstError())

	c.AddError(nil) // nil should be silently ignored
	assert.Nil(t, c.Err())

	a := errors.New("a")
	b := errors.New("b")
	c.AddError(a)
	c.AddError(b)
	assert.Eq(t, b, c.Err())       // most recent
	assert.Eq(t, a, c.FirstError()) // earliest
}

// --- data bag (SafeGet) --------------------------------------------

func TestContext_SafeGet_PresentValue(t *testing.T) {
	c, _ := newCtx(t, "GET", "/x")
	c.Set("k", 7)
	assert.Eq(t, 7, c.SafeGet("k"))
}

func TestContext_SafeGet_MissingPanics(t *testing.T) {
	c, _ := newCtx(t, "GET", "/x")
	defer func() {
		r := recover()
		assert.NotNil(t, r, "SafeGet on missing key should panic")
	}()
	_ = c.SafeGet("nope")
}

// --- params / route / matched path --------------------------------

func TestContext_Params_ReturnsPointer(t *testing.T) {
	c, _ := newCtx(t, "GET", "/x")
	c.params.append("id", "9")
	ps := c.Params()
	assert.Eq(t, "9", ps.Get("id"))
	// Returned pointer should alias the inline array (same backing storage).
	assert.Same(t, &c.params, ps)
}

func TestContext_Route_DefaultsNil(t *testing.T) {
	c, _ := newCtx(t, "GET", "/x")
	assert.Nil(t, c.Route())
	assert.Eq(t, "", c.MatchedPath())
}

// --- writer pass-through ------------------------------------------

func TestContext_WriteString_StatusCode_Length(t *testing.T) {
	c, w := newCtx(t, "GET", "/x")
	c.WriteString("hello")
	assert.Eq(t, "hello", w.Body.String())
	assert.Eq(t, 200, c.StatusCode())
	assert.Eq(t, 5, c.Length())
}

func TestContext_WriteBytes_EmptySliceStillSetsStatus(t *testing.T) {
	c, w := newCtx(t, "GET", "/x")
	c.WriteBytes(nil)
	// ensureWriteHeader still flushes the default 200.
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, 0, c.Length())
}

// --- response_writer.Hijack & SetStatus / SetHeader --------------

func TestContext_SetStatus(t *testing.T) {
	c, w := newCtx(t, "GET", "/x")
	c.SetStatus(http.StatusTeapot)
	// Trigger header flush by writing a byte.
	c.WriteString("")
	assert.Eq(t, http.StatusTeapot, w.Code)
}

func TestContext_SetHeader(t *testing.T) {
	c, w := newCtx(t, "GET", "/x")
	c.SetHeader("X-Trace", "trace-1")
	assert.Eq(t, "trace-1", w.Header().Get("X-Trace"))
}
