package core

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/rux/v2/pkg/render"
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

// ----- helpers ----------------------------------------------------

// renderCtx wires a Context against a fresh recorder + request — used by
// render tests that don't need the full ServeHTTP plumbing.
func renderCtx(t *testing.T, method, target string) (*Context, *httptest.ResponseRecorder) {
	t.Helper()
	c := &Context{}
	w := httptest.NewRecorder()
	c.Init(w, httptest.NewRequest(method, target, nil))
	return c, w
}

// stubRenderer captures the call without doing anything expensive.
type stubRenderer struct {
	called bool
	err    error
	last   any
}

func (s *stubRenderer) Render(w http.ResponseWriter, obj any) error {
	s.called = true
	s.last = obj
	if s.err != nil {
		return s.err
	}
	_, _ = w.Write([]byte("stub"))
	return nil
}

// stubViewRenderer satisfies the Renderer interface declared on Context.
type stubViewRenderer struct {
	called bool
	err    error
}

func (s *stubViewRenderer) Render(w io.Writer, name string, data any, c *Context) error {
	s.called = true
	if s.err != nil {
		return s.err
	}
	_, _ = io.WriteString(w, "<h1>"+name+"</h1>")
	return nil
}

// ----- Render / ShouldRender / MustRender -------------------------

func TestContext_Render_NoRendererReturnsErr(t *testing.T) {
	c, _ := renderCtx(t, "GET", "/x")
	err := c.Render(200, "home.tpl", nil)
	assert.Err(t, err)
	assert.True(t, strings.Contains(err.Error(), "renderer not registered"))
}

func TestContext_Render_WithRenderer(t *testing.T) {
	c, w := renderCtx(t, "GET", "/x")
	c.Renderer = &stubViewRenderer{}
	assert.NoErr(t, c.Render(200, "home.tpl", nil))
	assert.True(t, strings.Contains(w.Body.String(), "home.tpl"))
}

func TestContext_Render_PropagatesRendererErr(t *testing.T) {
	c, _ := renderCtx(t, "GET", "/x")
	c.Renderer = &stubViewRenderer{err: errors.New("boom")}
	assert.Err(t, c.Render(200, "home.tpl", nil))
}

func TestContext_ShouldRender_PassesThrough(t *testing.T) {
	c, w := renderCtx(t, "GET", "/x")
	s := &stubRenderer{}
	assert.NoErr(t, c.ShouldRender(201, map[string]int{"a": 1}, s))
	assert.True(t, s.called)
	assert.Eq(t, "stub", w.Body.String())
	assert.Eq(t, 201, c.StatusCode())
}

func TestContext_MustRender_NoErrorJustWrites(t *testing.T) {
	c, _ := renderCtx(t, "GET", "/x")
	c.MustRender(200, "x", &stubRenderer{})
	assert.NoErr(t, c.Err())
}

func TestContext_MustRender_RecordsErr(t *testing.T) {
	c, _ := renderCtx(t, "GET", "/x")
	c.MustRender(200, "x", &stubRenderer{err: errors.New("boom")})
	assert.NotNil(t, c.Err())
}

// ----- HTTPError / Back / HTMLString / Stream / JSONBytes / XML / JSONP

func TestContext_HTTPError(t *testing.T) {
	c, w := renderCtx(t, "GET", "/x")
	c.HTTPError("forbidden", 403)
	assert.Eq(t, 403, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "forbidden"))
}

func TestContext_Back_RedirectsToReferer(t *testing.T) {
	c, w := renderCtx(t, "GET", "/x")
	c.Req.Header.Set("Referer", "/origin")
	c.Back()
	assert.Eq(t, 302, w.Code)
	assert.Eq(t, "/origin", w.Header().Get("Location"))
}

func TestContext_HTMLString(t *testing.T) {
	c, w := renderCtx(t, "GET", "/x")
	c.HTMLString(200, "<p>hi</p>")
	assert.Eq(t, "<p>hi</p>", w.Body.String())
	assert.True(t, strings.Contains(w.Header().Get("Content-Type"), "html"))
}

func TestContext_Stream(t *testing.T) {
	c, w := renderCtx(t, "GET", "/x")
	src := strings.NewReader("streamed")
	c.Stream(200, "text/plain", src)
	assert.Eq(t, "streamed", w.Body.String())
}

func TestContext_JSONBytes(t *testing.T) {
	c, w := renderCtx(t, "GET", "/x")
	c.JSONBytes(200, []byte(`{"k":"v"}`))
	assert.Eq(t, `{"k":"v"}`, w.Body.String())
	assert.True(t, strings.Contains(w.Header().Get("Content-Type"), "json"))
}

func TestContext_XML(t *testing.T) {
	c, w := renderCtx(t, "GET", "/x")
	type doc struct {
		V string `xml:"v"`
	}
	c.XML(200, doc{V: "x"})
	body := w.Body.String()
	assert.True(t, strings.Contains(body, "<v>x</v>"))
}

func TestContext_XML_WithIndent(t *testing.T) {
	c, w := renderCtx(t, "GET", "/x")
	type doc struct {
		V string `xml:"v"`
	}
	c.XML(200, doc{V: "x"}, "  ")
	assert.True(t, strings.Contains(w.Body.String(), "\n"))
}

func TestContext_JSONP(t *testing.T) {
	c, w := renderCtx(t, "GET", "/x")
	c.JSONP(200, "cb", map[string]int{"a": 1})
	body := w.Body.String()
	assert.True(t, strings.HasPrefix(body, "cb("))
	assert.True(t, strings.HasSuffix(body, ");"))
}

// ----- File / FileContent / Attachment / Inline / Binary ---------

// writeTempFile drops content into t.TempDir() and returns the path.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	assert.NoErr(t, os.WriteFile(p, []byte(content), 0644))
	return p
}

func TestContext_File(t *testing.T) {
	p := writeTempFile(t, "served.txt", "hello-from-file")
	c, w := renderCtx(t, "GET", "/served.txt")
	c.File(p)
	assert.Eq(t, "hello-from-file", w.Body.String())
}

func TestContext_FileContent_KnownName(t *testing.T) {
	p := writeTempFile(t, "doc.txt", "content")
	c, w := renderCtx(t, "GET", "/doc.txt")
	c.FileContent(p)
	assert.Eq(t, "content", w.Body.String())
	// setRawContentHeader fired as part of FileContent.
	assert.Eq(t, "must-revalidate", w.Header().Get("Cache-Control"))
}

func TestContext_FileContent_ExplicitName(t *testing.T) {
	p := writeTempFile(t, "doc.txt", "x")
	c, w := renderCtx(t, "GET", "/doc.txt")
	c.FileContent(p, "renamed.txt")
	assert.Eq(t, "x", w.Body.String())
}

func TestContext_FileContent_MissingFile_500(t *testing.T) {
	c, w := renderCtx(t, "GET", "/nope.txt")
	c.FileContent("/no/such/path/nope.txt")
	assert.Eq(t, 500, w.Code)
}

func TestContext_Attachment(t *testing.T) {
	p := writeTempFile(t, "a.bin", "payload")
	c, w := renderCtx(t, "GET", "/a.bin")
	c.Attachment(p, "download.bin")
	disp := w.Header().Get("Content-Disposition")
	assert.True(t, strings.HasPrefix(disp, "attachment;"))
	assert.True(t, strings.Contains(disp, "download.bin"))
}

func TestContext_Inline(t *testing.T) {
	p := writeTempFile(t, "b.txt", "shown")
	c, w := renderCtx(t, "GET", "/b.txt")
	c.Inline(p, "view.txt")
	disp := w.Header().Get("Content-Disposition")
	assert.True(t, strings.HasPrefix(disp, "inline;"))
}

func TestContext_Binary(t *testing.T) {
	c, w := renderCtx(t, "GET", "/b.bin")
	c.Binary(200, bytes.NewReader([]byte("rawbytes")), "out.bin", false)
	assert.Eq(t, "rawbytes", w.Body.String())
	disp := w.Header().Get("Content-Disposition")
	assert.True(t, strings.HasPrefix(disp, "attachment;"))
}

func TestContext_Binary_Inline(t *testing.T) {
	c, w := renderCtx(t, "GET", "/b.bin")
	c.Binary(200, bytes.NewReader([]byte("x")), "view.bin", true)
	disp := w.Header().Get("Content-Disposition")
	assert.True(t, strings.HasPrefix(disp, "inline;"))
}

// ----- pkg/render guard: a passthrough renderer composes with Context

func TestContext_Respond_UsesRendererArg(t *testing.T) {
	c, w := renderCtx(t, "GET", "/x")
	c.Respond(202, map[string]string{"k": "v"}, render.JSONRenderer{})
	assert.Eq(t, 202, c.StatusCode())
	assert.True(t, strings.Contains(w.Body.String(), `"k":"v"`))
}
