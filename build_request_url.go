package rux

import (
	"net/url"
	"strings"
)

type BuildRequestURL struct {
	queries url.Values
	params  []string
	path    string
	scheme  string
	host    string
	user    *url.Userinfo
}

func NewBuildRequestURL() *BuildRequestURL {
	return &BuildRequestURL{}
}

func (b *BuildRequestURL) Queries(queries url.Values) *BuildRequestURL {
	b.queries = queries

	return b
}

func (b *BuildRequestURL) Params(params ...string) *BuildRequestURL {
	b.params = params

	return b
}

func (b *BuildRequestURL) Scheme(scheme string) *BuildRequestURL {
	b.scheme = scheme

	return b
}

func (b *BuildRequestURL) User(username, password string) *BuildRequestURL {
	b.user = url.UserPassword(username, password)

	return b
}

func (b *BuildRequestURL) Host(host string) *BuildRequestURL {
	b.host = host

	return b
}

func (b *BuildRequestURL) Path(path string) *BuildRequestURL {
	b.path = path

	return b
}

func (b *BuildRequestURL) Build() *url.URL {
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
