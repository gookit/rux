package v2

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{
		"":           "/",
		"/":          "/",
		"/users":     "/users",
		"users":      "/users",
		"/users/":    "/users",
		"//users":    "/users",
		"/users//":   "/users",
		"/a//b///c/": "/a/b/c",
	}
	for in, want := range cases {
		assert.Eq(t, want, normalizePath(in), "input=%q", in)
	}
}

func TestIsStaticPath(t *testing.T) {
	cases := map[string]bool{
		"/users":       true,
		"/users/{id}":  false,
		"/files/*path": false,
		"/path[/{x}]":  false,
		"/users/:id":   false,
		"/":            true,
	}
	for in, want := range cases {
		assert.Eq(t, want, isStaticPath(in), "input=%q", in)
	}
}

func TestLongestCommonPrefix(t *testing.T) {
	assert.Eq(t, 0, longestCommonPrefix("", "abc"))
	assert.Eq(t, 0, longestCommonPrefix("abc", ""))
	assert.Eq(t, 3, longestCommonPrefix("abc", "abcdef"))
	assert.Eq(t, 3, longestCommonPrefix("abcdef", "abc"))
	assert.Eq(t, 2, longestCommonPrefix("abxx", "abyy"))
	assert.Eq(t, 0, longestCommonPrefix("xyz", "abc"))
}
