package rux

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gookit/goutil/netutil/httpctype"
	"github.com/gookit/goutil/testutil"
	"github.com/gookit/rux/binding"
	"github.com/stretchr/testify/assert"
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
		is.NoError(err)
		is.Equal(12, u.Age)
		is.Equal("inhere", u.Name)
	})
	r.POST("/ShouldBind-err", func(c *Context) {
		u := &User{}

		err := c.ShouldBind(u, binding.JSON)
		is.Error(err)
		is.Equal("invalid character 'i' looking for beginning of value", err.Error())
		c.SetStatus(http.StatusInternalServerError)
	})

	w := testutil.MockRequest(r, POST, "/ShouldBind", &testutil.MD{
		Body: strings.NewReader(userJSON),
	})
	is.Equal(http.StatusOK, w.Code)

	w = testutil.MockRequest(r, POST, "/ShouldBind-err", &testutil.MD{
		Body: strings.NewReader("invalid-json"),
	})
	is.Equal(http.StatusInternalServerError, w.Code)
}

func TestContext_MustBind(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.POST("/MustBind", func(c *Context) {
		u := &User{}

		c.MustBind(u, binding.JSON)
		is.Equal(12, u.Age)
		is.Equal("inhere", u.Name)

		// fmt.Println(u)
		// bs, _ := xml.Marshal(u)
		// fmt.Println(string(bs))
	})

	w := testutil.MockRequest(r, POST, "/MustBind", &testutil.MD{
		Body: strings.NewReader(userJSON),
	})
	is.Equal(http.StatusOK, w.Code)

	r.OnPanic = func(c *Context) {
		ret, ok := c.Get(CTXRecoverResult)
		is.True(ok)
		err, ok := ret.(error)
		is.True(ok)
		is.Equal("invalid character 'i' looking for beginning of value", err.Error())
	}
	w = testutil.MockRequest(r, POST, "/MustBind", &testutil.MD{
		Body: strings.NewReader("invalid-json"),
	})
	is.Equal(http.StatusOK, w.Code)
}

func TestContext_Bind(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.Add("/Bind", func(c *Context) {
		u := &User{}

		fmt.Printf(" - auto bind data by content type: %s\n", c.ContentType())
		err := c.AutoBind(u)
		is.NoError(err)
		is.Equal(12, u.Age)
		is.Equal("inhere", u.Name)
	}, GET, POST)

	// post Form body
	w := testutil.MockRequest(r, POST, "/Bind", &testutil.MD{
		Body: strings.NewReader(userQuery),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEPOSTForm,
		},
	})
	is.Equal(http.StatusOK, w.Code)
}

func TestContext_AutoBind(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.Add("/AutoBind", func(c *Context) {
		u := &User{}

		fmt.Printf(" - auto bind data by content type: %s\n", c.ContentType())
		err := c.AutoBind(u)
		is.NoError(err)
		is.Equal(12, u.Age)
		is.Equal("inhere", u.Name)
	}, GET, POST)

	// post Form body
	w := testutil.MockRequest(r, POST, "/AutoBind", &testutil.MD{
		Body: strings.NewReader(userQuery),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEPOSTForm,
		},
	})
	is.Equal(http.StatusOK, w.Code)

	// post JSON body
	w = testutil.MockRequest(r, POST, "/AutoBind", &testutil.MD{
		Body: strings.NewReader(userJSON),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEJSON,
		},
	})
	is.Equal(http.StatusOK, w.Code)

	// post XML body
	w = testutil.MockRequest(r, POST, "/AutoBind", &testutil.MD{
		Body: strings.NewReader(userXML),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEXML,
		},
	})
	is.Equal(http.StatusOK, w.Code)
}
