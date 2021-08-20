package rux_test

import (
	"testing"

	"github.com/gookit/goutil/dump"
	"github.com/gookit/rux"
	"github.com/stretchr/testify/assert"
)

func TestIssue_60(t *testing.T)  {
	r := rux.New()
	is := assert.New(t)

	r.GET(`/blog/{id:\d+}`, func(c *rux.Context) {
		c.Text(200, "view detail, id: " + c.Param("id"))
	})

	route, _, _ := r.Match("GET", "/blog/100")
	is.NotEmpty(route)
	dump.P(route.Info())

	route, _, _ = r.Match("GET", "/blog/100/")
	is.NotEmpty(route)
	dump.P(route.Info())

	r1 := rux.New(rux.StrictLastSlash)
	r1.GET(`/blog/{id:\d+}`, func(c *rux.Context) {
		c.Text(200, "view detail, id: " + c.Param("id"))
	})

	route, _, _ = r1.Match("GET", "/blog/100/")
	is.Empty(route)
	// dump.P(route.Info())
}
