package rux

import (
	"net/url"
	"strings"
)

// BuildRequestURL struct
type BuildRequestURL struct {
	queries url.Values
	params  []string
	path    string
	scheme  string
	host    string
	user    *url.Userinfo
}

// NewBuildRequestURL get new obj
func NewBuildRequestURL() *BuildRequestURL {
	return &BuildRequestURL{}
}

// Queries set Queries
func (b *BuildRequestURL) Queries(queries url.Values) *BuildRequestURL {
	b.queries = queries

	return b
}

// Params set Params
func (b *BuildRequestURL) Params(params ...string) *BuildRequestURL {
	b.params = params

	return b
}

// Scheme set Scheme
func (b *BuildRequestURL) Scheme(scheme string) *BuildRequestURL {
	b.scheme = scheme

	return b
}

// User set User
func (b *BuildRequestURL) User(username, password string) *BuildRequestURL {
	b.user = url.UserPassword(username, password)

	return b
}

// Host set Host
func (b *BuildRequestURL) Host(host string) *BuildRequestURL {
	b.host = host

	return b
}

// Path set Path
func (b *BuildRequestURL) Path(path string) *BuildRequestURL {
	b.path = path

	return b
}

// Build build url
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
