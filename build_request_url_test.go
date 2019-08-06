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
	homepageFiexdPath := NewNamedRoute("homepage_fiexd_path", `/build-test/fiexd/path`, emptyHandler, GET)

	r.AddRoute(homepage)
	r.AddRoute(homepageFiexdPath)

	b := NewBuildRequestURL()
	b.Params("{name}", "test", "{id}", "20")

	is.Equal(r.BuildRequestURL("homepage", b).String(), `/build-test/test/20`)
	is.Equal(r.BuildRequestURL("homepage_fiexd_path").String(), `/build-test/fiexd/path`)
}

func TestBuildRequestUrl_With(t *testing.T) {
	is := assert.New(t)

	r := New()

	homepage := NewNamedRoute("homepage", `/build-test/{name}/{id:\d+}`, emptyHandler, GET)

	r.AddRoute(homepage)

	is.Equal(r.BuildRequestURL("homepage", M{
		"{name}":   "test",
		"{id}":     "20",
		"username": "demo",
	}).String(), `/build-test/test/20?username=demo`)
}

func TestBuildRequestUrl_WithCustom(t *testing.T) {
	is := assert.New(t)

	b := NewBuildRequestURL()
	b.Path("/build-test/test/{id}")

	is.Equal(b.Build(M{
		"{id}":     "20",
		"username": "demo",
	}).String(), `/build-test/test/20?username=demo`)
}

func TestBuildRequestUrl_WithMutilArgs(t *testing.T) {
	is := assert.New(t)

	r := New()

	homepage := NewNamedRoute("homepage", `/build-test/{name}/{id:\d+}`, emptyHandler, GET)

	r.AddRoute(homepage)

	is.Equal(r.BuildRequestURL("homepage", "{name}", "test", "{id}", "20", "username", "demo").String(), `/build-test/test/20?username=demo`)
}
