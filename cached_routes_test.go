package rux

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCachedRoutes_SetAndGet(t *testing.T) {
	is := assert.New(t)

	r := New()

	test := r.GET("/users/{id}", func(c *Context) {

	})

	c := NewCachedRoutes()
	c.Set("test", test)

	is.Equal(c.Get("test"), test)
}

func TestCachedRoutes_Delete(t *testing.T) {
	is := assert.New(t)

	r := New()

	test := r.GET("/users/{id}", func(c *Context) {

	})

	c := NewCachedRoutes()
	c.Set("test", test)
	c.Delete("test")

	is.Equal(c.Len(), 0)
}

func TestCachedRoutes_Has(t *testing.T) {
	is := assert.New(t)

	r := New()

	test := r.GET("/users/{id}", func(c *Context) {

	})

	c := NewCachedRoutes()
	c.Set("test", test)

	_, ok := c.Has("test")

	is.True(ok)
}

func TestCachedRoutes_Items(t *testing.T) {
	is := assert.New(t)

	r := New()

	test := r.GET("/users/{id}", func(c *Context) {

	})

	c := NewCachedRoutes()
	c.Set("test", test)

	for _, r := range c.Items() {
		is.Equal(r, test)
	}
}
