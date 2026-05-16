package v2

import (
	"strings"
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

/*************************************************************
 * Task 3.3: Group / Use / NotFound / NotAllowed
 *************************************************************/

func TestRouter_Group(t *testing.T) {
	r := New()
	r.Group("/api", func() {
		r.GET("/users", func(c *Context) {})
		r.GET("/posts", func(c *Context) {})
	})
	idx := methodIndex(GET)
	_, ok1 := r.staticRoutes[idx]["/api/users"]
	_, ok2 := r.staticRoutes[idx]["/api/posts"]
	assert.True(t, ok1)
	assert.True(t, ok2)
}

func TestRouter_GroupMiddleware_PrefixedToRouteChain(t *testing.T) {
	r := New()
	apiMW := func(c *Context) {}
	routeMW := func(c *Context) {}
	main := func(c *Context) {}

	r.Group("/api", func() {
		r.GET("/x", main, routeMW)
	}, apiMW)

	idx := methodIndex(GET)
	route := r.staticRoutes[idx]["/api/x"]
	assert.NotNil(t, route)
	// chain order should be [apiMW, routeMW, main]
	assert.Eq(t, 3, len(route.chain))
}

func TestRouter_Use_AddsToGlobalChain(t *testing.T) {
	r := New()
	r.Use(func(c *Context) {})
	assert.Eq(t, 1, len(r.globalChain))
}

func TestRouter_UseAfterRouteRegistration_Panics(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) {})
	assert.Panics(t, func() {
		r.Use(func(c *Context) {})
	})
}

func TestRouter_NotFound(t *testing.T) {
	r := New()
	r.NotFound(func(c *Context) {})
	assert.Eq(t, 1, len(r.noRoute))
}

/*************************************************************
 * Task 3.5: Controller / Resource
 *************************************************************/

type fakeController struct{}

func (f *fakeController) AddRoutes(g *Router) {
	g.GET("/", func(c *Context) {})
	g.POST("/", func(c *Context) {})
}

func TestRouter_Controller(t *testing.T) {
	r := New()
	r.Controller("/api", &fakeController{})

	idx := methodIndex(GET)
	_, ok := r.staticRoutes[idx]["/api"]
	assert.True(t, ok, "GET /api should be registered")

	idxPost := methodIndex(POST)
	_, ok = r.staticRoutes[idxPost]["/api"]
	assert.True(t, ok, "POST /api should be registered")
}

type fakeResource struct{}

func (f *fakeResource) Index(c *Context)  {}
func (f *fakeResource) Show(c *Context)   {}
func (f *fakeResource) Store(c *Context)  {}
func (f *fakeResource) Update(c *Context) {}
func (f *fakeResource) Delete(c *Context) {}

func TestRouter_Resource(t *testing.T) {
	r := New()
	r.Resource("/", &fakeResource{})

	paths := make(map[string]bool)
	for _, ri := range r.Routes() {
		paths[ri.Path] = true
	}
	assert.True(t, paths["/fakeresource"])
}

/*************************************************************
 * Task 3.7: Inspection
 *************************************************************/

func TestRouter_GetRoute_NamedRoute(t *testing.T) {
	r := New()
	r.AddNamed("user_show", "/users/{id}", func(c *Context) {}, GET)
	rt := r.GetRoute("user_show")
	assert.NotNil(t, rt)
	// Path is converted to colon form at registration.
	assert.Eq(t, "/users/:id", rt.Path())
}

func TestRouter_IterateRoutes(t *testing.T) {
	r := New()
	r.GET("/a", func(c *Context) {})
	r.POST("/b", func(c *Context) {})

	var paths []string
	r.IterateRoutes(func(route *Route) {
		paths = append(paths, route.Path())
	})
	assert.Eq(t, 2, len(paths))
	assert.Eq(t, "/a", paths[0])
	assert.Eq(t, "/b", paths[1])
}

func TestRouter_String_ContainsCount(t *testing.T) {
	r := New()
	r.GET("/a", func(c *Context) {})
	s := r.String()
	assert.True(t, strings.Contains(s, "Routes Count: 1"))
}

func TestRouter_Err(t *testing.T) {
	r := New()
	assert.NoErr(t, r.Err())
}
