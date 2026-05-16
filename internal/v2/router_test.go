package v2

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestNewRouter_Defaults(t *testing.T) {
	r := New()
	assert.NotNil(t, r)
	assert.Eq(t, "default", r.Name)
	assert.False(t, r.Frozen())
}

func TestNewRouter_WithOptions(t *testing.T) {
	r := New(StrictLastSlash, HandleMethodNotAllowed)
	assert.True(t, r.strictLastSlash)
	assert.True(t, r.handleMethodNotAllowed)
}

/*************************************************************
 * Task 3.2: Add / verb shortcuts / Any / static-vs-dynamic
 *************************************************************/

func TestRouter_Add_Static(t *testing.T) {
	r := New()
	h := func(c *Context) {}
	route := r.Add("/users", h, GET)
	assert.NotNil(t, route)
	assert.Eq(t, "/users", route.Path())
	assert.Eq(t, []string{GET}, route.Methods())
	idx := methodIndex(GET)
	_, ok := r.staticRoutes[idx]["/users"]
	assert.True(t, ok)
}

func TestRouter_Add_Dynamic(t *testing.T) {
	r := New()
	h := func(c *Context) {}
	r.Add("/users/{id}", h, GET)
	idx := methodIndex(GET)
	assert.NotNil(t, r.dynamicTrees[idx])
	assert.Nil(t, r.staticRoutes[idx])
}

func TestRouter_GET(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) {})
	r.POST("/y", func(c *Context) {})
	assert.Eq(t, 2, r.counter)
}

func TestRouter_Any_RegistersAllMethods(t *testing.T) {
	r := New()
	r.Any("/wild", func(c *Context) {})
	for _, m := range []string{GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD, CONNECT, TRACE} {
		idx := methodIndex(m)
		_, ok := r.staticRoutes[idx]["/wild"]
		assert.True(t, ok, "method %s missing", m)
	}
}

func TestRouter_AddAfterFreeze_Panics(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) {})
	// Phase 4 will replace Freeze with full merge logic.
	r.frozen.Store(true)
	assert.Panics(t, func() {
		r.GET("/y", func(c *Context) {})
	})
}

/*************************************************************
 * Task 3.4: Optional segment expansion
 *************************************************************/

func TestRouter_OptionalSegment_ExpandsToTwoRoutes(t *testing.T) {
	r := New()
	h := func(c *Context) {}
	r.GET("/posts[/{id}]", h)

	idxGet := methodIndex(GET)
	_, hasStatic := r.staticRoutes[idxGet]["/posts"]
	assert.True(t, hasStatic, "/posts (without id) should be static")

	assert.NotNil(t, r.dynamicTrees[idxGet])
	var ps Params
	route, ok := r.dynamicTrees[idxGet].lookup("/posts/42", &ps)
	assert.True(t, ok)
	assert.NotNil(t, route)
	assert.Eq(t, "42", ps.Get("id"))
}

func TestRouter_OptionalSegment_InvalidPosition_Panics(t *testing.T) {
	r := New()
	assert.Panics(t, func() {
		r.GET("/posts[/{cat}]/{id}", func(c *Context) {})
	})
}
