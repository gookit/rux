package render

import "io"

type Driver interface {
	Render(io.Writer, interface{}) error
}

type HtmlRenderer struct {
	template string
}

func (r *HtmlRenderer) Render(w io.Writer, data interface{}) error {

	return nil
}
