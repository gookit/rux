package rux

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCachedRoutes_SetAndGet(t *testing.T) {
	r := New()
	is := assert.New(t)

	test := r.GET("/users/{id}", func(c *Context) {

	})

	c := NewCachedRoutes(10)
	c.Set("test", test)

	is.Equal(test, c.Get("test"))
	is.Nil(c.Get("not-exist"))
}

func TestCachedRoutes_Delete(t *testing.T) {
	r := New()
	is := assert.New(t)

	test := r.GET("/users/{id}", func(c *Context) {

	})

	c := NewCachedRoutes(10)
	c.Set("test", test)
	c.Delete("test")

	is.Equal(c.Len(), 0)
}

func TestCachedRoutes_Has(t *testing.T) {
	r := New()
	is := assert.New(t)

	test := r.GET("/users/{id}", func(c *Context) {

	})

	c := NewCachedRoutes(10)
	c.Set("test", test)

	_, ok := c.Has("test")
	is.True(ok)
}

func TestCachedRoutes_Items(t *testing.T) {
	r := New()
	is := assert.New(t)

	test := r.GET("/users/{id}", func(c *Context) {})
	test1 := r.GET("/news/{id}", func(c *Context) {})

	c := NewCachedRoutes(10)
	c.Set("test", test)

	for _, r := range c.Items() {
		is.Equal(r, test)
	}

	is.False(c.Set("test", test))
	is.True(c.Set("test", test1))
	is.Equal(1, c.Len())
}
