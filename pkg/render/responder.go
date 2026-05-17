package render

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/gookit/goutil/netutil/httpctype"
	"github.com/gookit/goutil/netutil/httpreq"
)

// Response disposition values used in Content-Disposition headers.
const (
	ContentDisposition = "Content-Disposition"

	dispositionInline     = "inline"
	dispositionAttachment = "attachment"
)

// ContentType is a re-export of httpctype.Key kept for ergonomics.
const ContentType = httpctype.Key

// Options configure a Responder. All fields have working defaults — see
// New for the values.
type Options struct {
	// JSONIndent indents JSON output using PrettyIndent.
	JSONIndent bool
	// JSONPrefix is written before each JSON body. Useful for anti-hijack
	// payloads like ")]}',\n".
	JSONPrefix string

	// XMLIndent indents XML output using PrettyIndent.
	XMLIndent bool
	// XMLPrefix is written before each XML body. Defaults to xml.Header.
	XMLPrefix string

	// Charset appended to text-like content types when AddCharset is true.
	Charset string
	// AddCharset toggles charset suffixing on text-like content types.
	AddCharset bool

	// ContentType is the fallback used by Auto when the request supplies
	// neither Accept nor a response Content-Type.
	ContentType string

	// Per-format content types. Defaults to httpctype MIME constants
	// (no charset); the charset is appended at write time when AddCharset
	// is true. Override for custom MIME variants.
	ContentBinary string
	ContentHTML   string
	ContentXML    string
	ContentText   string
	ContentJSON   string
	ContentJSONP  string
}

// OptionFn mutates Options during New / Init.
type OptionFn func(*Options)

// Responder writes HTTP responses in common formats. It is safe for
// concurrent use after construction; do not mutate Options after New.
//
// HTML rendering is delegated to a pluggable TemplateRenderer, set via
// SetTemplateRenderer — Responder itself imports no template engine.
type Responder struct {
	opts *Options
	tpl  TemplateRenderer
}

// New returns a Responder with sensible defaults; pass OptionFn to override.
func New(fns ...OptionFn) *Responder {
	r := &Responder{
		opts: &Options{
			ContentBinary: httpctype.Binary,
			ContentHTML:   httpctype.MIMEHTML,
			ContentXML:    httpctype.MIMEXML,
			ContentText:   httpctype.MIMEText,
			ContentJSON:   httpctype.MIMEJSON,
			ContentJSONP:  "application/javascript",
			Charset:       "utf-8",
			AddCharset:    true,
			XMLPrefix:     xml.Header,
		},
	}
	for _, fn := range fns {
		fn(r.opts)
	}
	return r
}

// Options returns a copy of the current configuration (read-only).
func (r *Responder) Options() Options { return *r.opts }

// SetTemplateRenderer installs the engine used by HTML(). Passing nil
// disables HTML(name, data) — HTMLString and HTMLText remain usable.
func (r *Responder) SetTemplateRenderer(t TemplateRenderer) { r.tpl = t }

// TemplateRenderer returns the currently configured engine (or nil).
func (r *Responder) TemplateRenderer() TemplateRenderer { return r.tpl }

// LoadTemplateGlob forwards to the configured TemplateRenderer if it
// implements TemplateLoader.
func (r *Responder) LoadTemplateGlob(pattern string) error {
	loader, ok := r.tpl.(TemplateLoader)
	if !ok {
		return errors.New("render: TemplateRenderer does not implement TemplateLoader")
	}
	return loader.LoadGlob(pattern)
}

// LoadTemplateFiles forwards to the configured TemplateRenderer if it
// implements TemplateLoader.
func (r *Responder) LoadTemplateFiles(files ...string) error {
	loader, ok := r.tpl.(TemplateLoader)
	if !ok {
		return errors.New("render: TemplateRenderer does not implement TemplateLoader")
	}
	return loader.LoadFiles(files...)
}

// setContentType writes ct as the Content-Type header, optionally
// appending "; charset=…" for text-like payloads. Must be called before
// WriteHeader so http.ResponseWriter actually sends the header.
func (r *Responder) setContentType(w http.ResponseWriter, ct string, isText bool) {
	if isText && r.opts.AddCharset && r.opts.Charset != "" && !strings.Contains(ct, "charset=") {
		ct = ct + "; charset=" + r.opts.Charset
	}
	w.Header().Set(ContentType, ct)
}

// Empty is an alias for NoContent.
func (r *Responder) Empty(w http.ResponseWriter) error { return r.NoContent(w) }

// NoContent writes 204 with no body.
func (r *Responder) NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Content writes the given bytes with the given Content-Type and status.
func (r *Responder) Content(w http.ResponseWriter, status int, body []byte, contentType string) error {
	w.Header().Set(ContentType, contentType)
	w.WriteHeader(status)
	_, err := w.Write(body)
	return err
}

// Data is an alias for Content — kept for API symmetry with respond.
func (r *Responder) Data(w http.ResponseWriter, status int, body []byte, contentType string) error {
	return r.Content(w, status, body, contentType)
}

// Text writes a plain-text response.
func (r *Responder) Text(w http.ResponseWriter, status int, v string) error {
	r.setContentType(w, r.opts.ContentText, true)
	w.WriteHeader(status)
	_, err := io.WriteString(w, v)
	return err
}

// String is an alias for Text.
func (r *Responder) String(w http.ResponseWriter, status int, v string) error {
	return r.Text(w, status, v)
}

// JSON encodes v as JSON and writes the response.
func (r *Responder) JSON(w http.ResponseWriter, status int, v any) error {
	r.setContentType(w, r.opts.ContentJSON, true)
	w.WriteHeader(status)

	if r.opts.JSONPrefix != "" {
		if _, err := io.WriteString(w, r.opts.JSONPrefix); err != nil {
			return err
		}
	}
	enc := json.NewEncoder(w)
	if r.opts.JSONIndent {
		enc.SetIndent("", PrettyIndent)
	}
	return enc.Encode(v)
}

// JSONP wraps the JSON-encoded v in a JavaScript callback invocation.
func (r *Responder) JSONP(w http.ResponseWriter, status int, callback string, v any) error {
	if callback == "" {
		return errors.New("render: JSONP callback cannot be empty")
	}
	r.setContentType(w, r.opts.ContentJSONP, true)
	w.WriteHeader(status)

	if _, err := io.WriteString(w, callback+"("); err != nil {
		return err
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return err
	}
	_, err := io.WriteString(w, ");")
	return err
}

// XML encodes v as XML and writes the response, optionally prefixed
// with XMLPrefix (defaults to xml.Header).
func (r *Responder) XML(w http.ResponseWriter, status int, v any) error {
	r.setContentType(w, r.opts.ContentXML, true)
	w.WriteHeader(status)

	if r.opts.XMLPrefix != "" {
		if _, err := io.WriteString(w, r.opts.XMLPrefix); err != nil {
			return err
		}
	}
	enc := xml.NewEncoder(w)
	if r.opts.XMLIndent {
		enc.Indent("", PrettyIndent)
	}
	return enc.Encode(v)
}

// Binary streams in as an attachment (or inline when inline=true) with
// the given filename. Uses io.Copy so memory usage is bounded.
func (r *Responder) Binary(w http.ResponseWriter, status int, in io.Reader, outName string, inline bool) error {
	disp := dispositionAttachment
	if inline {
		disp = dispositionInline
	}
	w.Header().Set(ContentType, r.opts.ContentBinary)
	w.Header().Set(ContentDisposition, fmt.Sprintf("%s; filename=%q", disp, outName))
	w.WriteHeader(status)
	_, err := io.Copy(w, in)
	return err
}

// HTML renders the named template via the configured TemplateRenderer.
// Returns an error if no engine has been installed.
func (r *Responder) HTML(w http.ResponseWriter, status int, name string, data any, layout ...string) error {
	if r.tpl == nil {
		return errors.New("render: no TemplateRenderer configured; call SetTemplateRenderer or use HTMLString/HTMLText")
	}
	r.setContentType(w, r.opts.ContentHTML, true)
	w.WriteHeader(status)
	return r.tpl.Render(w, name, data, layout...)
}

// HTMLString parses and executes an inline template using std html/template.
// No external engine needed.
func (r *Responder) HTMLString(w http.ResponseWriter, status int, tplContent string, data any) error {
	t, err := template.New("inline").Parse(tplContent)
	if err != nil {
		return err
	}
	r.setContentType(w, r.opts.ContentHTML, true)
	w.WriteHeader(status)
	return t.Execute(w, data)
}

// HTMLText writes html as the response body without any templating.
func (r *Responder) HTMLText(w http.ResponseWriter, status int, html string) error {
	r.setContentType(w, r.opts.ContentHTML, true)
	w.WriteHeader(status)
	_, err := io.WriteString(w, html)
	return err
}

// Auto picks an output format based on the request Accept header.
// HTML responses require a non-empty tplName and a configured TemplateRenderer.
// Returns an error when none of the accepted types are supported.
func (r *Responder) Auto(w http.ResponseWriter, req *http.Request, data any, tplName ...string) error {
	accepts := httpreq.ParseAccept(req.Header.Get("Accept"))
	if len(accepts) == 0 {
		fallback := r.opts.ContentType
		if fallback == "" {
			fallback = FallbackType
		}
		accepts = []string{fallback}
	}

	for _, accept := range accepts {
		switch accept {
		case httpctype.MIMEJSON:
			return r.JSON(w, http.StatusOK, data)
		case httpctype.MIMEXML, httpctype.MIMEXML2:
			return r.XML(w, http.StatusOK, data)
		case httpctype.MIMEHTML:
			if len(tplName) == 0 || tplName[0] == "" {
				return errors.New("render: Auto needs a tplName to serve HTML")
			}
			return r.HTML(w, http.StatusOK, tplName[0], data)
		case httpctype.MIMEText:
			return r.Text(w, http.StatusOK, fmt.Sprint(data))
		}
	}
	return fmt.Errorf("render: no supported Accept type in %v", accepts)
}
