package rux

import (
	"io"
	"net/url"
	"strings"

	"github.com/gookit/goutil"
)

/*************************************************************
 * Extends interfaces definition
 *************************************************************/

// Renderer interface
type Renderer interface {
	Render(io.Writer, string, any, *Context) error
}

// Validator interface
type Validator interface {
	Validate(i any) error
}

/*************************************************************
 * Quick build uri by route name
 *************************************************************/

// BuildRequestURL struct
type BuildRequestURL struct {
	queries url.Values
	params  M
	path    string
	scheme  string
	host    string
	user    *url.Userinfo
}

// NewBuildRequestURL get new obj
func NewBuildRequestURL() *BuildRequestURL {
	return &BuildRequestURL{
		queries: make(url.Values),
		params:  make(M),
	}
}

// Queries set Queries
func (b *BuildRequestURL) Queries(queries url.Values) *BuildRequestURL {
	b.queries = queries
	return b
}

// Params set Params
func (b *BuildRequestURL) Params(params M) *BuildRequestURL {
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

// Build url
func (b *BuildRequestURL) Build(withParams ...M) *url.URL {
	var path = b.path

	if len(withParams) > 0 {
		for k, d := range withParams[0] {
			if strings.IndexByte(k, '{') == -1 && strings.IndexByte(k, '}') == -1 {
				b.queries.Add(k, goutil.String(d))
			} else {
				b.params[k] = goutil.String(d)
			}
		}
	}

	var u = new(url.URL)

	u.Scheme = b.scheme
	u.User = b.user
	u.Host = b.host
	u.Path = path
	u.RawQuery = b.queries.Encode()

	ss := varRegex.FindAllString(path, -1)

	if len(ss) == 0 {
		return u
	}

	var n string
	var varParams = make(map[string]string)

	// TODO should optimize ...
	for _, str := range ss {
		nvStr := str[1 : len(str)-1]

		if strings.IndexByte(nvStr, ':') > 0 {
			nv := strings.SplitN(nvStr, ":", 2)
			n, _ = strings.TrimSpace(nv[0]), strings.TrimSpace(nv[1])
			varParams[str] = "{" + n + "}"
		} else {
			varParams[str] = str
		}
	}

	for paramRegex, name := range varParams {
		path = strings.NewReplacer(paramRegex, goutil.String(b.params[name])).Replace(path)
	}

	u.Path = path

	return u
}
