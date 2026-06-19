package core

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestParams_Empty(t *testing.T) {
	var p Params
	assert.Eq(t, 0, p.Len())
	assert.Eq(t, "", p.Get("missing"))
	assert.False(t, p.Has("missing"))
	assert.Eq(t, 0, p.Int("missing"))
}

func TestParams_AppendAndGet(t *testing.T) {
	var p Params
	p.append("id", "42")
	p.append("slug", "hello")
	assert.Eq(t, 2, p.Len())
	assert.Eq(t, "42", p.Get("id"))
	assert.Eq(t, "hello", p.Get("slug"))
	assert.True(t, p.Has("id"))
	assert.False(t, p.Has("missing"))
	assert.Eq(t, 42, p.Int("id"))
	assert.Eq(t, 0, p.Int("slug")) // not int-parseable
	assert.Eq(t, 0, p.Int("missing"))
}

func TestParams_Reset(t *testing.T) {
	var p Params
	p.append("a", "1")
	p.append("b", "2")
	p.Reset()
	assert.Eq(t, 0, p.Len())
	assert.Eq(t, "", p.Get("a"))
}

func TestParams_Snapshot(t *testing.T) {
	var p Params
	p.append("k", "v")
	snap := p.Snapshot()
	assert.Eq(t, 1, len(snap))
	assert.Eq(t, "k", snap[0].Key)
	assert.Eq(t, "v", snap[0].Value)

	// mutating snapshot must not affect original
	snap[0].Value = "modified"
	assert.Eq(t, "v", p.Get("k"))
}

func TestParams_OverflowPanics(t *testing.T) {
	var p Params
	for i := 0; i < MaxParams; i++ {
		p.append("k", "v")
	}
	assert.Panics(t, func() { p.append("overflow", "x") })
}
