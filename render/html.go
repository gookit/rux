package render

import (
	"net/http"

	"github.com/gookit/goutil/netutil/httpctype"
)

// HTMLRenderer for response HTML contents to client
type HTMLRenderer struct {
	TplName string
}

// Render JSON to client
func (r HTMLRenderer) Render(w http.ResponseWriter, obj interface{}) (err error) {
	writeContentType(w, httpctype.HTML)
	return
}

