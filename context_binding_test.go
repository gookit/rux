package rux

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gookit/goutil/netutil/httpctype"
	"github.com/gookit/goutil/testutil"
	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/rux/binding"
)

var (
	userQuery = "age=12&name=inhere"
	userJSON  = `{"age": 12, "name": "inhere"}`
	userXML   = `<?xml version="1.0" encoding="UTF-8" ?>
<root>
  <age type="number">12</age>
  <name type="string">inhere</name>
</root>`
)

type User struct {
	Age  int    `query:"age" form:"age" xml:"age"`
	Name string `query:"name" form:"name" xml:"name"`
}

func TestContext_ShouldBind(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.POST("/ShouldBind", func(c *Context) {
		u := &User{}

		err := c.ShouldBind(u, binding.JSON)
		testBoundedUserIsOK(is, err, u)
	})
	r.POST("/ShouldBind-err", func(c *Context) {
		u := &User{}

		err := c.ShouldBind(u, binding.JSON)
		is.Err(err)
		is.Eq("invalid character 'i' looking for beginning of value", err.Error())
		c.SetStatus(http.StatusInternalServerError)
	})

	w := testutil.MockRequest(r, POST, "/ShouldBind", &testutil.MD{
		Body: strings.NewReader(userJSON),
	})
	is.Eq(http.StatusOK, w.Code)

	w = testutil.MockRequest(r, POST, "/ShouldBind-err", &testutil.MD{
		Body: strings.NewReader("invalid-json"),
	})
	is.Eq(http.StatusInternalServerError, w.Code)
}

func TestContext_MustBind(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.POST("/MustBind", func(c *Context) {
		u := &User{}

		c.MustBind(u, binding.JSON)
		is.Eq(12, u.Age)
		is.Eq("inhere", u.Name)
	})

	w := testutil.MockRequest(r, POST, "/MustBind", &testutil.MD{
		Body: strings.NewReader(userJSON),
	})
	is.Eq(http.StatusOK, w.Code)

	r.OnPanic = func(c *Context) {
		ret, ok := c.Get(CTXRecoverResult)
		is.True(ok)
		err, ok := ret.(error)
		is.True(ok)
		is.Eq("invalid character 'i' looking for beginning of value", err.Error())
	}
	w = testutil.MockRequest(r, POST, "/MustBind", &testutil.MD{
		Body: strings.NewReader("invalid-json"),
	})
	is.Eq(http.StatusOK, w.Code)
}

func TestContext_Bind(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.Add("/Bind", func(c *Context) {
		u := &User{}

		fmt.Printf(" - auto bind data by content type: %s\n", c.ContentType())
		err := c.Bind(u)
		testBoundedUserIsOK(is, err, u)
	}, GET, POST)

	// post Form body
	w := testutil.MockRequest(r, POST, "/Bind", &testutil.MD{
		Body: strings.NewReader(userQuery),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEPOSTForm,
		},
	})
	is.Eq(http.StatusOK, w.Code)
}

func TestContext_AutoBind(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.Add("/AutoBind", func(c *Context) {
		u := &User{}

		if ctype := c.ContentType(); ctype != "" {
			fmt.Printf(" - auto bind data by content type: %s\n", ctype)
		} else {
			fmt.Println(" - auto bind data from URL query string")
		}

		err := c.AutoBind(u)
		testBoundedUserIsOK(is, err, u)
	}, GET, POST)

	// post Form body
	w := testutil.MockRequest(r, POST, "/AutoBind", &testutil.MD{
		Body: strings.NewReader(userQuery),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEPOSTForm,
		},
	})
	is.Eq(http.StatusOK, w.Code)

	// post JSON body
	w = testutil.MockRequest(r, POST, "/AutoBind", &testutil.MD{
		Body: strings.NewReader(userJSON),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEJSON,
		},
	})
	is.Eq(http.StatusOK, w.Code)

	// post XML body
	w = testutil.MockRequest(r, POST, "/AutoBind", &testutil.MD{
		Body: strings.NewReader(userXML),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEXML,
		},
	})
	is.Eq(http.StatusOK, w.Code)

	// URL query string
	w = testutil.MockRequest(r, GET, "/AutoBind?"+userQuery, nil)
	is.Eq(http.StatusOK, w.Code)
}

func TestContext_BindForm(t *testing.T) {
	r := New()
	is := assert.New(t)

	r.POST("/BindForm", func(c *Context) {
		u := &User{}
		err := c.BindForm(u)
		testBoundedUserIsOK(is, err, u)
	})

	w := testutil.MockRequest(r, POST, "/BindForm", &testutil.MD{
		Body: strings.NewReader(userQuery),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEPOSTForm,
		},
	})
	is.Eq(http.StatusOK, w.Code)
}

func TestContext_BindJSON(t *testing.T) {
	r := New()
	is := assert.New(t)

	r.POST("/BindJSON", func(c *Context) {
		u := &User{}
		err := c.BindJSON(u)
		testBoundedUserIsOK(is, err, u)
	})

	w := testutil.MockRequest(r, POST, "/BindJSON", &testutil.MD{
		Body: strings.NewReader(userJSON),
	})
	is.Eq(http.StatusOK, w.Code)
}

func TestContext_BindXML(t *testing.T) {
	r := New()
	is := assert.New(t)

	r.POST("/BindXML", func(c *Context) {
		u := &User{}
		err := c.BindXML(u)
		testBoundedUserIsOK(is, err, u)
	})

	w := testutil.MockRequest(r, POST, "/BindXML", &testutil.MD{
		Body: strings.NewReader(userXML),
	})
	is.Eq(http.StatusOK, w.Code)
}

func testBoundedUserIsOK(is *assert.Assertions, err error, u *User) {
	is.NoErr(err)
	is.NotEmpty(u)
	is.Eq(12, u.Age)
	is.Eq("inhere", u.Name)
}
