package v2

import (
	"github.com/gookit/goutil"
	"github.com/gookit/rux/pkg/binding"
)

// ShouldBind binds request data to obj using the given binder.
//
// Usage:
//
//	err := c.ShouldBind(&user, binding.JSON)
func (c *Context) ShouldBind(obj any, binder binding.Binder) error {
	return binder.Bind(c.Req, obj)
}

// MustBind binds request data to obj using binder, panicking on error.
//
// Usage:
//
//	c.MustBind(&user, binding.JSON)
func (c *Context) MustBind(obj any, binder binding.Binder) {
	goutil.PanicErr(binder.Bind(c.Req, obj))
}

// AutoBind selects a binder based on Content-Type and binds request data.
//
// Usage:
//
//	err := c.AutoBind(&user)
func (c *Context) AutoBind(obj any) error {
	return binding.Auto(c.Req, obj)
}

// Bind is an alias for AutoBind — content-type drives binder selection.
//
// Usage:
//
//	err := c.Bind(&user)
func (c *Context) Bind(obj any) error {
	return binding.Auto(c.Req, obj)
}

// Validate runs the registered binding validator against obj.
//
// Recommended: call ShouldBind directly — it binds and validates.
func (c *Context) Validate(obj any) error {
	return binding.Validate(obj)
}

/*************************************************************
 * quick context data binding
 *************************************************************/

// BindForm binds form-encoded request data to obj.
func (c *Context) BindForm(obj any) error {
	return binding.Form.Bind(c.Req, obj)
}

// BindJSON binds the JSON request body to obj.
func (c *Context) BindJSON(obj any) error {
	return binding.JSON.Bind(c.Req, obj)
}

// BindXML binds the XML request body to obj.
func (c *Context) BindXML(obj any) error {
	return binding.XML.Bind(c.Req, obj)
}
