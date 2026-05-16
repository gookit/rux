package core

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

func TestParseOptionalSegments(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"/posts", []string{"/posts"}},
		{"/posts[/{id}]", []string{"/posts", "/posts/:id"}},
		{"/api/users[/{name}]/profile",
			[]string{"/api/users/profile", "/api/users/:name/profile"}},
		{"/about[.html]", []string{"/about", "/about.html"}},
	}
	for _, c := range cases {
		got := parseOptionalSegments(c.in)
		assert.Eq(t, c.want, got, "input=%q", c.in)
	}
}

func TestConvertParamSyntax(t *testing.T) {
	cases := map[string]string{
		"/users/{id}":       "/users/:id",
		"/users/{id}/posts": "/users/:id/posts",
		"/files/{path:.+}":  "/files/*path",
		"/files/{path:.*}":  "/files/*path",
	}
	for in, want := range cases {
		assert.Eq(t, want, convertParamSyntax(in), "input=%q", in)
	}
}

func TestHasOptionalSegment(t *testing.T) {
	cases := map[string]bool{
		"/users":         false,
		"/posts[/{id}]":  true,
		"/about[.html]":  true,
		"/users/{id}":    false, // braces, not brackets
		"/file{x:[1-9]}": false, // bracket inside braces
	}
	for in, want := range cases {
		assert.Eq(t, want, hasOptionalSegment(in), "input=%q", in)
	}
}
