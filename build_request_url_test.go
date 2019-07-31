package rux

import (
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

func TestBuildRequestUrl_Params(t *testing.T) {
	art := assert.New(t)

	b := NewBuildRequestUrl()
	b.Path(`/news/{category_id}/{new_id}/detail`)
	b.Params("{category_id}", "100", "{new_id}", "20")

	art.Equal(b.Build().String(), `/news/100/20/detail`)
}

func TestBuildRequestUrl_Host(t *testing.T) {
	art := assert.New(t)

	b := NewBuildRequestUrl()
	b.Scheme("https")
	b.Host("127.0.0.1")
	b.Path(`/news`)

	art.Equal(b.Build().String(), `https://127.0.0.1/news`)
}

func TestBuildRequestUrl_Queries(t *testing.T) {
	art := assert.New(t)

	var u = make(url.Values)
	u.Add("username", "admin")
	u.Add("password", "12345")

	b := NewBuildRequestUrl()
	b.Queries(u)
	b.Path(`/news`)

	art.Equal(b.Build().String(), `/news?password=12345&username=admin`)
}
