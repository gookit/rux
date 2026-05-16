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
