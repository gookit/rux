package rux

import (
	"github.com/gookit/goutil"
	"github.com/gookit/rux/pkg/binding"
)

// ShouldBind bind request data to a struct, will auto call validator
//
// Usage:
//
//	err := c.ShouldBind(&user, binding.JSON)
func (c *Context) ShouldBind(obj any, binder binding.Binder) error {
	return binder.Bind(c.Req, obj)
}

// MustBind bind request data to a struct, will auto call validator
//
// Usage:
//
//	c.MustBind(&user, binding.Json)
func (c *Context) MustBind(obj any, binder binding.Binder) {
	goutil.PanicErr(binder.Bind(c.Req, obj))
}

// AutoBind auto bind request data to a struct, will auto select binding.Binder by content-type
//
// Usage:
//
//	err := c.AutoBind(&user)
func (c *Context) AutoBind(obj any) error {
	return binding.Auto(c.Req, obj)
}

// Bind auto bind request data to a struct, will auto select binding.Binder by content-type.
// Alias method of the AutoBind()
//
// Usage:
//
//	err := c.Bind(&user)
func (c *Context) Bind(obj any) error {
	return binding.Auto(c.Req, obj)
}

// Validate input struct or map data. should call Bind() before validate.
//
// Recommended use ShouldBind() instead, it will binding and validate.
func (c *Context) Validate(obj any) error {
	return binding.Validate(obj)
}

/*************************************************************
 * quick context data binding
 *************************************************************/

// BindForm request data to an struct, will auto call validator
//
// Usage:
//
//	err := c.BindForm(&user)
func (c *Context) BindForm(obj any) error {
	return binding.Form.Bind(c.Req, obj)
}

// BindJSON request data to an struct, will auto call validator
//
// Usage:
//
//	err := c.BindJSON(&user)
func (c *Context) BindJSON(obj any) error {
	return binding.JSON.Bind(c.Req, obj)
}

// BindXML request data to an struct, will auto call validator
//
// Usage:
//
//	err := c.BindXML(&user)
func (c *Context) BindXML(obj any) error {
	return binding.XML.Bind(c.Req, obj)
}
