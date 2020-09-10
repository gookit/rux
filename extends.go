package rux

import (
	"bytes"
	"errors"
	"io"
	"net/url"
	"strings"
)

/*************************************************************
 * Extends interfaces definition
 *************************************************************/

// Binder interface
type Binder interface {
	Bind(i interface{}, c *Context) error
}

// Renderer interface
type Renderer interface {
	Render(io.Writer, string, interface{}, *Context) error
}

// Validator interface
type Validator interface {
	Validate(i interface{}) error
}

/*************************************************************
 * Context function extends()
 *************************************************************/

// Bind context bind struct
// Deprecated
// please use ShouldBind(),
func (c *Context) Bind(i interface{}) error {
	if c.router.Binder == nil {
		return errors.New("binder not registered")
	}

	return c.router.Binder.Bind(i, c)
}

// Render context template
func (c *Context) Render(status int, name string, data interface{}) (err error) {
	if c.router.Renderer == nil {
		return errors.New("renderer not registered")
	}

	var buf = new(bytes.Buffer)
	if err = c.router.Renderer.Render(buf, name, data, c); err != nil {
		return err
	}

	c.HTML(status, buf.Bytes())
	return
}

// Validate context validator
// Deprecated
// please use ShouldBind(), it will auto call validator
func (c *Context) Validate(i interface{}) error {
	if c.Router().Validator == nil {
		return errors.New("validator not registered")
	}

	return c.Router().Validator.Validate(i)
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

// Build build url
func (b *BuildRequestURL) Build(withParams ...M) *url.URL {
	var path = b.path

	if len(withParams) > 0 {
		for k, d := range withParams[0] {
			if strings.IndexByte(k, '{') == -1 && strings.IndexByte(k, '}') == -1 {
				b.queries.Add(k, toString(d))
			} else {
				b.params[k] = toString(d)
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
		path = strings.NewReplacer(paramRegex, toString(b.params[name])).Replace(path)
	}

	u.Path = path

	return u
}
