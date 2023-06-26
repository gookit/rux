package render

import (
	"net/http"

	"github.com/gookit/goutil/netutil/httpctype"
)

// ViewRenderer for response HTML contents to client
type ViewRenderer struct {
	TplName string
}

// Render template to client
func (r ViewRenderer) Render(w http.ResponseWriter, obj any) (err error) {
	writeContentType(w, httpctype.HTML)
	return
}
