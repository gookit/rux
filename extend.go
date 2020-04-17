package rux

import (
	"bytes"
	"errors"
	"io"
)

// Binder interface
type Binder interface {
	Bind(i interface{}, c *Context) error
}

// Renderer interface
type Renderer interface {
	Render(io.Writer, string, interface{}, *Context) error
}

// Validator interface
type Validator interface {
	Validate(i interface{}) error
}

/*************************************************************
 * Context function extends
 *************************************************************/

// Bind context bind struct
func (c *Context) Bind(i interface{}) error {
	if c.router.Binder == nil {
		return errors.New("binder not registered")
	}

	return c.router.Binder.Bind(i, c)
}

// Render context template
func (c *Context) Render(status int, name string, data interface{}) (err error) {
	if c.router.Renderer == nil {
		return errors.New("renderer not registered")
	}

	var buf = new(bytes.Buffer)
	if err = c.router.Renderer.Render(buf, name, data, c); err != nil {
		return err
	}

	c.HTML(status, buf.Bytes())
	return
}

// Validate context validator
func (c *Context) Validate(i interface{}) error {
	if c.Router().Validator == nil {
		return errors.New("validator not registered")
	}

	return c.Router().Validator.Validate(i)
}
