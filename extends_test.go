package rux

import (
	"errors"
	"html/template"
	"io"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestBuildRequestUrl_Params(t *testing.T) {
	is := assert.New(t)

	b := NewBuildRequestURL()
	b.Path(`/news/{category_id}/{new_id}/detail`)
	b.Params(M{"{category_id}": "100", "{new_id}": "20"})

	is.Eq(b.Build().String(), `/news/100/20/detail`)
}

func TestBuildRequestUrl_Host(t *testing.T) {
	is := assert.New(t)

	b := NewBuildRequestURL()
	b.Scheme("https")
	b.Host("127.0.0.1")
	b.Path(`/news`)

	is.Eq(b.Build().String(), `https://127.0.0.1/news`)
}

func TestBuildRequestURL_User(t *testing.T) {
	is := assert.New(t)

	b := NewBuildRequestURL()
	b.Scheme("https")
	b.User("tom", "123")
	b.Host("127.0.0.1")
	b.Path(`/news`)

	is.Eq(b.Build().String(), `https://tom:123@127.0.0.1/news`)
}

func TestBuildRequestUrl_Queries(t *testing.T) {
	is := assert.New(t)

	var u = make(url.Values)
	u.Add("username", "admin")
	u.Add("password", "12345")

	b := NewBuildRequestURL()
	b.Queries(u)
	b.Path(`/news`)

	is.Eq(b.Build().String(), `/news?password=12345&username=admin`)
}

func TestBuildRequestUrl_Build(t *testing.T) {
	is := assert.New(t)

	r := New()

	homepage := NewNamedRoute("homepage", `/build-test/{name}/{id:\d+}`, emptyHandler, GET)
	homepageFiexdPath := NewNamedRoute("homepage_fiexd_path", `/build-test/fiexd/path`, emptyHandler, GET)

	r.AddRoute(homepage)
	r.AddRoute(homepageFiexdPath)

	b := NewBuildRequestURL()
	b.Params(M{"{name}": "test", "{id}": "20"})

	is.Eq(r.BuildURL("homepage", b).String(), `/build-test/test/20`)
	is.Eq(r.BuildRequestURL("homepage_fiexd_path").String(), `/build-test/fiexd/path`)
}

func TestBuildRequestUrl_With(t *testing.T) {
	r := New()
	is := assert.New(t)

	homepage := NewNamedRoute("homepage", `/build-test/{name}/{id:\d+}`, emptyHandler, GET)

	r.AddRoute(homepage)

	is.Eq(r.BuildRequestURL("homepage", M{
		"{name}":   "test",
		"{id}":     20,
		"username": "demo",
	}).String(), `/build-test/test/20?username=demo`)
}

func TestBuildRequestUrl_WithCustom(t *testing.T) {
	is := assert.New(t)

	b := NewBuildRequestURL()
	b.Path("/build-test/test/{id}")

	is.Eq(b.Build(M{
		"{id}":     20,
		"username": "demo",
	}).String(), `/build-test/test/20?username=demo`)
}

func TestBuildRequestUrl_WithMutilArgs(t *testing.T) {
	r := New()
	is := assert.New(t)

	homepage := NewNamedRoute("homepage", `/build-test/{name}/{id:\d+}`, emptyHandler, GET)

	r.AddRoute(homepage)

	str := r.BuildRequestURL("homepage", "{name}", "test", "{id}", 20, "username", "demo").String()
	is.Eq(`/build-test/test/20?username=demo`, str)
}

func TestBuildRequestUrl_WithMutilArgs2(t *testing.T) {
	r := New()
	is := assert.New(t)

	homepage := NewNamedRoute("homepage", `/build-test`, emptyHandler, GET)

	r.AddRoute(homepage)

	str := r.BuildRequestURL("homepage", "{name}", "test", "{id}", 20, "username", "demo").String()
	is.Eq(`/build-test?username=demo`, str)

	str = r.BuildURL("homepage", "{name}", "test", "{id}", 20).String()
	is.Eq(`/build-test`, str)
}

func TestBuildRequestUrl_WithMutilArgs3(t *testing.T) {
	r := New()
	is := assert.New(t)

	homepage := NewNamedRoute("homepage", `/build-test/{id}`, emptyHandler, GET)

	r.AddRoute(homepage)

	str := r.BuildRequestURL("homepage", "{name}", "test", "{id}", 20, "username", "demo").String()
	is.Eq(`/build-test/20?username=demo`, str)

	str = r.BuildURL("homepage", "{name}", "test", "{id}", 23).String()
	is.Eq(`/build-test/23`, str)
}

func TestBuildRequestUrl_EmptyRoute(t *testing.T) {
	r := New()
	is := assert.New(t)

	homepage := NewNamedRoute("homepage", `/build-test/{name}/{id:\d+}`, emptyHandler, GET)

	r.AddRoute(homepage)

	is.PanicsMsg(func() {
		r.BuildRequestURL("homepage-empty", "{name}", "test", "{id}", "20", "username", "demo")
	}, "BuildRequestURL get route is nil(name: homepage-empty)")
}

func TestBuildRequestUrl_ErrorArgs(t *testing.T) {
	r := New()
	is := assert.New(t)

	homepage := NamedRoute("homepage", `/build-test/{name}/{id:\d+}`, emptyHandler, GET)
	r.AddRoute(homepage)

	is.PanicsMsg(func() {
		r.BuildRequestURL("homepage", "one")
	}, "buildArgs odd argument count")

	is.PanicsMsg(func() {
		r.BuildRequestURL("homepage", "{name}", "test", "{id}")
	}, "buildArgs odd argument count")
}

type MyValidator string

func (mv *MyValidator) Validate(v any) error {
	var rt = reflect.TypeOf(v)
	var rv = reflect.ValueOf(v)
	var field = rt.Elem().Field(0)
	var value = rv.Elem().Field(0)
	var rules = field.Tag.Get("valid")

	for _, rule := range strings.Split(rules, "|") {
		if rule == "required" && value.Interface() == "" {
			return errors.New("must required")
		}

		if rule == "email" && value.Interface() != "admin@me.com" {
			return errors.New("email error")
		}
	}

	return nil
}

func TestContext_Validator(t *testing.T) {
	ris := assert.New(t)
	r := New()

	// r.Validator = new(MyValidator)

	r.Any("/validator", func(c *Context) {
		var form = new(struct {
			Username string `valid:"required|email"`
		})

		form.Username = "admin@me.com"

		if err := c.Validate(form); err != nil {
			c.Text(200, err.Error())
			return
		}

		c.Text(200, "passed")
	})

	w := mockRequest(r, GET, "/validator", nil)

	ris.Eq(200, w.Code)
	ris.Eq(w.Body.String(), `passed`)
}

type MyRenderer string

func (mr *MyRenderer) Render(w io.Writer, name string, data any, _ *Context) error {
	tpl, err := template.New(name).Funcs(template.FuncMap{
		"Upper": strings.ToUpper,
	}).Parse("{{.Name|Upper}}, ID is {{ .ID}}")

	if err != nil {
		return err
	}
	return tpl.Execute(w, data)
}

func TestContext_Renderer(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.Renderer = new(MyRenderer)

	r.Any("/renderer", func(c *Context) {
		_ = c.Render(200, "index", M{
			"ID":   100,
			"Name": "admin",
		})
	})

	w := mockRequest(r, GET, "/renderer", nil)

	is.Eq(200, w.Code)
	is.Eq(`ADMIN, ID is 100`, w.Body.String())
}
