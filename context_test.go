package rux

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
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

	_ = c.ParseMultipartForm(8 << 20)

	val, has := c.PostParam("page")
	ris.True(has)
	ris.Equal("11", val)
	ris.Equal("11", c.Post("page"))
	ris.Equal("11", c.Post("page", "1"))

	ris.Equal([]string{"application/json"}, c.AcceptedTypes())
	ris.Equal("application/x-www-form-urlencoded", c.ContentType())
}

func TestContext_SetCookie(t *testing.T) {
	ris := assert.New(t)
	c := mockContext("GET", "/?both=v1", nil, m{
		"Accept":    "application/json",
		ContentType: "application/x-www-form-urlencoded",
	})

	c.SetCookie("ck-name", "val", 30, "/", "abc.com", true, true)

	s := c.Resp.Header().Get("Set-Cookie")
	ris.NotEmpty(s)
	ris.Contains(s, "ck-name=val")
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
