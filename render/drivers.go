package render

import (
	"io"
	"net/http"
)

type IRender interface {
	Render(io.Writer, interface{}) error
}

// IHttpRender interface for renderer handler
type IHttpRender interface {
	Render(w http.ResponseWriter, status int, data interface{}) error
}

type JSON struct {
	Data interface{}
}

func (d *JSON) Render()  {

}

type FormattedJSON struct {
	Data interface{}
}

func (d *FormattedJSON) Render()  {

}