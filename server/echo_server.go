package server

import (
	"fmt"
	"math/rand/v2"
	"net/http"
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

	// Method-specific endpoints. We deliberately register only the matching
	// verb — clients using the wrong method get 404, which is the simplest
	// behavior without enabling the router-wide 405 option.
	r.GET("/get", echoHandler)
	r.POST("/post", echoHandler)
	r.PUT("/put", echoHandler)
	r.PATCH("/patch", echoHandler)
	r.DELETE("/delete", echoHandler)

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
	n := c.Params().Int("n")
	if n < 0 {
		n = 0
	}
	if n > maxBytes {
		n = maxBytes
	}
	c.Resp.Header().Set("Content-Type", "application/octet-stream")
	c.Resp.WriteHeader(http.StatusOK)
	if n == 0 {
		return
	}
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = byte(rand.Uint32())
	}
	_, _ = c.Resp.Write(buf)
}

// echoUUIDHandler returns a freshly generated RFC 4122 v4 UUID.
// Inline implementation avoids pulling in a UUID dependency.
func echoUUIDHandler(c *rux.Context) {
	var b [16]byte
	// Fill with 4 random uint32s — plenty of entropy for a debug UUID.
	for i := 0; i < 16; i += 4 {
		v := rand.Uint32()
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
