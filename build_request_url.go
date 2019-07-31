package rux

import (
	"net/url"
	"strings"
)

type BuildRequestUrl struct {
	queries url.Values
	params  []string
	path    string
	scheme  string
	host    string
	user    *url.Userinfo
}

func NewBuildRequestUrl() *BuildRequestUrl {
	return &BuildRequestUrl{}
}

func (b *BuildRequestUrl) Queries(queries url.Values) *BuildRequestUrl {
	b.queries = queries

	return b
}

func (b *BuildRequestUrl) Params(params ...string) *BuildRequestUrl {
	b.params = params

	return b
}

func (b *BuildRequestUrl) Scheme(scheme string) *BuildRequestUrl {
	b.scheme = scheme

	return b
}

func (b *BuildRequestUrl) User(username, password string) *BuildRequestUrl {
	b.user = url.UserPassword(username, password)

	return b
}

func (b *BuildRequestUrl) Host(host string) *BuildRequestUrl {
	b.host = host

	return b
}

func (b *BuildRequestUrl) Path(path string) *BuildRequestUrl {
	b.path = path

	return b
}

func (b *BuildRequestUrl) Build() *url.URL {
	var path = b.path

	if len(b.params) > 0 {
		path = strings.NewReplacer(b.params...).Replace(path)
	}

	return &url.URL{
		Scheme:   b.scheme,
		User:     b.user,
		Host:     b.host,
		Path:     path,
		RawQuery: b.queries.Encode(),
	}
}
