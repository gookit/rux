package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/rux/v2"
)

// mockEcho dispatches a synthetic request to a fresh echo server.
func mockEcho(_ *testing.T, method, path string, body io.Reader) *httptest.ResponseRecorder {
	s := NewEchoServer()
	req := httptest.NewRequest(method, path, body)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}

func TestEcho_Home(t *testing.T) {
	w := mockEcho(t, "GET", "/", nil)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Header().Get("Content-Type"), "html"))
	assert.True(t, strings.Contains(w.Body.String(), "/anything"))
}

func TestEcho_Anything(t *testing.T) {
	w := mockEcho(t, "GET", "/anything", nil)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), `"method": "GET"`))
}

func TestEcho_AnythingWithPath(t *testing.T) {
	w := mockEcho(t, "GET", "/anything/foo/bar", nil)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), `"method": "GET"`))
}

func TestEcho_Anything_POSTBody(t *testing.T) {
	s := NewEchoServer()
	req := httptest.NewRequest("POST", "/anything",
		strings.NewReader(`{"hello":"world"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
	body := w.Body.String()
	assert.True(t, strings.Contains(body, `"method": "POST"`))
	assert.True(t, strings.Contains(body, `"hello"`))
}

func TestEcho_GetEndpoint(t *testing.T) {
	w := mockEcho(t, "GET", "/get", nil)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), `"method": "GET"`))
}

func TestEcho_PostEndpoint(t *testing.T) {
	w := mockEcho(t, "POST", "/post", strings.NewReader("hello"))
	assert.Eq(t, 200, w.Code)
}

func TestEcho_PutEndpoint(t *testing.T) {
	w := mockEcho(t, "PUT", "/put", strings.NewReader("hi"))
	assert.Eq(t, 200, w.Code)
}

func TestEcho_PatchEndpoint(t *testing.T) {
	w := mockEcho(t, "PATCH", "/patch", strings.NewReader("hi"))
	assert.Eq(t, 200, w.Code)
}

func TestEcho_DeleteEndpoint(t *testing.T) {
	w := mockEcho(t, "DELETE", "/delete", nil)
	assert.Eq(t, 200, w.Code)
}

func TestEcho_MethodRestricted(t *testing.T) {
	// GET /post should be 404: we only registered POST /post.
	w := mockEcho(t, "GET", "/post", nil)
	assert.Eq(t, 404, w.Code)
}

func TestEcho_Headers(t *testing.T) {
	s := NewEchoServer()
	req := httptest.NewRequest("GET", "/headers", nil)
	req.Header.Set("X-Custom", "hello")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
	body := w.Body.String()
	assert.True(t, strings.Contains(body, "X-Custom"))
	assert.True(t, strings.Contains(body, `"headers"`))
}

func TestEcho_IP(t *testing.T) {
	w := mockEcho(t, "GET", "/ip", nil)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), `"origin"`))
}

func TestEcho_UserAgent(t *testing.T) {
	s := NewEchoServer()
	req := httptest.NewRequest("GET", "/user-agent", nil)
	req.Header.Set("User-Agent", "test-agent/1.0")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "test-agent/1.0"))
}

func TestEcho_Status(t *testing.T) {
	w := mockEcho(t, "GET", "/status/418", nil)
	assert.Eq(t, 418, w.Code)
}

func TestEcho_StatusInvalid_DefaultsTo200(t *testing.T) {
	w := mockEcho(t, "GET", "/status/9999", nil)
	assert.Eq(t, 200, w.Code)
}

func TestEcho_Status_PostAlsoWorks(t *testing.T) {
	w := mockEcho(t, "POST", "/status/201", nil)
	assert.Eq(t, 201, w.Code)
}

func TestEcho_Delay(t *testing.T) {
	start := time.Now()
	w := mockEcho(t, "GET", "/delay/1", nil)
	elapsed := time.Since(start)
	assert.Eq(t, 200, w.Code)
	assert.True(t, elapsed >= time.Second, "should sleep at least 1s")
}

func TestEcho_Delay_CappedAt10(t *testing.T) {
	// Negative input should be normalized to zero (no sleep).
	start := time.Now()
	w := mockEcho(t, "GET", "/delay/-5", nil)
	elapsed := time.Since(start)
	assert.Eq(t, 200, w.Code)
	assert.True(t, elapsed < time.Second, "negative delay should not sleep")
}

func TestEcho_Redirect(t *testing.T) {
	w := mockEcho(t, "GET", "/redirect/3", nil)
	assert.Eq(t, 302, w.Code)
	assert.Eq(t, "/redirect/2", w.Header().Get("Location"))
}

func TestEcho_RedirectZero_GoesToGet(t *testing.T) {
	w := mockEcho(t, "GET", "/redirect/0", nil)
	assert.Eq(t, 302, w.Code)
	assert.Eq(t, "/get", w.Header().Get("Location"))
}

func TestEcho_Cookies(t *testing.T) {
	s := NewEchoServer()
	req := httptest.NewRequest("GET", "/cookies", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "session"))
}

func TestEcho_CookiesSet(t *testing.T) {
	w := mockEcho(t, "GET", "/cookies/set/foo/bar", nil)
	assert.Eq(t, 302, w.Code)
	setCookie := w.Header().Get("Set-Cookie")
	assert.True(t, strings.Contains(setCookie, "foo=bar"))
	assert.Eq(t, "/cookies", w.Header().Get("Location"))
}

func TestEcho_BasicAuth_Missing(t *testing.T) {
	w := mockEcho(t, "GET", "/basic-auth/user/pass", nil)
	assert.Eq(t, 401, w.Code)
	assert.True(t, strings.Contains(w.Header().Get("WWW-Authenticate"), "Basic"))
}

func TestEcho_BasicAuth_Wrong(t *testing.T) {
	s := NewEchoServer()
	req := httptest.NewRequest("GET", "/basic-auth/user/pass", nil)
	req.SetBasicAuth("user", "wrongpass")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Eq(t, 401, w.Code)
}

func TestEcho_BasicAuth_Correct(t *testing.T) {
	s := NewEchoServer()
	req := httptest.NewRequest("GET", "/basic-auth/user/pass", nil)
	req.SetBasicAuth("user", "pass")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), `"authenticated"`))
}

func TestEcho_Bytes(t *testing.T) {
	w := mockEcho(t, "GET", "/bytes/100", nil)
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, 100, w.Body.Len())
	assert.Eq(t, "application/octet-stream", w.Header().Get("Content-Type"))
}

func TestEcho_Bytes_CappedAt100KB(t *testing.T) {
	w := mockEcho(t, "GET", "/bytes/1000000", nil) // requests 1 MB
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, 100*1024, w.Body.Len())
}

func TestEcho_Bytes_NegativeIsZero(t *testing.T) {
	w := mockEcho(t, "GET", "/bytes/-5", nil)
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, 0, w.Body.Len())
}

func TestEcho_UUID(t *testing.T) {
	w := mockEcho(t, "GET", "/uuid", nil)
	assert.Eq(t, 200, w.Code)
	body := w.Body.String()
	assert.True(t, strings.Contains(body, `"uuid"`))
	pattern := `[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`
	assert.True(t, regexp.MustCompile(pattern).MatchString(body))
}

func TestMountEchoRoutes(t *testing.T) {
	r := rux.New()
	r.Group("/debug", func() {
		MountEchoRoutes(r)
	})
	req := httptest.NewRequest("GET", "/debug/anything", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
}
