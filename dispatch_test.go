package sux

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"runtime"
	"testing"
)

func ExampleRouter_ServeHTTP() {
	r := New()
	r.GET("/", func(c *Context) {
		c.Text(200, "hello")
	})
	r.GET("/users/{id}", func(c *Context) {
		c.Text(200, "hello")
	})
	r.POST("/post", func(c *Context) {
		c.Text(200, "hello")
	})

	r.Listen(":8080")
}

type aStr struct {
	str string
}

func (a *aStr) reset() {
	a.str = ""
}

func (a *aStr) set(s ...interface{}) {
	a.str = fmt.Sprint(s...)
}

func (a *aStr) append(s string) {
	a.str += s
}

func TestRouterListen(t *testing.T) {
	art := assert.New(t)
	r := New()

	// multi params
	art.Panics(func() {
		r.Listen(":8080", "9090")
	})

	if runtime.GOOS != "darwin" {
		return
	}

	discardStdout()
	art.Error(r.Listen("invalid]"))
	art.Error(r.Listen(":invalid]"))
	art.Error(r.Listen("127.0.0.1:invalid]"))
	art.Error(r.ListenTLS("invalid]", "", ""))
	art.Error(r.ListenUnix(""))
	os.Setenv("PORT", "invalid]")
	art.Error(r.Listen())
	restoreStdout()
}

func TestRouter_ServeHTTP(t *testing.T) {
	art := assert.New(t)

	r := New()
	s := &aStr{}

	// simple
	r.GET("/", func(c *Context) {
		s.set("ok")

		art.Equal(c.URL().Path, "/")
	})
	mockRequest(r, GET, "/", "")
	art.Equal("ok", s.str)

	// use Params
	r.GET("/users/{id}", func(c *Context) {
		s.set("id:" + c.Param("id"))
	})
	mockRequest(r, GET, "/users/23", "")
	art.Equal("id:23", s.str)
	mockRequest(r, GET, "/users/tom", "")
	art.Equal("id:tom", s.str)

	// not exist
	s.reset()
	mockRequest(r, GET, "/users", "")
	art.Equal("", s.str)

	// receive input data
	r.POST("/users", func(c *Context) {
		bd, _ := c.RawData()
		s.set("body:", string(bd))

		p := c.Query("page")
		if p != "" {
			s.append(",page=" + p)
		}
	})
	s.reset()
	mockRequest(r, POST, "/users", "data")
	art.Equal("body:data", s.str)
	s.reset()
	w := mockRequest(r, POST, "/users?page=2", "data")
	art.Equal("body:data,page=2", s.str)
	art.Equal(200, w.Status())

	// no handler for NotFound
	s.reset()
	w = mockRequest(r, GET, "/not-exist", "")
	art.Equal("", s.str)
	art.Equal(404, w.Status())

	// add not found handler
	r.NotFound(func(c *Context) {
		s.set("not-found")
	})
	w = mockRequest(r, GET, "/not-exist", "")
	art.Equal("not-found", s.str)
	art.Equal(200, w.Status())

	// enable handle method not allowed
	r = New(HandleMethodNotAllowed)
	r.GET("/users/{id}", emptyHandler)

	// no handler for NotAllowed
	s.reset()
	w = mockRequest(r, POST, "/users/21", "")
	art.Equal("", s.str)
	art.Equal(405, w.Status())
	art.Contains(w.Header().Get("allow"), "GET")

	// but allow OPTIONS request
	w = mockRequest(r, OPTIONS, "/users/21", "")
	art.Equal(200, w.Status())

	// add handler
	r.NotAllowed(func(c *Context) {
		s.set("not-allowed")
	})
	s.reset()
	mockRequest(r, POST, "/users/23", "")
	art.Equal("not-allowed", s.str)

	s.reset()
	mockRequest(r, OPTIONS, "/users/23", "")
	art.Equal("not-allowed", s.str)
}

func TestRouter_WrapHttpHandlers(t *testing.T) {
	r := New()
	art := assert.New(t)

	r.GET("/", func(c *Context) {
		c.WriteString("hello")
	})

	// create some http.Handler
	gh := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
			w.Write([]byte(",tom"))
			w.WriteHeader(503)
		})
	}
	gh1 := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("res: "))
			h.ServeHTTP(w, r)
		})
	}

	h := r.WrapHttpHandlers(gh, gh1)
	w := mockRequest(h, "GET", "/", "")
	art.Equal(503, w.Status())
	art.Equal("res: hello,tom", w.buf.String())
}

func TestContext(t *testing.T) {
	art := assert.New(t)
	r := New()

	route := r.GET("/ctx", namedHandler) // main handler
	route.Use(func(c *Context) {         // middle 1
		// -> STEP 1:
		art.NotEmpty(c.Handler())
		art.NotEmpty(c.Router())
		art.NotEmpty(c.Copy())
		art.False(c.IsWebSocket())
		art.False(c.IsAjax())
		art.True(c.IsMethod("GET"))
		art.Equal("github.com/gookit/sux.namedHandler", c.HandlerName())
		// set a new context data
		c.Set("newKey", "val")

		c.Next()

		// STEP 4 ->:
		art.Equal("namedHandler1", c.Get("name").(string))
	}, func(c *Context) { // middle 2
		// -> STEP 2:
		_, ok := c.Values()["newKey"]
		art.True(ok)
		art.Equal("val", c.Get("newKey").(string))

		c.Next()

		// STEP 3 ->:
		art.Equal("namedHandler", c.Get("name").(string))
		c.Set("name", "namedHandler1") // change value
	})

	// Call sequence: middle 1 -> middle 2 -> main handler -> middle 2 -> middle 1
	mockRequest(r, GET, "/ctx", "data")

	r.GET("/ws", func(c *Context) {
		art.True(c.IsWebSocket())
	})
	requestWithData(r, GET, "/ws", &mockData{Heads: m{
		"Connection": "upgrade",
		"Upgrade":    "websocket",
	}})
}

func TestContext_ClientIP(t *testing.T) {
	art := assert.New(t)
	r := New()

	uri := "/ClientIP"
	r.GET(uri, func(c *Context) {
		c.WriteString(c.ClientIP())
	})
	w := requestWithData(r, GET, uri, &mockData{Heads: m{"X-Forwarded-For": "127.0.0.1"}})
	art.Equal(200, w.Status())
	art.Equal("127.0.0.1", w.buf.String())

	uri = "/ClientIP1"
	r.GET(uri, func(c *Context) {
		c.WriteString(c.ClientIP())
	})
	w = requestWithData(r, GET, uri, &mockData{Heads: m{"X-Forwarded-For": "127.0.0.2,localhost"}})
	art.Equal(200, w.Status())
	art.Equal("127.0.0.2", w.buf.String())

	uri = "/ClientIP2"
	r.GET(uri, func(c *Context) {
		c.WriteString(c.ClientIP())
	})
	w = requestWithData(r, GET, uri, &mockData{Heads: m{"X-Real-Ip": "127.0.0.3"}})
	art.Equal(200, w.Status())
	art.Equal("127.0.0.3", w.buf.String())

	uri = "/ClientIP3"
	r.GET(uri, func(c *Context) {
		c.WriteString(c.ClientIP())
	})
	w = requestWithData(r, GET, uri, &mockData{Heads: m{}})
	art.Equal(200, w.Status())
	art.Equal("", w.buf.String())
}

func TestContext_Write(t *testing.T) {
	art := assert.New(t)
	r := New()

	uri := "/Write"
	r.GET(uri, func(c *Context) {
		c.Write([]byte("hello"))
	})
	w := mockRequest(r, GET, uri, "data")
	art.Equal(200, w.Status())
	art.Equal("hello", w.buf.String())

	uri = "/WriteString"
	r.GET(uri, func(c *Context) {
		c.WriteString("hello")
	})
	w = mockRequest(r, GET, uri, "data")
	art.Equal(200, w.Status())
	art.Equal("hello", w.buf.String())

	uri = "/Text"
	r.GET(uri, func(c *Context) {
		c.Text(200, "hello")
	})
	w = mockRequest(r, GET, uri, "data")
	art.Equal(200, w.Status())
	art.Equal("hello", w.buf.String())
	art.Equal("text/plain; charset=UTF-8", w.Header().Get("content-type"))

	uri = "/JSONBytes"
	r.GET(uri, func(c *Context) {
		c.JSONBytes(200, []byte(`{"name": "inhere"}`))
	})
	w = mockRequest(r, GET, uri, "data")
	art.Equal(200, w.Status())
	art.Equal("application/json; charset=UTF-8", w.Header().Get("content-type"))
	art.Equal(`{"name": "inhere"}`, w.buf.String())

	uri = "/NoContent"
	r.GET(uri, func(c *Context) {
		c.NoContent()
	})
	w = mockRequest(r, GET, uri, "")
	art.Equal(204, w.Status())

	uri = "/SetHeader"
	r.GET(uri, func(c *Context) {
		c.SetHeader("new-key", "val")
	})
	w = mockRequest(r, GET, uri, "")
	art.Equal(200, w.Status())
	art.Equal("val", w.Header().Get("new-key"))

	uri = "/SetStatus"
	r.GET(uri, func(c *Context) {
		c.SetStatus(504)
	})
	w = mockRequest(r, GET, uri, "")
	art.Equal(504, w.Status())
}
