package rux

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gookit/goutil/netutil/httpctype"
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
		"Accept":      "application/json",
		httpctype.Key: "application/x-www-form-urlencoded",
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
	is := assert.New(t)

	c1 := mockContext("GET", "/test1?a=1&b=2&c=3", nil, nil)
	c2 := mockContext("GET", "/test2?a=1&b=2&c=3", nil, nil)

	var err error

	form1, err := c1.FormParams()

	is.NoError(err)

	form2, err := c2.FormParams([]string{"b"})

	is.NoError(err)

	is.Equal(form1.Encode(), "a=1&b=2&c=3")
	is.Equal(form2.Encode(), "a=1&c=3")

	// test parse multipart/form-data
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	err = mw.WriteField("kay0", "val0")
	is.NoError(err)
	err = mw.Close()
	is.NoError(err)

	c3 := mockContext("POST", "/", buf, m{
		"Content-Type": mw.FormDataContentType(),
	})

	f3, err := c3.FormParams()
	is.NoError(err)
	is.Equal("kay0=val0", f3.Encode())
}

func TestContext_SetCookie(t *testing.T) {
	is := assert.New(t)
	c := mockContext("GET", "/?both=v1", nil, m{
		"Accept":    "application/json",
		ContentType: "application/x-www-form-urlencoded",
	})

	c.SetCookie("ck-name", "val", 30, "/", "abc.com", true, true)
	c.SetCookie("ck-name1", "val1", 40, "", "abc.com", true, true)

	// Header().Get() will only return first
	s := c.Resp.Header().Get("Set-Cookie")
	is.NotEmpty(s)
	is.Contains(s, "ck-name=val")

	hs := c.Resp.Header()
	is.Contains(hs, "Set-Cookie")
	is.Len(hs["Set-Cookie"], 2)
	is.Contains(hs["Set-Cookie"][0], "ck-name=val")
	is.Contains(hs["Set-Cookie"][1], "ck-name1=val1")
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
	is := assert.New(t)

	r := New()
	r.GET("/test", func(c *Context) {
		val := c.Cookie("req-cke")
		is.Equal("req-val", val)

		val = c.Cookie("not-exist")
		is.Equal("", val)

		c.FastSetCookie("res-cke", "val1", 300)
	})
	r.GET("/delcookie", func(c *Context) {
		c.DelCookie("req-cke")
	})

	w := mockRequest(r, GET, "/test", nil, func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: "req-cke", Value: "req-val"})
	})

	is.Equal(200, w.Code)

	resCke := w.Header().Get("Set-Cookie")
	is.Equal("res-cke=val1; Path=/; Max-Age=300; HttpOnly", resCke)

	w = mockRequest(r, GET, "/delcookie", nil, func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: "req-cke", Value: "req-val"})
	})

	is.Equal(200, w.Code)

	resCke = w.Header().Get("Set-Cookie")
	is.Equal("req-cke=; Path=/; Max-Age=0; HttpOnly", resCke)
}

func TestContext_Length(t *testing.T) {
	ris := assert.New(t)

	c := mockContext("GET", "/", nil, nil)
	c.WriteString("#length#")

	ris.Equal(8, c.Length())
}
