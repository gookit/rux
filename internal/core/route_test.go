package core

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestHandlersChain_LengthSemantics(t *testing.T) {
	h1 := func(c *Context) {}
	h2 := func(c *Context) {}
	chain := HandlersChain{h1, h2}
	assert.Eq(t, 2, len(chain))
}

func TestNewRoute_Defaults(t *testing.T) {
	h := func(c *Context) {}
	r := newRoute("/users", h, []string{GET})
	assert.Eq(t, "/users", r.Path())
	assert.Eq(t, []string{GET}, r.Methods())
	assert.Eq(t, 1, len(r.chain), "main handler appended")
}

func TestNewRoute_PanicsOnNilHandler(t *testing.T) {
	assert.Panics(t, func() {
		newRoute("/users", nil, []string{GET})
	})
}

func TestNewRoute_DefaultMethodGET(t *testing.T) {
	h := func(c *Context) {}
	r := newRoute("/users", h, nil)
	assert.Eq(t, []string{GET}, r.Methods())
}

func TestRoute_Use_PrependsBeforeMain(t *testing.T) {
	var order []string
	h := func(c *Context) { order = append(order, "main") }
	mw := func(c *Context) { order = append(order, "mw") }

	r := newRoute("/x", h, []string{GET})
	r.Use(mw)

	// chain should be [mw, h] — middleware before main
	assert.Eq(t, 2, len(r.chain))
	r.chain[0](nil)
	r.chain[1](nil)
	assert.Eq(t, []string{"mw", "main"}, order)
}
