package rux

import (
	"bytes"
	"errors"
	"io"

	"github.com/gookit/rux/binding"
	"github.com/gookit/rux/render"
)

const (
	// ContentType header key
	ContentType = "Content-Type"
	// ContentBinary represents content type application/octet-stream
	ContentBinary = "application/octet-stream"

	// ContentDisposition describes contentDisposition
	ContentDisposition = "Content-Disposition"
	// describes content disposition type
	dispositionInline = "inline"
	// describes content disposition type
	dispositionAttachment = "attachment"
)

/*************************************************************
 * Extends interfaces definition
 *************************************************************/

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
// Deprecated
// please use ShouldBind(),
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

/*************************************************************
 * Context binding and response render(RECOMMENDED)
 *************************************************************/

// ShouldBind bind request data to an struct, will auto call validator
//
// Usage:
//	err := c.ShouldBind(u, binding.JSON)
func (c *Context) ShouldBind(obj interface{}, binder binding.Binder) error {
	return binder.Bind(c.Req, obj)
}

// MustBind bind request data to an struct, will auto call validator
//
// Usage:
//	c.MustBind(&user, binding.Json)
func (c *Context) MustBind(obj interface{}, binder binding.Binder) {
	err := binder.Bind(c.Req, obj)
	if err != nil {
		panic(err)
	}
}

// ShouldRender render and response to client
func (c *Context) ShouldRender(status int, obj interface{}, renderer render.Renderer) error {
	c.SetStatus(status)
	return renderer.Render(c.Resp, obj)
}

// MustRender render and response to client
func (c *Context) MustRender(status int, obj interface{}, renderer render.Renderer) {
	c.Respond(status, obj, renderer)
}

// Respond render and response to client
func (c *Context) Respond(status int, obj interface{}, renderer render.Renderer) {
	c.SetStatus(status)

	err := renderer.Render(c.Resp, obj)
	if err != nil {
		panic(err)
	}
}
