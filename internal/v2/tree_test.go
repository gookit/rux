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
