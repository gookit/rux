package render

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// mockTpl implements TemplateRenderer + TemplateLoader for tests.
type mockTpl struct {
	called  string
	loaded  []string
	loadErr error
	renderErr error
}

func (m *mockTpl) Render(w io.Writer, name string, data any, layout ...string) error {
	if m.renderErr != nil {
		return m.renderErr
	}
	m.called = name
	_, err := io.WriteString(w, "<h1>"+name+"</h1>")
	return err
}

func (m *mockTpl) LoadGlob(pattern string) error {
	if m.loadErr != nil {
		return m.loadErr
	}
	m.loaded = append(m.loaded, "glob:"+pattern)
	return nil
}

func (m *mockTpl) LoadFiles(files ...string) error {
	if m.loadErr != nil {
		return m.loadErr
	}
	for _, f := range files {
		m.loaded = append(m.loaded, "file:"+f)
	}
	return nil
}

// onlyRender omits the optional Loader interface — used to verify
// LoadTemplateGlob / LoadTemplateFiles surface a clear error.
type onlyRender struct{}

func (onlyRender) Render(w io.Writer, name string, data any, layout ...string) error { return nil }

func TestResponder_Defaults(t *testing.T) {
	r := New()
	opts := r.Options()
	assert.True(t, opts.AddCharset)
	assert.Eq(t, "utf-8", opts.Charset)
	assert.Eq(t, "application/json", opts.ContentJSON)
	assert.Eq(t, "application/octet-stream", opts.ContentBinary)
}

func TestResponder_Text_AppendsCharset(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()
	assert.NoErr(t, r.Text(w, 201, "hello"))

	assert.Eq(t, 201, w.Code)
	assert.Eq(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Eq(t, "hello", w.Body.String())
}

func TestResponder_Text_NoCharsetWhenDisabled(t *testing.T) {
	r := New(func(o *Options) { o.AddCharset = false })
	w := httptest.NewRecorder()
	assert.NoErr(t, r.Text(w, 200, "hi"))
	assert.Eq(t, "text/plain", w.Header().Get("Content-Type"))
}

func TestResponder_JSON_Indented(t *testing.T) {
	r := New(func(o *Options) { o.JSONIndent = true })
	w := httptest.NewRecorder()
	assert.NoErr(t, r.JSON(w, 200, map[string]int{"n": 1}))
	body := w.Body.String()
	assert.True(t, strings.Contains(body, "\n"))
	assert.True(t, strings.Contains(body, "  \"n\""))
	assert.Eq(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestResponder_JSON_Prefix(t *testing.T) {
	r := New(func(o *Options) { o.JSONPrefix = ")]}',\n" })
	w := httptest.NewRecorder()
	assert.NoErr(t, r.JSON(w, 200, "x"))
	assert.True(t, strings.HasPrefix(w.Body.String(), ")]}',\n"))
}

func TestResponder_JSONP(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()
	assert.NoErr(t, r.JSONP(w, 200, "cb", map[string]string{"a": "b"}))
	body := w.Body.String()
	assert.True(t, strings.HasPrefix(body, "cb("))
	assert.True(t, strings.HasSuffix(body, ");"))
}

func TestResponder_JSONP_EmptyCallback(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()
	err := r.JSONP(w, 200, "", nil)
	assert.Err(t, err)
}

func TestResponder_XML(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()
	type Doc struct {
		XMLName struct{} `xml:"doc"`
		Name    string   `xml:"name"`
	}
	assert.NoErr(t, r.XML(w, 200, Doc{Name: "rux"}))
	body := w.Body.String()
	assert.True(t, strings.HasPrefix(body, `<?xml`))
	assert.True(t, strings.Contains(body, "<name>rux</name>"))
}

func TestResponder_Binary_Attachment(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()
	src := strings.NewReader("hello-bin")
	assert.NoErr(t, r.Binary(w, 200, src, "a.bin", false))

	disp := w.Header().Get("Content-Disposition")
	assert.True(t, strings.HasPrefix(disp, "attachment;"))
	assert.True(t, strings.Contains(disp, `"a.bin"`))
	assert.Eq(t, "application/octet-stream", w.Header().Get("Content-Type"))
	assert.Eq(t, "hello-bin", w.Body.String())
}

func TestResponder_Binary_Inline(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()
	assert.NoErr(t, r.Binary(w, 200, strings.NewReader("x"), "x.bin", true))
	assert.True(t, strings.HasPrefix(w.Header().Get("Content-Disposition"), "inline;"))
}

func TestResponder_Content_HeaderOrdering(t *testing.T) {
	// Regression: respond's Content() set the header AFTER WriteHeader,
	// which silently lost the Content-Type. Make sure we got the order right.
	r := New()
	w := httptest.NewRecorder()
	assert.NoErr(t, r.Content(w, 200, []byte("x"), "image/png"))
	assert.Eq(t, "image/png", w.Header().Get("Content-Type"))
}

func TestResponder_NoContent(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()
	assert.NoErr(t, r.NoContent(w))
	assert.Eq(t, 204, w.Code)
	assert.Eq(t, 0, w.Body.Len())
}

func TestResponder_HTML_RequiresEngine(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()
	err := r.HTML(w, 200, "home.tpl", nil)
	assert.Err(t, err)
}

func TestResponder_HTML_WithEngine(t *testing.T) {
	r := New()
	tpl := &mockTpl{}
	r.SetTemplateRenderer(tpl)

	w := httptest.NewRecorder()
	assert.NoErr(t, r.HTML(w, 200, "home.tpl", nil))
	assert.Eq(t, "home.tpl", tpl.called)
	assert.Eq(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	assert.True(t, strings.Contains(w.Body.String(), "home.tpl"))
}

func TestResponder_HTMLString_StdTemplate(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()
	assert.NoErr(t, r.HTMLString(w, 200, "<p>{{.Name}}</p>", map[string]string{"Name": "rux"}))
	assert.Eq(t, "<p>rux</p>", w.Body.String())
}

func TestResponder_HTMLText(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()
	assert.NoErr(t, r.HTMLText(w, 200, "<b>hi</b>"))
	assert.Eq(t, "<b>hi</b>", w.Body.String())
	assert.Eq(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestResponder_LoadTemplate_NoLoaderImpl(t *testing.T) {
	r := New()
	r.SetTemplateRenderer(onlyRender{})
	assert.Err(t, r.LoadTemplateGlob("*.tpl"))
	assert.Err(t, r.LoadTemplateFiles("a.tpl"))
}

func TestResponder_LoadTemplate_Forwarded(t *testing.T) {
	r := New()
	tpl := &mockTpl{}
	r.SetTemplateRenderer(tpl)

	assert.NoErr(t, r.LoadTemplateGlob("views/*"))
	assert.NoErr(t, r.LoadTemplateFiles("a.tpl", "b.tpl"))
	assert.Eq(t, []string{"glob:views/*", "file:a.tpl", "file:b.tpl"}, tpl.loaded)
}

func TestResponder_LoadTemplate_PropagatesError(t *testing.T) {
	r := New()
	r.SetTemplateRenderer(&mockTpl{loadErr: errors.New("boom")})
	assert.Err(t, r.LoadTemplateGlob("*"))
}

func TestResponder_Auto_JSON(t *testing.T) {
	r := New()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	assert.NoErr(t, r.Auto(w, req, map[string]string{"k": "v"}))
	assert.True(t, strings.Contains(w.Body.String(), `"k":"v"`))
}

func TestResponder_Auto_XML(t *testing.T) {
	r := New()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/xml")
	w := httptest.NewRecorder()
	type Doc struct {
		XMLName struct{} `xml:"d"`
		V       string   `xml:"v"`
	}
	assert.NoErr(t, r.Auto(w, req, Doc{V: "x"}))
	assert.True(t, strings.Contains(w.Body.String(), "<v>x</v>"))
}

func TestResponder_Auto_HTML_NoTplName(t *testing.T) {
	r := New()
	r.SetTemplateRenderer(&mockTpl{})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	assert.Err(t, r.Auto(w, req, nil))
}

func TestResponder_Auto_HTML_WithTplName(t *testing.T) {
	r := New()
	tpl := &mockTpl{}
	r.SetTemplateRenderer(tpl)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	assert.NoErr(t, r.Auto(w, req, nil, "p.tpl"))
	assert.Eq(t, "p.tpl", tpl.called)
}

func TestResponder_Auto_FallbackContentType(t *testing.T) {
	r := New(func(o *Options) { o.ContentType = "application/json" })
	req := httptest.NewRequest("GET", "/", nil) // no Accept
	w := httptest.NewRecorder()
	assert.NoErr(t, r.Auto(w, req, map[string]string{"a": "b"}))
	assert.Eq(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestPackageProxy_JSONStatus(t *testing.T) {
	// std must work without explicit New().
	w := httptest.NewRecorder()
	assert.NoErr(t, JSONStatus(w, 202, map[string]int{"x": 1}))
	assert.Eq(t, 202, w.Code)
	assert.Eq(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestPackageProxy_TextStatus(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, TextStatus(w, 200, "ok"))
	assert.Eq(t, "ok", w.Body.String())
}

func TestPackageProxy_HTMLTextStatus(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, HTMLTextStatus(w, 200, "<p>x</p>"))
	assert.Eq(t, "<p>x</p>", w.Body.String())
}

func TestPackageProxy_Default_RoundTrip(t *testing.T) {
	// Default() exposes std so callers can configure it.
	d := Default()
	assert.NotNil(t, d)
	// Default Charset should still be utf-8 on a freshly-loaded package.
	assert.Eq(t, "utf-8", d.Options().Charset)
}

// Sanity: make sure http.Handler integration shape works end-to-end.
func TestResponder_EndToEnd_HTTPHandler(t *testing.T) {
	r := New()
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = r.JSON(w, 200, map[string]string{"ok": "yes"})
	})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	assert.Eq(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), `"ok":"yes"`))
}
