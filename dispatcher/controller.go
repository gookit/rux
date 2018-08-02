package dispatcher

import "net/http"

// Controller
type Controller struct {
	Req *http.Request
	Res http.ResponseWriter
}
