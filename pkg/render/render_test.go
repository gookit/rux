package render

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// These tests target the stateless package helpers in render.go and
// the Renderer interface implementations in json.go / xml.go that
// live alongside the Responder type.

func TestStateless_Text(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, Text(w, "hi"))
	assert.Eq(t, "hi", w.Body.String())
	assert.Eq(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestStateless_Plain_Alias(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, Plain(w, "p"))
	assert.Eq(t, "p", w.Body.String())
}

func TestStateless_TextBytes(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, TextBytes(w, []byte("bytes")))
	assert.Eq(t, "bytes", w.Body.String())
}

func TestStateless_HTML(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, HTML(w, "<b>x</b>"))
	assert.Eq(t, "<b>x</b>", w.Body.String())
	assert.Eq(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestStateless_HTMLBytes(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, HTMLBytes(w, []byte("<i>x</i>")))
	assert.Eq(t, "<i>x</i>", w.Body.String())
}

func TestStateless_Blob_EmptyDataSetsHeaderOnly(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, Blob(w, "image/png", nil))
	assert.Eq(t, "image/png", w.Header().Get("Content-Type"))
	assert.Eq(t, 0, w.Body.Len())
}

func TestStateless_Blob_PreservesExistingContentType(t *testing.T) {
	// writeContentType only fills the header if it isn't already set.
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "application/json")
	assert.NoErr(t, Blob(w, "text/plain", []byte("x")))
	assert.Eq(t, "application/json", w.Header().Get("Content-Type"))
}

func TestStateless_JSON(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, JSON(w, map[string]int{"a": 1}))
	assert.True(t, strings.Contains(w.Body.String(), `"a":1`))
	assert.Eq(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestStateless_JSONIndented(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, JSONIndented(w, map[string]int{"a": 1}))
	// Pretty output should contain a newline.
	assert.True(t, strings.Contains(w.Body.String(), "\n"))
}

func TestJSONRenderer_NotEscape_TogglesHTMLEscape(t *testing.T) {
	// Default encoder escapes "<" so "<a>" never appears verbatim in
	// the body; NotEscape preserves it. Easier to assert by comparing
	// the two modes' outputs than spelling out the Unicode escape here.
	w1 := httptest.NewRecorder()
	assert.NoErr(t, JSONRenderer{NotEscape: true}.Render(w1, map[string]string{"v": "<a>"}))
	raw := w1.Body.String()

	w2 := httptest.NewRecorder()
	assert.NoErr(t, JSONRenderer{}.Render(w2, map[string]string{"v": "<a>"}))
	escaped := w2.Body.String()

	assert.True(t, strings.Contains(raw, "<a>"))
	assert.False(t, strings.Contains(escaped, "<a>"))
	assert.Neq(t, raw, escaped)
}

func TestJSONP_PackageFn(t *testing.T) {
	w := httptest.NewRecorder()
	assert.NoErr(t, JSONP("cb", map[string]int{"x": 1}, w))
	body := w.Body.String()
	assert.True(t, strings.HasPrefix(body, "cb("))
	assert.True(t, strings.HasSuffix(body, ");"))
}

func TestXML_Stateless(t *testing.T) {
	type doc struct {
		XMLName xml.Name `xml:"d"`
		V       string   `xml:"v"`
	}
	w := httptest.NewRecorder()
	assert.NoErr(t, XML(w, doc{V: "x"}))
	body := w.Body.String()
	assert.True(t, strings.Contains(body, "<v>x</v>"))
	assert.True(t, strings.HasPrefix(body, xml.Header))
}

func TestXMLPretty_Indents(t *testing.T) {
	type doc struct {
		XMLName xml.Name `xml:"d"`
		V       string   `xml:"v"`
	}
	w := httptest.NewRecorder()
	assert.NoErr(t, XMLPretty(w, doc{V: "x"}))
	body := w.Body.String()
	assert.True(t, strings.Contains(body, "\n"))
}

func TestRendererFunc_AdaptsToInterface(t *testing.T) {
	called := false
	var r Renderer = RendererFunc(func(w http.ResponseWriter, _ any) error {
		called = true
		_, _ = w.Write([]byte("rf"))
		return nil
	})
	w := httptest.NewRecorder()
	assert.NoErr(t, r.Render(w, nil))
	assert.True(t, called)
	assert.Eq(t, "rf", w.Body.String())
}

func TestAuto_JSON(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/json")
	assert.NoErr(t, Auto(w, req, map[string]int{"a": 1}))
	assert.True(t, strings.Contains(w.Body.String(), `"a":1`))
}

func TestAuto_XML(t *testing.T) {
	type doc struct {
		XMLName xml.Name `xml:"d"`
		V       string   `xml:"v"`
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/xml")
	assert.NoErr(t, Auto(w, req, doc{V: "x"}))
	assert.True(t, strings.Contains(w.Body.String(), "<v>x</v>"))
}

func TestAuto_Text_String(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "text/plain")
	assert.NoErr(t, Auto(w, req, "hello"))
	assert.Eq(t, "hello", w.Body.String())
}

func TestAuto_Text_Bytes(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "text/plain")
	assert.NoErr(t, Auto(w, req, []byte("bytes!")))
	assert.Eq(t, "bytes!", w.Body.String())
}

func TestAuto_Text_FallbackJSONMarshal(t *testing.T) {
	// responseText falls through to json.Marshal for non-string types.
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "text/plain")
	assert.NoErr(t, Auto(w, req, map[string]int{"n": 7}))
	assert.True(t, strings.Contains(w.Body.String(), `"n":7`))
}

func TestAuto_FallbackTypeWhenNoAccept(t *testing.T) {
	// No Accept → falls back to FallbackType (text/plain).
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	assert.NoErr(t, Auto(w, req, "x"))
	assert.Eq(t, "x", w.Body.String())
}

func TestAuto_Unsupported_ReturnsError(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/protobuf")
	assert.Err(t, Auto(w, req, nil))
}
