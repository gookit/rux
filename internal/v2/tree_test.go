package v2

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestNewRadixTree(t *testing.T) {
	tree := newRadixTree()
	assert.NotNil(t, tree)
	assert.NotNil(t, tree.root)
	assert.Eq(t, nodeRoot, tree.root.nType)
	assert.Eq(t, "/", tree.root.prefix)
	assert.Nil(t, tree.root.chain)
	assert.Eq(t, uint8(0), tree.maxParams)
}

func TestNode_AddStaticChild(t *testing.T) {
	parent := &node{
		prefix: "/",
		nType:  nodeRoot,
	}
	parent.addStaticChild(&node{prefix: "users", nType: nodeStatic})
	assert.Eq(t, 1, len(parent.children))
	assert.Eq(t, byte('u'), parent.indices[0])
	assert.Eq(t, "users", parent.children[0].prefix)
}

func TestTreeInsert_SingleStaticPath(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	route := newRoute("/users", h, []string{GET})
	tree.insert("/users", route)

	assert.Eq(t, 1, len(tree.root.children))
	assert.Eq(t, byte('u'), tree.root.indices[0])
}

func TestTreeInsert_TwoSiblings(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	tree.insert("/users", newRoute("/users", h, []string{GET}))
	tree.insert("/posts", newRoute("/posts", h, []string{GET}))
	assert.Eq(t, 2, len(tree.root.children))
}

func TestTreeInsert_NodeSplit(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	tree.insert("/userprofile", newRoute("/userprofile", h, []string{GET}))
	tree.insert("/userlist", newRoute("/userlist", h, []string{GET}))
	// Root -> "user" (split point) -> {"profile", "list"}
	assert.Eq(t, 1, len(tree.root.children))
	parent := tree.root.children[0]
	assert.Eq(t, "user", parent.prefix)
	assert.Eq(t, 2, len(parent.children))
}

func TestTreeInsert_DuplicatePathPanics(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	tree.insert("/users", newRoute("/users", h, []string{GET}))
	assert.Panics(t, func() {
		tree.insert("/users", newRoute("/users", h, []string{GET}))
	})
}

func TestTreeInsert_ParamRoute(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	tree.insert("/users/:id", newRoute("/users/:id", h, []string{GET}))
	// Verify the route is reachable structurally — full lookup test in Task 2.2
	assert.Eq(t, uint8(1), tree.maxParams)
}

func TestTreeInsert_WildcardRoute(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	tree.insert("/files/*path", newRoute("/files/*path", h, []string{GET}))
	assert.Eq(t, uint8(1), tree.maxParams)
}

func TestTreeLookup_Static(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	r := newRoute("/users", h, []string{GET})
	tree.insert("/users", r)

	var ps Params
	got, ok := tree.lookup("/users", &ps)
	assert.True(t, ok)
	assert.Same(t, r, got)
	assert.Eq(t, 0, ps.Len())
}

func TestTreeLookup_Param(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	r := newRoute("/users/:id", h, []string{GET})
	tree.insert("/users/:id", r)

	var ps Params
	got, ok := tree.lookup("/users/42", &ps)
	assert.True(t, ok)
	assert.Same(t, r, got)
	assert.Eq(t, "42", ps.Get("id"))
}

func TestTreeLookup_Wildcard(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	r := newRoute("/files/*path", h, []string{GET})
	tree.insert("/files/*path", r)

	var ps Params
	got, ok := tree.lookup("/files/a/b/c.txt", &ps)
	assert.True(t, ok)
	assert.Same(t, r, got)
	assert.Eq(t, "a/b/c.txt", ps.Get("path"))
}

func TestTreeLookup_StaticBeatsWildcard(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	rWild := newRoute("/users/*all", h, []string{GET})
	rStatic := newRoute("/users/me", h, []string{GET})
	tree.insert("/users/*all", rWild)
	tree.insert("/users/me", rStatic)

	var ps Params
	got, ok := tree.lookup("/users/me", &ps)
	assert.True(t, ok)
	assert.Same(t, rStatic, got)
}

func TestTreeLookup_StaticBeatsParam(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	rParam := newRoute("/users/:id", h, []string{GET})
	rStatic := newRoute("/users/me", h, []string{GET})
	tree.insert("/users/:id", rParam)
	tree.insert("/users/me", rStatic)

	var ps Params
	got, ok := tree.lookup("/users/me", &ps)
	assert.True(t, ok)
	assert.Same(t, rStatic, got)
}

func TestTreeLookup_ParamBeatsWildcard(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	rWild := newRoute("/files/*all", h, []string{GET})
	rParam := newRoute("/files/:name", h, []string{GET})
	tree.insert("/files/*all", rWild)
	tree.insert("/files/:name", rParam)

	var ps Params
	got, ok := tree.lookup("/files/foo.txt", &ps)
	assert.True(t, ok)
	assert.Same(t, rParam, got)
	assert.Eq(t, "foo.txt", ps.Get("name"))
}

func TestTreeLookup_Miss(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	tree.insert("/users", newRoute("/users", h, []string{GET}))

	var ps Params
	_, ok := tree.lookup("/posts", &ps)
	assert.False(t, ok)

	_, ok = tree.lookup("/", &ps)
	assert.False(t, ok)
}

func TestTreeLookup_MultipleParams(t *testing.T) {
	tree := newRadixTree()
	h := func(c *Context) {}
	r := newRoute("/users/:uid/posts/:pid", h, []string{GET})
	tree.insert("/users/:uid/posts/:pid", r)

	var ps Params
	got, ok := tree.lookup("/users/42/posts/100", &ps)
	assert.True(t, ok)
	assert.Same(t, r, got)
	assert.Eq(t, "42", ps.Get("uid"))
	assert.Eq(t, "100", ps.Get("pid"))
}
