package render

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gookit/goutil/netutil/httpctype"
)

// FallbackType for auto response
var FallbackType = httpctype.MIMEText

// Auto render data to response
func Auto(w http.ResponseWriter, r *http.Request, obj interface{}) (err error) {
	accepts := parseAccept(r.Header.Get("Accept"))

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

func responseText(w http.ResponseWriter, obj interface{}) error {
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

// from gin framework
func parseAccept(acceptHeader string) []string {
	if acceptHeader == "" {
		return []string{}
	}

	parts := strings.Split(acceptHeader, ",")
	outs := make([]string, 0, len(parts))

	for _, part := range parts {
		if part = strings.TrimSpace(strings.Split(part, ";")[0]); part != "" {
			outs = append(outs, part)
		}
	}
	return outs
}
