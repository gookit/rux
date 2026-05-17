package render

import (
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

// These tests exercise the package-level *Status proxies in std.go.
// Each one is a thin forward to the default Responder, so we only need
// a smoke test per entry point plus a couple of edge cases.

// resetStd swaps the package-level std for the duration of a test so
// per-test mutations (Init, SetTemplateRenderer) don't bleed across.
func resetStd(t *testing.T) {
	t.Helper()
	orig := std
	t.Cleanup(func() { std = orig })
	std = New()
}

func TestStd_Default_ReturnsSharedInstance(t *testing.T) {
	assert.Same(t, std, Default())
}

func TestStd_Init_AppliesOption(t *testing.T) {
	resetStd(t)
	Init(func(o *Options) { o.JSONIndent = true })
	w := httptest.NewRecorder()
	assert.NoErr(t, JSONStatus(w, 200, map[string]int{"x": 1}))
	assert.True(t, strings.Contains(w.Body.String(), "\n"))
}

func TestStd_JSONStatus(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, JSONStatus(w, 201, map[string]int{"a": 1}))
	assert.Eq(t, 201, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), `"a":1`))
}

func TestStd_JSONPStatus(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, JSONPStatus(w, 200, "cb", map[string]int{"x": 1}))
	body := w.Body.String()
	assert.True(t, strings.HasPrefix(body, "cb("))
	assert.True(t, strings.HasSuffix(body, ");"))
}

func TestStd_XMLStatus(t *testing.T) {
	type doc struct {
		V string `xml:"v"`
	}
	w := httptest.NewRecorder()
	assert.NoErr(t, XMLStatus(w, 200, doc{V: "x"}))
	assert.True(t, strings.Contains(w.Body.String(), "<v>x</v>"))
}

func TestStd_TextStatus(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, TextStatus(w, 200, "ok"))
	assert.Eq(t, "ok", w.Body.String())
}

func TestStd_ContentStatus(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, ContentStatus(w, 200, []byte("data"), "image/png"))
	assert.Eq(t, "image/png", w.Header().Get("Content-Type"))
	assert.Eq(t, "data", w.Body.String())
}

func TestStd_EmptyStatus(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, EmptyStatus(w))
	assert.Eq(t, 204, w.Code)
	assert.Eq(t, 0, w.Body.Len())
}

func TestStd_HTMLStringStatus(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, HTMLStringStatus(w, 200, "<p>{{.}}</p>", "hi"))
	assert.Eq(t, "<p>hi</p>", w.Body.String())
}

func TestStd_HTMLTextStatus(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, HTMLTextStatus(w, 200, "<b>x</b>"))
	assert.Eq(t, "<b>x</b>", w.Body.String())
}

func TestStd_HTMLStatus_NoEngineErr(t *testing.T) {
	// HTMLStatus without a TemplateRenderer should bubble up the error
	// from Responder.HTML.
	resetStd(t)
	w := httptest.NewRecorder()
	err := HTMLStatus(w, 200, "home.tpl", nil)
	assert.Err(t, err)
}

func TestStd_HTMLStatus_WithEngine(t *testing.T) {
	resetStd(t)
	tpl := &stubTpl{}
	SetTemplateRenderer(tpl)

	w := httptest.NewRecorder()
	assert.NoErr(t, HTMLStatus(w, 200, "home.tpl", nil))
	assert.Eq(t, "home.tpl", tpl.last)
}

func TestStd_BinaryStatus(t *testing.T) {
	w := httptest.NewRecorder()
	src := strings.NewReader("payload")
	assert.NoErr(t, BinaryStatus(w, 200, src, "f.bin", false))
	assert.True(t, strings.Contains(w.Header().Get("Content-Disposition"), "attachment;"))
	assert.Eq(t, "payload", w.Body.String())
}

func TestStd_AutoStatus(t *testing.T) {
	resetStd(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	assert.NoErr(t, AutoStatus(w, req, map[string]string{"k": "v"}))
	assert.True(t, strings.Contains(w.Body.String(), `"k":"v"`))
}

func TestStd_LoadTemplate_NoEngineErr(t *testing.T) {
	resetStd(t)
	// No engine installed → expect error from both helpers.
	assert.Err(t, LoadTemplateGlob("*.tpl"))
	assert.Err(t, LoadTemplateFiles("a.tpl"))
}

func TestStd_LoadTemplate_Forwards(t *testing.T) {
	resetStd(t)
	tpl := &stubTpl{}
	SetTemplateRenderer(tpl)
	assert.NoErr(t, LoadTemplateGlob("views/*"))
	assert.NoErr(t, LoadTemplateFiles("a.tpl", "b.tpl"))
	assert.Eq(t, []string{"glob:views/*", "file:a.tpl", "file:b.tpl"}, tpl.loads)
}

func TestStd_LoadTemplate_PropagatesError(t *testing.T) {
	resetStd(t)
	SetTemplateRenderer(&stubTpl{loadErr: errors.New("boom")})
	assert.Err(t, LoadTemplateGlob("*"))
	assert.Err(t, LoadTemplateFiles("x"))
}

// stubTpl is a small in-test TemplateRenderer + TemplateLoader.
// Kept local (not exported) so it doesn't leak into other packages.
type stubTpl struct {
	last    string
	loads   []string
	loadErr error
}

func (s *stubTpl) Render(w io.Writer, name string, _ any, _ ...string) error {
	s.last = name
	_, err := io.WriteString(w, "<h1>"+name+"</h1>")
	return err
}

func (s *stubTpl) LoadGlob(pattern string) error {
	if s.loadErr != nil {
		return s.loadErr
	}
	s.loads = append(s.loads, "glob:"+pattern)
	return nil
}

func (s *stubTpl) LoadFiles(files ...string) error {
	if s.loadErr != nil {
		return s.loadErr
	}
	for _, f := range files {
		s.loads = append(s.loads, "file:"+f)
	}
	return nil
}
