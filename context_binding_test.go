package rux

import (
	"testing"

	"github.com/gookit/rux/binding"
	"github.com/stretchr/testify/assert"
)

type User struct {
	Age  int
	Name string
}

func TestContext_ShouldBind(t *testing.T) {
	is := assert.New(t)
	r := New()
	r.GET("/", func(c *Context) {
		u := &User{}

		err := c.ShouldBind(u, binding.JSON)
		is.NoError(err)
	})
}
