package core

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestRouter_BuildURL_Named(t *testing.T) {
	r := New()
	r.AddNamed("user_show", "/users/{id}", func(c *Context) {}, GET)
	u := r.BuildURL("user_show", M{"{id}": 42})
	assert.Eq(t, "/users/42", u.Path)
}

func TestRouter_BuildURL_KvPairs(t *testing.T) {
	r := New()
	r.AddNamed("user_show", "/users/{id}", func(c *Context) {}, GET)
	u := r.BuildURL("user_show", "{id}", 42)
	assert.Eq(t, "/users/42", u.Path)
}

func TestRouter_BuildURL_QueryParams(t *testing.T) {
	r := New()
	r.AddNamed("user_show", "/users/{id}", func(c *Context) {}, GET)
	u := r.BuildURL("user_show", M{"{id}": 7, "lang": "en"})
	assert.Eq(t, "/users/7", u.Path)
	assert.Eq(t, "en", u.Query().Get("lang"))
}

func TestRouter_BuildURL_UnknownRoutePanics(t *testing.T) {
	r := New()
	assert.Panics(t, func() {
		r.BuildURL("missing")
	})
}

func TestRouter_BuildURL_NoArgs(t *testing.T) {
	r := New()
	r.AddNamed("home", "/home", func(c *Context) {}, GET)
	u := r.BuildURL("home")
	assert.Eq(t, "/home", u.Path)
}

func TestRouter_BuildURL_BuilderArg(t *testing.T) {
	r := New()
	r.AddNamed("user_show", "/users/{id}", func(c *Context) {}, GET)
	b := NewBuildRequestURL().Scheme("https").Host("example.com")
	b.Params(M{"{id}": 99})
	u := r.BuildURL("user_show", b)
	assert.Eq(t, "https", u.Scheme)
	assert.Eq(t, "example.com", u.Host)
	assert.Eq(t, "/users/99", u.Path)
}
