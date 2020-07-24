package rux

import (
	"container/list"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCachedRoutes_Delete(t *testing.T) {
	is := assert.New(t)
	c := NewCachedRoutes(3)

	c.Set("cache1", NewRoute("/cache1", nil))
	c.Delete("cache1")

	is.Equal(0, c.Len())

	is.False(c.Delete("cache2"))

	c.hashMap = nil

	is.False(c.Delete("cache1"))
}

func TestCachedRoutes_Has(t *testing.T) {
	is := assert.New(t)
	c := NewCachedRoutes(3)

	c.Set("cache1", NewRoute("/cache1", nil))

	_, ok := c.Has("cache1")
	is.True(ok)
}

func TestCacheRoutes(t *testing.T) {
	is := assert.New(t)
	r := New(EnableCaching, MaxNumCaches(3))

	r.GET("/cache1/{id}", func(c *Context) {})
	r.GET("/cache2/{id}", func(c *Context) {})
	r.GET("/cache3/{id}", func(c *Context) {
		c.WriteString("cache3")
	})
	r.GET("/cache4/{id}", func(c *Context) {
		c.WriteString("cache4")
	})

	w1 := mockRequest(r, "GET", "/cache3/1234", nil)
	is.Equal("cache3", w1.Body.String())
	w2 := mockRequest(r, "GET", "/cache4/1234", nil)
	is.Equal("cache4", w2.Body.String())

	is.Equal(2, r.cachedRoutes.Len())
}

func TestCachedRoutes_Set(t *testing.T) {
	is := assert.New(t)
	c := NewCachedRoutes(3)

	cache1 := c.Set("cache1", NewRoute("/cache1", nil))
	is.True(cache1)

	is.Equal(1, c.Len())

	c2 := NewCachedRoutes(3)
	c2.list = nil

	cache5 := c2.Set("cache5", NewRoute("/cache5", nil))
	is.False(cache5)
}

func TestCachedRoutes_Get(t *testing.T) {
	is := assert.New(t)
	c := NewCachedRoutes(3)

	cache1 := c.Set("cache1", NewRoute("/cache1", nil))

	if cache1 {
		is.NotNil(c.Get("cache1"))
	}

	is.Nil(c.Get("not-found"))

	c.hashMap["error"] = &list.Element{
		Value: "error",
	}

	is.Nil(c.Get("error"))

	c.hashMap = nil

	is.Nil(c.Get("cache1"))
}
