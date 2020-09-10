package rux

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRequestUrl_Params(t *testing.T) {
	is := assert.New(t)

	b := NewBuildRequestURL()
	b.Path(`/news/{category_id}/{new_id}/detail`)
	b.Params(M{"{category_id}": "100", "{new_id}": "20"})

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

func TestBuildRequestURL_User(t *testing.T) {
	is := assert.New(t)

	b := NewBuildRequestURL()
	b.Scheme("https")
	b.User("tom", "123")
	b.Host("127.0.0.1")
	b.Path(`/news`)

	is.Equal(b.Build().String(), `https://tom:123@127.0.0.1/news`)
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
	b.Params(M{"{name}": "test", "{id}": "20"})

	is.Equal(r.BuildURL("homepage", b).String(), `/build-test/test/20`)
	is.Equal(r.BuildRequestURL("homepage_fiexd_path").String(), `/build-test/fiexd/path`)
}

func TestBuildRequestUrl_With(t *testing.T) {
	r := New()
	is := assert.New(t)

	homepage := NewNamedRoute("homepage", `/build-test/{name}/{id:\d+}`, emptyHandler, GET)

	r.AddRoute(homepage)

	is.Equal(r.BuildRequestURL("homepage", M{
		"{name}":   "test",
		"{id}":     20,
		"username": "demo",
	}).String(), `/build-test/test/20?username=demo`)
}

func TestBuildRequestUrl_WithCustom(t *testing.T) {
	is := assert.New(t)

	b := NewBuildRequestURL()
	b.Path("/build-test/test/{id}")

	is.Equal(b.Build(M{
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
	is.Equal(`/build-test/test/20?username=demo`, str)
}

func TestBuildRequestUrl_WithMutilArgs2(t *testing.T) {
	r := New()
	is := assert.New(t)

	homepage := NewNamedRoute("homepage", `/build-test`, emptyHandler, GET)

	r.AddRoute(homepage)

	str := r.BuildRequestURL("homepage", "{name}", "test", "{id}", 20, "username", "demo").String()
	is.Equal(`/build-test?username=demo`, str)

	str = r.BuildURL("homepage", "{name}", "test", "{id}", 20).String()
	is.Equal(`/build-test`, str)
}

func TestBuildRequestUrl_WithMutilArgs3(t *testing.T) {
	r := New()
	is := assert.New(t)

	homepage := NewNamedRoute("homepage", `/build-test/{id}`, emptyHandler, GET)

	r.AddRoute(homepage)

	str := r.BuildRequestURL("homepage", "{name}", "test", "{id}", 20, "username", "demo").String()
	is.Equal(`/build-test/20?username=demo`, str)

	str = r.BuildURL("homepage", "{name}", "test", "{id}", 23).String()
	is.Equal(`/build-test/23`, str)
}

func TestBuildRequestUrl_EmptyRoute(t *testing.T) {
	r := New()
	is := assert.New(t)

	homepage := NewNamedRoute("homepage", `/build-test/{name}/{id:\d+}`, emptyHandler, GET)

	r.AddRoute(homepage)

	is.PanicsWithValue("BuildRequestURL get route is nil(name: homepage-empty)", func() {
		r.BuildRequestURL("homepage-empty", "{name}", "test", "{id}", "20", "username", "demo")
	})
}

func TestBuildRequestUrl_ErrorArgs(t *testing.T) {
	r := New()
	is := assert.New(t)

	homepage := NamedRoute("homepage", `/build-test/{name}/{id:\d+}`, emptyHandler, GET)
	r.AddRoute(homepage)

	is.PanicsWithValue("buildArgs odd argument count", func() {
		r.BuildRequestURL("homepage", "one")
	})

	is.PanicsWithValue("buildArgs odd argument count", func() {
		r.BuildRequestURL("homepage", "{name}", "test", "{id}")
	})
}


type MyBinder string

func (b *MyBinder) Bind(v interface{}, c *Context) error {
	if c.IsPost() {
		if err := json.NewDecoder(c.Req.Body).Decode(v); err != nil {
			return err
		}
	}

	return nil
}

func TestContext_Binder(t *testing.T) {
	ris := assert.New(t)
	r := New()

	r.Binder = new(MyBinder)

	r.Any("/binder", func(c *Context) {
		var form = new(struct {
			Username string `json:"username"`
			Password string `json:"password"`
		})

		if err := c.Bind(form); err != nil {
			c.AbortThen().Text(200, "binder error")
		}

		c.Text(200, fmt.Sprintf("%s=%s", form.Username, form.Password))
	})

	w := mockRequest(r, POST, "/binder", &md{B: `{"username":"admin","password":"123456"}`})

	ris.Equal(200, w.Code)
	ris.Equal(w.Body.String(), `admin=123456`)
}

type MyValidator string

func (mv *MyValidator) Validate(v interface{}) error {
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

	r.Validator = new(MyValidator)

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

	ris.Equal(200, w.Code)
	ris.Equal(w.Body.String(), `passed`)
}

type MyRenderer string

func (mr *MyRenderer) Render(w io.Writer, name string, data interface{}, ctx *Context) error {
	tpl, err := template.New(name).Funcs(template.FuncMap{
		"Upper": strings.ToUpper,
	}).Parse("{{.Name|Upper}}, ID is {{ .ID}}")

	if err != nil {
		return err
	}

	return tpl.Execute(w, data)
}

func TestContext_Renderer(t *testing.T) {
	ris := assert.New(t)
	r := New()

	r.Renderer = new(MyRenderer)

	r.Any("/renderer", func(c *Context) {
		c.Render(200, "index", M{
			"ID":   100,
			"Name": "admin",
		})
	})

	w := mockRequest(r, GET, "/renderer", nil)

	ris.Equal(200, w.Code)
	ris.Equal(w.Body.String(), `ADMIN, ID is 100`)
}
