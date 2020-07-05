package rux

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mockContext(method, uri string, body io.Reader, header m) *Context {
	w := httptest.NewRecorder()
	// create fake request
	r, _ := http.NewRequest(method, uri, body)
	r.RequestURI = r.URL.String()
	// add headers
	if len(header) > 0 {
		// req.Header.Set("Content-Type", "text/plain")
		for k, v := range header {
			r.Header.Set(k, v)
		}
	}

	c := &Context{}
	c.Init(w, r)
	return c
}

func TestContext_WithReqCtxValue(t *testing.T) {
	art := assert.New(t)
	c := mockContext("GET", "/", nil, nil)

	c.WithReqCtxValue("name", "inhere")

	art.Equal("inhere", c.ReqCtxValue("name"))
	art.False(c.IsTLS())
	art.False(c.IsAborted())
}

func TestContext_Query(t *testing.T) {
	art := assert.New(t)

	// c := mockContext("GET", "/test?a=12&b=tom&arr[]=4&arr[]=9", nil, nil)
	c := mockContext("GET", "/test?page=12&name=tom&arr=4&arr=9", nil, nil)

	art.Equal("GET", c.Req.Method)
	art.Equal("12", c.Query("page"))
	art.Equal("val0", c.Query("no-key", "val0"))
	ss, has := c.QueryParams("arr")
	art.True(has)
	art.Len(ss, 2)
	vs := c.QueryValues()
	art.Len(vs, 3)
	// fmt.Println(vs)

	art.Equal("", c.Post("page"))
	art.Equal("1", c.Post("page", "1"))

	val, has := c.PostParam("page")
	art.Equal("", val)
	art.False(has)
}

func TestContext_Post(t *testing.T) {
	ris := assert.New(t)
	body := bytes.NewBufferString("foo=bar&page=11&both=v0&foo=second")
	c := mockContext("POST", "/?both=v1", body, m{
		"Accept":    "application/json",
		ContentType: "application/x-www-form-urlencoded",
	})

	val, has := c.PostParam("page")
	ris.True(has)
	ris.Equal("11", val)
	ris.Equal("11", c.Post("page"))
	ris.Equal("11", c.Post("page", "1"))

	ris.Equal([]string{"application/json"}, c.AcceptedTypes())
	ris.Equal("application/x-www-form-urlencoded", c.ContentType())

	// test parse multipart/form-data
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	err := mw.WriteField("kay0", "val0")
	ris.NoError(err)
	ris.NoError(mw.Close()) // must call Close()

	c3 := mockContext("POST", "/", buf, m{
		"Content-Type": mw.FormDataContentType(),
	})

	err = c3.ParseMultipartForm(defaultMaxMemory)
	ris.NoError(err)

	f0 := c3.Req.Form
	ris.Equal("kay0=val0", f0.Encode())
}

func TestContext_FormParams(t *testing.T) {
	art := assert.New(t)

	c1 := mockContext("GET", "/test1?a=1&b=2&c=3", nil, nil)
	c2 := mockContext("GET", "/test2?a=1&b=2&c=3", nil, nil)

	var err error

	form1, err := c1.FormParams()

	art.NoError(err)

	form2, err := c2.FormParams([]string{"b"})

	art.NoError(err)

	art.Equal(form1.Encode(), "a=1&b=2&c=3")
	art.Equal(form2.Encode(), "a=1&c=3")

	// test parse multipart/form-data
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	err = mw.WriteField("kay0", "val0")
	art.NoError(err)
	err = mw.Close()
	art.NoError(err)

	c3 := mockContext("POST", "/", buf, m{
		"Content-Type": mw.FormDataContentType(),
	})

	f3, err := c3.FormParams()
	art.Equal("kay0=val0", f3.Encode())
}

func TestContext_SetCookie(t *testing.T) {
	ris := assert.New(t)
	c := mockContext("GET", "/?both=v1", nil, m{
		"Accept":    "application/json",
		ContentType: "application/x-www-form-urlencoded",
	})

	c.SetCookie("ck-name", "val", 30, "/", "abc.com", true, true)
	c.SetCookie("ck-name1", "val1", 40, "", "abc.com", true, true)

	// Header().Get() will only return first
	s := c.Resp.Header().Get("Set-Cookie")
	ris.NotEmpty(s)
	ris.Contains(s, "ck-name=val")

	hs := c.Resp.Header()
	ris.Contains(hs, "Set-Cookie")
	ris.Len(hs["Set-Cookie"], 2)
	ris.Contains(hs["Set-Cookie"][0], "ck-name=val")
	ris.Contains(hs["Set-Cookie"][1], "ck-name1=val1")
}

func TestContext_FormFile(t *testing.T) {
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)

	w, err := mw.CreateFormFile("file", "test.txt")
	if assert.NoError(t, err) {
		_, _ = w.Write([]byte("test"))
	}
	_ = mw.Close()

	c := mockContext("POST", "/", buf, m{
		"Content-Type": mw.FormDataContentType(),
	})

	f, err := c.FormFile("file")
	if assert.NoError(t, err) {
		assert.Equal(t, "test.txt", f.Filename)
	}

	assert.NoError(t, c.SaveFile(f, "testdata/test.txt"))
	assert.NoError(t, c.UploadFile("file", "testdata/test.txt"))
	assert.Error(t, c.UploadFile("no-exist", "testdata/test.txt"))
}

func TestContext_SaveFile(t *testing.T) {
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	_ = mw.Close()

	c := mockContext("POST", "/", buf, m{
		"Content-Type": mw.FormDataContentType(),
	})

	f := &multipart.FileHeader{
		Filename: "file",
	}
	assert.Error(t, c.SaveFile(f, "testdata/test.txt"))
}

func TestContext_FileContent(t *testing.T) {
	is := assert.New(t)

	c := mockContext("GET", "/site.md", nil, nil)
	c.FileContent("testdata/site.md", "new-name.md")

	w := c.RawWriter().(*httptest.ResponseRecorder)
	is.Equal(200, c.StatusCode())
	is.Equal(200, w.Code)
	ss, ok := w.Header()["Content-Type"]
	is.True(ok)
	is.Contains(ss[0], "text/plain")
	is.Equal("# readme", w.Body.String())
	is.Equal(8, c.writer.Length())

	c = mockContext("GET", "/site.md", nil, nil)
	c.FileContent("testdata/not-exist.md")
	w = c.RawWriter().(*httptest.ResponseRecorder)
	is.Equal(500, c.StatusCode())
	is.Equal(500, w.Code)
	is.Equal("Internal Server Error\n", w.Body.String())
}

func TestContext_Attachment(t *testing.T) {
	is := assert.New(t)

	c := mockContext("GET", "/site.md", nil, nil)
	c.Attachment("testdata/site.md", "new-name.md")

	w := c.RawWriter().(*httptest.ResponseRecorder)
	is.Equal(200, w.Code)
	ss, ok := w.Header()["Content-Type"]
	is.True(ok)
	is.Equal(ss[0], "application/octet-stream")
	ss, ok = w.Header()["Content-Disposition"]
	is.True(ok)
	is.Equal(ss[0], "attachment; filename=new-name.md")
	is.Equal("# readme", w.Body.String())
}

func TestContext_Inline(t *testing.T) {
	is := assert.New(t)

	// Inline
	c := mockContext("GET", "/site.md", nil, nil)
	c.Inline("testdata/site.md", "new-name.md")

	w := c.RawWriter().(*httptest.ResponseRecorder)
	is.Equal(200, w.Code)
	ss, ok := w.Header()["Content-Type"]
	is.True(ok)
	is.Equal(ss[0], "application/octet-stream")
	ss, ok = w.Header()["Content-Disposition"]
	is.True(ok)
	is.Equal(ss[0], "inline; filename=new-name.md")
	is.Equal("# readme", w.Body.String())
}

func TestContext_Binary(t *testing.T) {
	is := assert.New(t)
	c := mockContext("GET", "/site.md", nil, nil)

	in, _ := os.Open("testdata/site.md")
	c.Binary(200, in, "new-name.md", true)

	w := c.RawWriter().(*httptest.ResponseRecorder)
	is.Equal(200, w.Code)
	ss, ok := w.Header()["Content-Type"]
	is.True(ok)
	is.Equal(ss[0], "application/octet-stream")
	ss, ok = w.Header()["Content-Disposition"]
	is.True(ok)
	is.Equal(ss[0], "inline; filename=new-name.md")
	is.Equal("# readme", w.Body.String())
}

func TestContext_RouteName(t *testing.T) {
	is := assert.New(t)
	c := mockContext("GET", "/", nil, nil)

	c.Set(CTXCurrentRouteName, "test_name")

	name, ok := c.Get(CTXCurrentRouteName)

	is.True(ok)
	is.Equal("test_name", name)
}

func TestContext_RoutePath(t *testing.T) {
	is := assert.New(t)
	c := mockContext("GET", "/test/{name}", nil, nil)

	c.Set(CTXCurrentRoutePath, "/test/{name}")

	name, ok := c.Get(CTXCurrentRoutePath)

	is.True(ok)
	is.Equal("/test/{name}", name)
}

func TestContext_Cookie(t *testing.T) {
	ris := assert.New(t)

	r := New()
	r.GET("/test", func(c *Context) {
		val := c.Cookie("req-cke")
		ris.Equal("req-val", val)

		val = c.Cookie("not-exist")
		ris.Equal("", val)

		c.FastSetCookie("res-cke", "val1", 300)
	})

	w := mockRequest(r, GET, "/test", nil, func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: "req-cke", Value: "req-val"})
	})

	ris.Equal(200, w.Code)

	resCke := w.Header().Get("Set-Cookie")
	ris.Equal("res-cke=val1; Path=/; Max-Age=300; Secure", resCke)
}

func TestContext_Redirect(t *testing.T) {
	ris := assert.New(t)
	r := New()

	// Redirect()
	uri := "/Redirect"
	r.GET(uri, func(c *Context) {
		c.Redirect("/new-path")
	})
	w := mockRequest(r, GET, uri, nil)
	ris.Equal(301, w.Code)
	ris.Equal("/new-path", w.Header().Get("Location"))
	ris.Equal("<a href=\"/new-path\">Moved Permanently</a>.\n\n", w.Body.String())

	uri = "/Redirect1"
	r.GET(uri, func(c *Context) {
		c.Redirect("/new-path1", 302)
	})
	w = mockRequest(r, GET, uri, nil)
	ris.Equal(302, w.Code)
	ris.Equal("/new-path1", w.Header().Get("Location"))
	ris.Equal("<a href=\"/new-path1\">Found</a>.\n\n", w.Body.String())
}

func TestContext_Back(t *testing.T) {
	ris := assert.New(t)
	r := New()

	// Back()
	uri := "/Back"
	r.GET(uri, func(c *Context) {
		c.Back()
	})
	w := mockRequest(r, GET, uri, &md{H: m{"Referer": "/old-path"}})
	ris.Equal(302, w.Code)
	ris.Equal("/old-path", w.Header().Get("Location"))
	ris.Equal("<a href=\"/old-path\">Found</a>.\n\n", w.Body.String())

	// Back()
	uri = "/Back1"
	r.GET(uri, func(c *Context) {
		c.Back(301)
	})
	w = mockRequest(r, GET, uri, &md{H: m{"Referer": "/old-path1"}})
	ris.Equal(301, w.Code)
	ris.Equal("/old-path1", w.Header().Get("Location"))
	ris.Equal("<a href=\"/old-path1\">Moved Permanently</a>.\n\n", w.Body.String())
}

func TestContext_Blob(t *testing.T) {
	ris := assert.New(t)
	r := New()

	r.GET("/blob", func(c *Context) {
		c.Blob(200, "text/plain; charset=UTF-8", []byte("blob-test"))
	})

	w := mockRequest(r, GET, "/blob", nil)

	ris.Equal(200, w.Code)
	ris.Equal("text/plain; charset=UTF-8", w.Header().Get(ContentType))

	body, err := ioutil.ReadAll(w.Body)

	ris.NoError(err)
	ris.Equal(string(body), "blob-test")
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

func TestContext_Length(t *testing.T) {
	ris := assert.New(t)

	c := mockContext("GET", "/", nil, nil)
	c.WriteString("#length#")

	ris.Equal(8, c.Length())
}
