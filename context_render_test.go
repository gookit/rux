package rux

import (
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestContext_Redirect(t *testing.T) {
	is := assert.New(t)
	r := New()

	// Redirect()
	uri := "/Redirect"
	r.GET(uri, func(c *Context) {
		c.Redirect("/new-path")
	})
	w := mockRequest(r, GET, uri, nil)
	is.Eq(301, w.Code)
	is.Eq("/new-path", w.Header().Get("Location"))
	is.Eq("<a href=\"/new-path\">Moved Permanently</a>.\n\n", w.Body.String())

	uri = "/Redirect1"
	r.GET(uri, func(c *Context) {
		c.Redirect("/new-path1", 302)
	})
	w = mockRequest(r, GET, uri, nil)
	is.Eq(302, w.Code)
	is.Eq("/new-path1", w.Header().Get("Location"))
	is.Eq("<a href=\"/new-path1\">Found</a>.\n\n", w.Body.String())
}

func TestContext_Back(t *testing.T) {
	is := assert.New(t)
	r := New()

	// Back()
	uri := "/Back"
	r.GET(uri, func(c *Context) {
		c.Back()
	})
	w := mockRequest(r, GET, uri, &md{H: m{"Referer": "/old-path"}})
	is.Eq(302, w.Code)
	is.Eq("/old-path", w.Header().Get("Location"))
	is.Eq("<a href=\"/old-path\">Found</a>.\n\n", w.Body.String())

	// Back()
	uri = "/Back1"
	r.GET(uri, func(c *Context) {
		c.Back(301)
	})
	w = mockRequest(r, GET, uri, &md{H: m{"Referer": "/old-path1"}})
	is.Eq(301, w.Code)
	is.Eq("/old-path1", w.Header().Get("Location"))
	is.Eq("<a href=\"/old-path1\">Moved Permanently</a>.\n\n", w.Body.String())
}

func TestContext_Blob(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.GET("/blob", func(c *Context) {
		c.Blob(200, "text/plain; charset=UTF-8", []byte("blob-test"))
	})

	w := mockRequest(r, GET, "/blob", nil)

	is.Eq(200, w.Code)
	is.Eq("text/plain; charset=UTF-8", w.Header().Get(ContentType))

	body, err := ioutil.ReadAll(w.Body)

	is.NoErr(err)
	is.Eq(string(body), "blob-test")
}

func TestContext_Binary(t *testing.T) {
	is := assert.New(t)
	c := mockContext("GET", "/site.md", nil, nil)

	in, _ := os.Open("testdata/site.md")
	c.Binary(200, in, "new-name.md", true)

	w := c.RawWriter().(*httptest.ResponseRecorder)
	is.Eq(200, w.Code)
	ss, ok := w.Header()["Content-Type"]
	is.True(ok)
	is.Eq(ss[0], "application/octet-stream")

	ss, ok = w.Header()["Content-Disposition"]
	is.True(ok)
	is.Eq(ss[0], "inline; filename=new-name.md")
	is.Contains(w.Body.String(), "# readme")
}

func TestContext_FileContent(t *testing.T) {
	is := assert.New(t)

	c := mockContext("GET", "/site.md", nil, nil)
	c.FileContent("testdata/site.md", "new-name.md")

	w := c.RawWriter().(*httptest.ResponseRecorder)
	is.Eq(200, c.StatusCode())
	is.Eq(200, w.Code)

	ss, ok := w.Header()["Content-Type"]
	is.True(ok)
	// go 1.14.4 "text/markdown; charset=utf-8" does not contain "text/plain"
	is.Contains(ss[0], "text/")
	is.Contains(w.Body.String(), "# readme")
	is.True(8 < c.writer.Length())

	c = mockContext("GET", "/site.md", nil, nil)
	c.FileContent("testdata/not-exist.md")
	w = c.RawWriter().(*httptest.ResponseRecorder)
	is.Eq(500, c.StatusCode())
	is.Eq(500, w.Code)
	is.Eq("Internal Server Error\n", w.Body.String())
}

func TestContext_Attachment(t *testing.T) {
	is := assert.New(t)

	c := mockContext("GET", "/site.md", nil, nil)
	c.Attachment("testdata/site.md", "new-name.md")

	w := c.RawWriter().(*httptest.ResponseRecorder)
	is.Eq(200, w.Code)
	ss, ok := w.Header()["Content-Type"]
	is.True(ok)
	is.Eq(ss[0], "application/octet-stream")

	ss, ok = w.Header()["Content-Disposition"]
	is.True(ok)
	is.Eq(ss[0], "attachment; filename=new-name.md")
	is.Contains(w.Body.String(), "# readme")
}

func TestContext_Inline(t *testing.T) {
	is := assert.New(t)

	// Inline
	c := mockContext("GET", "/site.md", nil, nil)
	c.Inline("testdata/site.md", "new-name.md")

	w := c.RawWriter().(*httptest.ResponseRecorder)
	is.Eq(200, w.Code)
	ss, ok := w.Header()["Content-Type"]
	is.True(ok)
	is.Eq(ss[0], "application/octet-stream")

	ss, ok = w.Header()["Content-Disposition"]
	is.True(ok)
	is.Eq(ss[0], "inline; filename=new-name.md")
	is.Contains(w.Body.String(), "# readme")
}
