package rux

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCachedRoutes(t *testing.T) {
	is := assert.New(t)
	c := NewCachedRoutes(3)

	c.Set("cache1", NewRoute("/cache1", nil))
	c.Set("cache2", NewRoute("/cache2", nil))
	c.Set("cache3", NewRoute("/cache3", nil))
	c.Set("cache4", NewRoute("/cache4", nil))

	is.Equal(c.list.Front().Value.(*cacheNode).Key, "cache4")

	c.Get("cache3")

	is.Equal(c.list.Front().Value.(*cacheNode).Key, "cache3")
	is.Equal(3, c.Len())
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
