package rux

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRequestUrl_Params(t *testing.T) {
	is := assert.New(t)

	b := NewBuildRequestURL()
	b.Path(`/news/{category_id}/{new_id}/detail`)
	b.Params("{category_id}", "100", "{new_id}", "20")

	is.Equal(b.Build().String(), `/news/100/20/detail`)
}

func TestBuildRequestUrl_Host(t *testing.T) {
	is := assert.New(t)

	b := NewBuildRequestURL()
	b.Scheme("https")
	b.Host("127.0.0.1")
	b.Path(`/news`)

	is.Equal(b.Build().String(), `https://127.0.0.1/news`)
}

func TestBuildRequestUrl_Queries(t *testing.T) {
	is := assert.New(t)

	var u = make(url.Values)
	u.Add("username", "admin")
	u.Add("password", "12345")

	b := NewBuildRequestURL()
	b.Queries(u)
	b.Path(`/news`)

	is.Equal(b.Build().String(), `/news?password=12345&username=admin`)
}

func TestBuildRequestUrl_Build(t *testing.T) {
	is := assert.New(t)

	r := New()

	homepage := NewNamedRoute("homepage", `/build-test/{name}/{id:\d+}`, emptyHandler, GET)
	r.AddRoute(homepage)

	b := NewBuildRequestURL()
	b.Params("{name}", "test", "{id}", "20")

	is.Equal(r.BuildRequestURL("homepage", b).String(), `/build-test/test/20`)
}
