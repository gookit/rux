package render

import "io"

// Renderer interface
type Renderer interface {
	Render(io.Writer, string, interface{}, *Context) error
}

