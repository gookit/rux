package rux

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCachedRoutes_SetAndGet(t *testing.T) {
	is := assert.New(t)
	c := NewCachedRoutes(3)

	cache1 := c.Set("cache1", NewRoute("/cache1", nil))
	is.True(cache1)

	cache2 := c.Set("cache2", NewRoute("/cache2", nil))
	is.True(cache2)

	cache3 := c.Set("cache3", NewRoute("/cache3", nil))
	is.True(cache3)

	cache4 := c.Set("cache4", NewRoute("/cache4", nil))
	is.True(cache4)

	is.Equal(c.list.Front().Value.(*cacheNode).Key, "cache4")

	is.NotNil(c.Get("cache3"))

	is.Equal(c.list.Front().Value.(*cacheNode).Key, "cache3")
	is.Equal(3, c.Len())

	c2 := NewCachedRoutes(3)
	c2.list = nil

	cache5 := c2.Set("cache5", NewRoute("/cache5", nil))
	is.False(cache5)

	is.Nil(c2.Get("not-found"))
}

func TestCachedRoutes_Delete(t *testing.T) {
	is := assert.New(t)
	c := NewCachedRoutes(3)

	c.Set("cache1", NewRoute("/cache1", nil))
	c.Delete("cache1")

	is.Equal(0, c.Len())

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
