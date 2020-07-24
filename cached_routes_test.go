package rux

import (
	"fmt"
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
	is.True(c.Has("cache1"))
	is.False(c.Has("not-exists"))
}

func TestCacheRoutes(t *testing.T) {
	is := assert.New(t)
	r := New(CachingWithNum(3))

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

	ok := c.Set("cache1", NewRoute("/cache1", nil))
	is.True(ok)
	is.Equal(1, c.Len())

	// repeat set same key
	ok = c.Set("cache1", NewRoute("/cache1", nil))
	is.True(ok)
	is.Equal(1, c.Len())

	// test delete elements
	cr := NewCachedRoutes(3)
	for i := 0; i < 5; i++ {
		key := fmt.Sprint("key", i)
		cr.Set(key, NewRoute("/"+key, emptyHandler))
	}

	is.Equal(3, cr.Len())
}

func TestCachedRoutes_Get(t *testing.T) {
	is := assert.New(t)
	c := NewCachedRoutes(3)

	ok := c.Set("cache1", NewRoute("/cache1", nil))

	if is.True(ok) {
		is.True(c.Has("cache1"))
		route, ok := c.Get("cache1")
		is.True(ok)
		is.Equal("/cache1", route.Path())
	}

	is.Nil(c.Get("not-exists"))
}
