package render

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gookit/goutil/netutil/httpctype"
	"github.com/gookit/goutil/netutil/httpreq"
)

var (
	// PrettyIndent indent string for render JSON or XML
	PrettyIndent = "  "

	// FallbackType for auto response
	FallbackType = httpctype.MIMEText
)

// Renderer interface
type Renderer interface {
	Render(w http.ResponseWriter, obj any) error
}

// RendererFunc definition
type RendererFunc func(w http.ResponseWriter, obj any) error

// Render to http.ResponseWriter
func (fn RendererFunc) Render(w http.ResponseWriter, obj any) error {
	return fn(w, obj)
}

// Text writes out a string as plain text.
func Text(w http.ResponseWriter, str string) error {
	return Blob(w, httpctype.Text, []byte(str))
}

// Plain writes out a string as plain text. alias of the Text()
func Plain(w http.ResponseWriter, str string) error {
	return Blob(w, httpctype.Text, []byte(str))
}

// TextBytes writes out a string as plain text.
func TextBytes(w http.ResponseWriter, data []byte) error {
	return Blob(w, httpctype.Text, data)
}

// HTML writes out as html text. if data is empty, only write headers
func HTML(w http.ResponseWriter, data string) error {
	return Blob(w, httpctype.HTML, []byte(data))
}

// HTMLBytes writes out as html text. if data is empty, only write headers
func HTMLBytes(w http.ResponseWriter, data []byte) error {
	return Blob(w, httpctype.HTML, data)
}

// Blob writes out []byte
func Blob(w http.ResponseWriter, contentType string, data []byte) (err error) {
	writeContentType(w, contentType)

	if len(data) > 0 {
		_, err = w.Write(data)
	}
	return
}

// Auto render data to response
func Auto(w http.ResponseWriter, r *http.Request, obj any) (err error) {
	accepts := httpreq.ParseAccept(r.Header.Get("Accept"))

	// fallback use FallbackType
	if len(accepts) == 0 {
		accepts = []string{FallbackType}
	}

	var handled bool
	// auto render response by Accept type.
	for _, accept := range accepts {
		switch accept {
		case httpctype.MIMEJSON:
			err = JSON(w, obj)
			handled = true
			break
		case httpctype.MIMEHTML:
			handled = true
			break
		case httpctype.MIMEText:
			err = responseText(w, obj)
			handled = true
			break
		case httpctype.MIMEXML:
		case httpctype.MIMEXML2:
			err = XML(w, obj)
			handled = true
			break
			// case httpctype.MIMEYAML:
			// 	break
		}

		if handled {
			break
		}
	}

	if !handled {
		return errors.New("not supported Accept type")
	}
	return
}

func responseText(w http.ResponseWriter, obj any) error {
	switch typVal := obj.(type) {
	case string:
		return Text(w, typVal)
	case []byte:
		return TextBytes(w, typVal)
	default:
		jsonBs, err := json.Marshal(obj)
		if err != nil {
			return err
		}

		return TextBytes(w, jsonBs)
	}
}

func writeContentType(w http.ResponseWriter, value string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		w.Header().Set("Content-Type", value)
	}
}
