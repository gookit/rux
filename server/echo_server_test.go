package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
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

func TestEcho_MethodRouting_ExplicitVerbWins(t *testing.T) {
	// POST /post matches the explicit POST handler.
	w := mockEcho(t, "POST", "/post", strings.NewReader("hello"))
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), `"method": "POST"`))
}

func TestEcho_CatchAll_UnknownPath(t *testing.T) {
	w := mockEcho(t, "GET", "/totally-random-path", nil)
	assert.Eq(t, 200, w.Code)
	body := w.Body.String()
	assert.True(t, strings.Contains(body, `"method": "GET"`))
	assert.True(t, strings.Contains(body, "/totally-random-path"))
}

func TestEcho_MethodLocked_WrongVerbReturns405(t *testing.T) {
	// Method-locked endpoints return 405 with an Allow header instead of
	// falling through to /*path. Mirrors httpbin.
	cases := []struct {
		method, path, allow string
	}{
		{"GET", "/post", "POST"},
		{"GET", "/put", "PUT"},
		{"POST", "/get", "GET"},
		{"PUT", "/delete", "DELETE"},
		{"DELETE", "/patch", "PATCH"},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			w := mockEcho(t, c.method, c.path, nil)
			assert.Eq(t, 405, w.Code)
			assert.Eq(t, c.allow, w.Header().Get("Allow"))
		})
	}
}

func TestEcho_CatchAll_AnyVerb(t *testing.T) {
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		w := mockEcho(t, method, "/no-such-endpoint", nil)
		assert.Eq(t, 200, w.Code, "method=%s", method)
	}
}

func TestEcho_SpecificRoutes_BeatCatchAll(t *testing.T) {
	// Static routes beat root wildcard (rux P-2 priority).
	w := mockEcho(t, "GET", "/status/418", nil)
	assert.Eq(t, 418, w.Code)

	w = mockEcho(t, "GET", "/uuid", nil)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), `"uuid"`))

	// /anything/foo should hit /anything/*path (its own wildcard), not
	// the root /*path catch-all.
	w = mockEcho(t, "GET", "/anything/foo", nil)
	assert.Eq(t, 200, w.Code)
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

func TestEcho_Download_DefaultBin(t *testing.T) {
	w := mockEcho(t, "GET", "/download/hello.bin", nil)
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "application/octet-stream", w.Header().Get("Content-Type"))
	assert.True(t, strings.Contains(w.Header().Get("Content-Disposition"), "attachment"))
	assert.True(t, strings.Contains(w.Header().Get("Content-Disposition"), `"hello.bin"`))
	assert.Eq(t, 1024, w.Body.Len()) // default size
}

func TestEcho_Download_SizeOverride(t *testing.T) {
	w := mockEcho(t, "GET", "/download/x.bin?size=256", nil)
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, 256, w.Body.Len())
}

func TestEcho_Download_SizeCapped(t *testing.T) {
	w := mockEcho(t, "GET", "/download/x.bin?size=10000000", nil)
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, 100*1024, w.Body.Len()) // capped at maxBytes
}

func TestEcho_Download_TypeText(t *testing.T) {
	w := mockEcho(t, "GET", "/download/poem.txt?type=text&size=128", nil)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Header().Get("Content-Type"), "text/plain"))
	assert.Eq(t, 128, w.Body.Len())
	// Pattern text should be human-readable ASCII.
	assert.True(t, strings.Contains(w.Body.String(), "quick brown fox"))
}

func TestEcho_Download_TypeJSON(t *testing.T) {
	w := mockEcho(t, "GET", "/download/meta.json?type=json&size=256", nil)
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Header().Get("Content-Type"), "application/json"))
	body := w.Body.String()
	assert.True(t, strings.Contains(body, `"filename":"meta.json"`))
	assert.True(t, strings.Contains(body, `"generated_at"`))
}

func TestEcho_Download_Inline(t *testing.T) {
	w := mockEcho(t, "GET", "/download/show.bin?inline=1", nil)
	assert.Eq(t, 200, w.Code)
	disp := w.Header().Get("Content-Disposition")
	assert.True(t, strings.HasPrefix(disp, "inline;"))
}

func TestEcho_Download_NegativeSize(t *testing.T) {
	w := mockEcho(t, "GET", "/download/x.bin?size=-5", nil)
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, 0, w.Body.Len())
}

// buildMultipart creates a request body with one or more files and
// optional form fields. Returns the body, the multipart Content-Type,
// and the sha256 of each file payload (in input order) for verification.
func buildMultipart(t *testing.T, files []struct {
	Field, Filename, Content string
}, formFields map[string]string) (*bytes.Buffer, string, []string) {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	sums := make([]string, 0, len(files))
	for _, f := range files {
		fw, err := mw.CreateFormFile(f.Field, f.Filename)
		assert.NoErr(t, err)
		_, err = fw.Write([]byte(f.Content))
		assert.NoErr(t, err)
		h := sha256.Sum256([]byte(f.Content))
		sums = append(sums, hex.EncodeToString(h[:]))
	}
	for k, v := range formFields {
		assert.NoErr(t, mw.WriteField(k, v))
	}
	assert.NoErr(t, mw.Close())
	return body, mw.FormDataContentType(), sums
}

func TestEcho_Upload_SingleFile(t *testing.T) {
	body, ctype, sums := buildMultipart(t,
		[]struct{ Field, Filename, Content string }{
			{"file", "a.txt", "hello world"},
		}, nil)
	s := NewEchoServer()
	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", ctype)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
	out := w.Body.String()
	assert.True(t, strings.Contains(out, `"filename": "a.txt"`))
	assert.True(t, strings.Contains(out, `"size": 11`))
	assert.True(t, strings.Contains(out, sums[0]))
}

func TestEcho_Upload_MultiFile(t *testing.T) {
	body, ctype, sums := buildMultipart(t,
		[]struct{ Field, Filename, Content string }{
			{"f1", "a.txt", "alpha"},
			{"f2", "b.txt", "beta-content"},
		}, nil)
	s := NewEchoServer()
	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", ctype)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
	out := w.Body.String()
	assert.True(t, strings.Contains(out, "a.txt"))
	assert.True(t, strings.Contains(out, "b.txt"))
	assert.True(t, strings.Contains(out, sums[0]))
	assert.True(t, strings.Contains(out, sums[1]))
}

func TestEcho_Upload_FormFieldsAlongsideFiles(t *testing.T) {
	body, ctype, _ := buildMultipart(t,
		[]struct{ Field, Filename, Content string }{
			{"file", "a.txt", "x"},
		},
		map[string]string{"note": "hi there", "tag": "v1"})
	s := NewEchoServer()
	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", ctype)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
	out := w.Body.String()
	assert.True(t, strings.Contains(out, `"note"`))
	assert.True(t, strings.Contains(out, "hi there"))
	assert.True(t, strings.Contains(out, "v1"))
}

func TestEcho_Upload_NoMultipart(t *testing.T) {
	// Plain body without multipart Content-Type → 400.
	s := NewEchoServer()
	req := httptest.NewRequest("POST", "/upload", strings.NewReader("not multipart"))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	assert.Eq(t, 400, w.Code)
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
