package rux

import "github.com/gookit/rux/binding"

// ShouldBind bind request data to an struct, will auto call validator
//
// Usage:
//	err := c.ShouldBind(&user, binding.JSON)
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

// AutoBind auto bind request data to an struct, will auto select binding.Binder by content-type
//
// Usage:
//	err := c.AutoBind(&user)
func (c *Context) AutoBind(obj interface{}) error {
	return binding.Auto(c.Req, obj)
}

/*************************************************************
 * quick context data binding
 *************************************************************/

// BindJSON request data to an struct, will auto call validator
//
// Usage:
//	err := c.BindJSON(&user)
//
func (c *Context) BindJSON(obj interface{}) error {
	return binding.JSON.Bind(c.Req, obj)
}

// BindXML request data to an struct, will auto call validator
//
// Usage:
//	err := c.BindXML(&user)
//
func (c *Context) BindXML(obj interface{}) error {
	return binding.XML.Bind(c.Req, obj)
}
