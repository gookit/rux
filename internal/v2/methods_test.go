package v2

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestMethodIndex(t *testing.T) {
	cases := []struct {
		method string
		want   int
	}{
		{"GET", 0},
		{"HEAD", 1},
		{"POST", 2},
		{"PUT", 3},
		{"PATCH", 4},
		{"DELETE", 5},
		{"OPTIONS", 6},
		{"CONNECT", 7},
		{"TRACE", 8},
		{"", -1},
		{"FOO", -1},
		{"get", -1},
	}
	for _, c := range cases {
		assert.Eq(t, c.want, methodIndex(c.method), "method=%s", c.method)
	}
}
