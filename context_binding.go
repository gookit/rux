package rux

import "github.com/gookit/rux/binding"

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

/*************************************************************
 * context data binding
 *************************************************************/
