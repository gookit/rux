package render

import "io"

// Renderer interface
type Renderer interface {
	Render(w io.Writer, obj interface{}) error
}

