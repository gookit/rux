package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/gookit/goutil/testutil"
	"github.com/gookit/rux/v2"
	"github.com/gookit/rux/v2/pkg/render"
)

// echoHomePage is a minimal HTML index that lists every endpoint mounted
// by MountEchoRoutes. Hard-coded so we don't need a template engine here.
const echoHomePage = `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>rux echo server</title></head>
<body>
<h1>rux echo server</h1>
<p>A httpbin-style HTTP echo / debug server.</p>
<h2>Endpoints</h2>
<ul>
  <li><code>ANY  /anything</code>           — echo the full request</li>
  <li><code>ANY  /anything/{path}</code>    — same, ignores trailing path</li>
  <li><code>GET  /get</code></li>
  <li><code>POST /post</code></li>
  <li><code>PUT  /put</code></li>
  <li><code>PATCH /patch</code></li>
  <li><code>DELETE /delete</code></li>
  <li><code>GET  /headers</code>            — return request headers only</li>
  <li><code>GET  /ip</code>                 — return client origin</li>
  <li><code>GET  /user-agent</code>         — return User-Agent</li>
  <li><code>ANY  /status/{code}</code>      — return that HTTP status</li>
  <li><code>GET  /delay/{seconds}</code>    — sleep N seconds (max 10)</li>
  <li><code>GET  /redirect/{n}</code>       — redirect N times then to /get</li>
  <li><code>GET  /cookies</code>            — return cookies as JSON</li>
  <li><code>GET  /cookies/set/{name}/{value}</code> — set cookie, 302 → /cookies</li>
  <li><code>GET  /basic-auth/{user}/{passwd}</code> — verify Basic Auth</li>
  <li><code>GET  /bytes/{n}</code>          — N random bytes (max 100KB)</li>
  <li><code>GET  /uuid</code>               — RFC 4122 v4 UUID</li>
  <li><code>GET  /download/{filename}</code> — auto-generate file (?size=N&amp;type=bin|text|json&amp;inline=1)</li>
  <li><code>POST /upload</code>             — multipart upload, echoes per-file sha256/size/mime</li>
  <li><code>ANY  /*path</code>              — echoes back any path</li>
</ul>
</body>
</html>`

// NewEchoServer builds a Server with httpbin-style echo endpoints pre-mounted.
// Use Server.Run() to start with graceful shutdown / lifecycle hooks.
func NewEchoServer() *Server {
	s := New(false)
	MountEchoRoutes(s.Router)
	return s
}

// MountEchoRoutes attaches all httpbin-style endpoints to an existing router.
// Use this when embedding echo functionality into a larger app (e.g. as a
// debug helper under /debug prefix via Router.Group).
func MountEchoRoutes(r *rux.Router) {
	// Home page lists available endpoints.
	r.GET("/", echoHomeHandler)

	// /anything mirrors the request as JSON.
	r.Any("/anything", echoHandler)
	r.Any("/anything/*path", echoHandler)

	// Method-locked endpoints. Registered as Any so a wrong method is
	// caught here (returning 405 with Allow header, httpbin-style) instead
	// of falling through to the root /*path catch-all and being echoed
	// as 200.
	registerMethodLocked(r, "/get", http.MethodGet)
	registerMethodLocked(r, "/post", http.MethodPost)
	registerMethodLocked(r, "/put", http.MethodPut)
	registerMethodLocked(r, "/patch", http.MethodPatch)
	registerMethodLocked(r, "/delete", http.MethodDelete)

	// Inspection endpoints.
	r.GET("/headers", echoHeadersHandler)
	r.GET("/ip", echoIPHandler)
	r.GET("/user-agent", echoUserAgentHandler)

	// Dynamic behavior.
	r.Any("/status/{code}", echoStatusHandler)
	r.GET("/delay/{seconds}", echoDelayHandler)
	r.GET("/redirect/{n}", echoRedirectHandler)

	// Cookies.
	r.GET("/cookies", echoCookiesHandler)
	r.GET("/cookies/set/{name}/{value}", echoCookiesSetHandler)

	// Auth.
	r.GET("/basic-auth/{user}/{passwd}", echoBasicAuthHandler)

	// Raw data.
	r.GET("/bytes/{n}", echoBytesHandler)
	r.GET("/uuid", echoUUIDHandler)

	// File transfer helpers — echo server never persists, so download
	// always synthesizes content on the fly and upload only hashes &
	// reports metadata back.
	r.GET("/download/{filename}", echoDownloadHandler)
	r.POST("/upload", echoUploadHandler)

	// Catch-all: any path not matched above is echoed back. rux v2's
	// routing priority is static > param > wildcard (P-2), so the
	// specific routes registered above always win — this only fires for
	// genuinely unhandled paths (e.g. GET /foo, POST /random).
	// Registered last for code-reading clarity; order doesn't affect lookup.
	r.Any("/*path", echoHandler)
}

// indentedJSON is the shared JSON renderer used by every echo handler so
// curl/httpie output stays human-readable.
var indentedJSON = render.NewJSONIndented()

// echoHomeHandler renders the static HTML index page.
func echoHomeHandler(c *rux.Context) {
	c.HTMLString(http.StatusOK, echoHomePage)
}

// echoHandler mirrors the incoming request as a JSON document.
func echoHandler(c *rux.Context) {
	reply := testutil.BuildEchoReply(c.Req)
	c.Respond(http.StatusOK, reply, indentedJSON)
}

// registerMethodLocked binds path to a handler that only responds to the
// given HTTP method; any other verb returns 405 with an Allow header.
// We register via Any so the wrong-method case is owned by this route
// rather than leaking down to the /*path catch-all.
func registerMethodLocked(r *rux.Router, path, method string) {
	r.Any(path, func(c *rux.Context) {
		if c.Req.Method != method {
			c.SetHeader("Allow", method)
			c.Resp.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		echoHandler(c)
	})
}

// echoHeadersHandler returns only the request headers section.
func echoHeadersHandler(c *rux.Context) {
	reply := testutil.BuildEchoReply(c.Req)
	c.Respond(http.StatusOK, rux.M{"headers": reply.Headers}, indentedJSON)
}

// echoIPHandler returns the best-effort client IP.
func echoIPHandler(c *rux.Context) {
	c.Respond(http.StatusOK, rux.M{"origin": c.ClientIP()}, indentedJSON)
}

// echoUserAgentHandler returns the User-Agent header value.
func echoUserAgentHandler(c *rux.Context) {
	c.Respond(http.StatusOK, rux.M{"user-agent": c.Req.UserAgent()}, indentedJSON)
}

// echoStatusHandler writes the requested status code with an empty body.
// Out-of-range values fall back to 200 to avoid panics from invalid input.
func echoStatusHandler(c *rux.Context) {
	code := c.Params().Int("code")
	if code < 100 || code > 599 {
		code = http.StatusOK
	}
	c.Resp.WriteHeader(code)
}

// echoDelayHandler sleeps up to 10 seconds, then echoes the request.
func echoDelayHandler(c *rux.Context) {
	n := c.Params().Int("seconds")
	if n < 0 {
		n = 0
	}
	if n > 10 {
		n = 10
	}
	if n > 0 {
		time.Sleep(time.Duration(n) * time.Second)
	}
	echoHandler(c)
}

// echoRedirectHandler implements an httpbin-style countdown redirect.
// /redirect/0 hops to /get; otherwise we 302 to /redirect/{n-1}.
func echoRedirectHandler(c *rux.Context) {
	n := c.Params().Int("n")
	if n <= 0 {
		c.Redirect("/get")
		return
	}
	if n > 30 {
		n = 30 // cap to keep clients from looping forever
	}
	c.Redirect(fmt.Sprintf("/redirect/%d", n-1))
}

// echoCookiesHandler returns the request cookies as a name→value map.
func echoCookiesHandler(c *rux.Context) {
	cookies := c.Req.Cookies()
	out := make(map[string]string, len(cookies))
	for _, ck := range cookies {
		out[ck.Name] = ck.Value
	}
	c.Respond(http.StatusOK, rux.M{"cookies": out}, indentedJSON)
}

// echoCookiesSetHandler sets a cookie then redirects to /cookies so the
// client immediately sees it echoed back.
func echoCookiesSetHandler(c *rux.Context) {
	name := c.Param("name")
	value := c.Param("value")
	c.SetCookie(name, value, 3600, "/", "", false, false)
	c.Redirect("/cookies")
}

// echoBasicAuthHandler challenges with WWW-Authenticate if creds are missing
// or wrong; on success it echoes the authenticated user.
func echoBasicAuthHandler(c *rux.Context) {
	wantUser := c.Param("user")
	wantPass := c.Param("passwd")
	gotUser, gotPass, ok := c.Req.BasicAuth()
	if !ok || gotUser != wantUser || gotPass != wantPass {
		c.SetHeader("WWW-Authenticate", `Basic realm="echo"`)
		c.Resp.WriteHeader(http.StatusUnauthorized)
		return
	}
	c.Respond(http.StatusOK,
		rux.M{"authenticated": true, "user": wantUser},
		indentedJSON)
}

// maxBytes caps the response size of /bytes/{n} so a single client can't
// scribble megabytes of garbage per request.
const maxBytes = 100 * 1024

// echoBytesHandler returns N pseudo-random bytes as application/octet-stream.
// Uses math/rand/v2 — output is deterministic-enough fake data, not crypto.
func echoBytesHandler(c *rux.Context) {
	buf := makeRandomBytes(c.Params().Int("n"))
	c.Resp.Header().Set("Content-Type", "application/octet-stream")
	c.Resp.WriteHeader(http.StatusOK)
	if len(buf) > 0 {
		_, _ = c.Resp.Write(buf)
	}
}

// makeRandomBytes returns n pseudo-random bytes, with n clamped to
// [0, maxBytes]. Uses math/rand/v2 — output is deterministic-enough
// fake data, not crypto.
func makeRandomBytes(n int) []byte {
	n = clampSize(n)
	if n == 0 {
		return nil
	}
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = byte(mrand.Uint32())
	}
	return buf
}

// echoUUIDHandler returns a freshly generated RFC 4122 v4 UUID.
// Inline implementation avoids pulling in a UUID dependency.
func echoUUIDHandler(c *rux.Context) {
	var b [16]byte
	// Fill with 4 random uint32s — plenty of entropy for a debug UUID.
	for i := 0; i < 16; i += 4 {
		v := mrand.Uint32()
		b[i] = byte(v)
		b[i+1] = byte(v >> 8)
		b[i+2] = byte(v >> 16)
		b[i+3] = byte(v >> 24)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	c.Respond(http.StatusOK, rux.M{"uuid": uuid}, indentedJSON)
}

// maxUpload caps total multipart body size accepted by /upload (32 MB).
// Mirrors net/http's default ParseMultipartForm budget.
const maxUpload = 32 << 20

// echoDownloadHandler synthesizes a download on demand. The echo server
// does not persist files, so the "file" never exists — we always generate
// content from query params: ?size=N (default 1024, capped at maxBytes),
// ?type=bin|text|json (default bin), ?inline=1 to render in browser.
func echoDownloadHandler(c *rux.Context) {
	name := c.Param("filename")
	if name == "" {
		name = "download.bin"
	}

	size := 1024
	if s := c.Query("size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			size = n
		}
	}
	if size < 0 {
		size = 0
	}
	if size > maxBytes {
		size = maxBytes
	}

	kind := c.Query("type")
	if kind == "" {
		kind = "bin"
	}
	inline := c.Query("inline") == "1"

	var body []byte
	var ctype string
	switch kind {
	case "text":
		ctype = "text/plain; charset=utf-8"
		body = makeTextBytes(size)
	case "json":
		ctype = "application/json; charset=utf-8"
		body = makeJSONBytes(name, size)
	default: // "bin" or unknown
		ctype = "application/octet-stream"
		body = makeRandomBytes(size)
	}

	disp := "attachment"
	if inline {
		disp = "inline"
	}
	h := c.Resp.Header()
	h.Set("Content-Type", ctype)
	h.Set("Content-Disposition", fmt.Sprintf(`%s; filename=%q`, disp, name))
	h.Set("Content-Length", strconv.Itoa(len(body)))
	c.Resp.WriteHeader(http.StatusOK)
	_, _ = c.Resp.Write(body)
}

// clampSize bounds a user-supplied size into [0, maxBytes]. Centralized so
// every make([]byte, n) site has a visible upper bound and CodeQL can prove
// the allocation cannot blow up regardless of how the helper is reached.
func clampSize(n int) int {
	if n <= 0 {
		return 0
	}
	if n > maxBytes {
		return maxBytes
	}
	return n
}

// makeTextBytes fills a buffer with a repeating ASCII pattern so the
// downloaded text is readable in any viewer.
func makeTextBytes(n int) []byte {
	n = clampSize(n)
	if n == 0 {
		return nil
	}
	const pattern = "The quick brown fox jumps over the lazy dog.\n"
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		buf[i] = pattern[i%len(pattern)]
	}
	return buf
}

// makeJSONBytes builds a JSON document describing the synthetic file.
// When size > the natural payload, the buffer is padded with spaces so
// the response still hits the requested size; when size is too small to
// fit the payload, we just return the payload (size becomes a floor, not
// a hard cap — matching httpbin's loose semantics for debug endpoints).
func makeJSONBytes(name string, size int) []byte {
	size = clampSize(size)
	payload := fmt.Sprintf(`{"filename":%q,"size":%d,"generated_at":%q}`,
		name, size, time.Now().UTC().Format(time.RFC3339))
	if size <= len(payload) {
		return []byte(payload)
	}
	buf := make([]byte, size)
	copy(buf, payload)
	for i := len(payload); i < size; i++ {
		buf[i] = ' '
	}
	return buf
}

// echoUploadHandler accepts multipart/form-data and echoes per-file
// metadata (size, MIME, sha256) plus any non-file form values.
// Files are streamed through sha256 and discarded — nothing touches disk.
func echoUploadHandler(c *rux.Context) {
	if err := c.Req.ParseMultipartForm(maxUpload); err != nil {
		c.Resp.WriteHeader(http.StatusBadRequest)
		_, _ = c.Resp.Write([]byte(err.Error()))
		return
	}

	type fileInfo struct {
		Field    string `json:"field"`
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
		MIME     string `json:"mime"`
		SHA256   string `json:"sha256"`
	}

	files := []fileInfo{}
	form := map[string][]string{}

	if mf := c.Req.MultipartForm; mf != nil {
		for field, fhs := range mf.File {
			for _, fh := range fhs {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				h := sha256.New()
				n, _ := io.Copy(h, f)
				_ = f.Close()
				files = append(files, fileInfo{
					Field:    field,
					Filename: fh.Filename,
					Size:     n,
					MIME:     fh.Header.Get("Content-Type"),
					SHA256:   hex.EncodeToString(h.Sum(nil)),
				})
			}
		}
		for k, v := range mf.Value {
			form[k] = v
		}
	}

	c.Respond(http.StatusOK, rux.M{
		"files": files,
		"form":  form,
	}, indentedJSON)
}
