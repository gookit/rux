package rux

import (
	"fmt"
	"github.com/gookit/goutil/testutil"
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
	is := assert.New(t)
	r := New()

	Debug(true)

	// multi params
	is.Panics(func() {
		r.Listen(":8080", "9090")
	})

	if runtime.GOOS != "darwin" {
		return
	}

	// httptest.NewServer().Start()
	testutil.RewriteStdout()
	r.Listen("invalid]")
	r.Listen(":invalid]")
	r.Listen("127.0.0.1:invalid]")
	r.ListenTLS("invalid]", "", "")
	r.ListenUnix("")
	_ = os.Setenv("PORT", "invalid]")
	r.Listen()
	s := testutil.RestoreStdout()

	is.Contains(s, "[ERROR] listen tcp: address 0.0.0.0:invalid]")
	is.Contains(s, "[ERROR] listen tcp: address 127.0.0.1:invalid]")

	Debug(false)
}

func TestRouter_ServeHTTP(t *testing.T) {
	is := assert.New(t)

	r := New()
	s := &aStr{}

	// simple
	r.GET("/", func(c *Context) {
		c.WriteString("ok")
		is.Equal(c.URL().Path, "/")
	})
	w := mockRequest(r, GET, "/", nil)
	is.Equal("ok", w.Body.String())

	// use Params
	r.GET("/users/{id}", func(c *Context) {
		s.set("id:" + c.Param("id"))
	})
	mockRequest(r, GET, "/users/23", nil)
	is.Equal("id:23", s.str)
	mockRequest(r, GET, "/users/tom", nil)
	is.Equal("id:tom", s.str)

	// not exist
	s.reset()
	mockRequest(r, GET, "/users", nil)
	is.Equal("", s.str)

	// receive input data
	r.POST("/users", func(c *Context) {
		bd, _ := c.RawBodyData()
		s.set("body:", string(bd))

		p := c.Query("page")
		if p != "" {
			s.append(",page=" + p)
		}

		n := c.Query("no-key", "defVal")
		is.Equal("defVal", n)
	})
	s.reset()
	mockRequest(r, POST, "/users", &md{B: "data"})
	is.Equal("body:data", s.str)
	s.reset()
	w = mockRequest(r, POST, "/users?page=2", &md{B: "data"})
	is.Equal("body:data,page=2", s.str)
	is.Equal(200, w.Code)

	// no handler for NotFound
	s.reset()
	w = mockRequest(r, GET, "/not-exist", nil)
	is.Equal("", s.str)
	is.Equal(404, w.Code)

	// add not found handler
	r.NotFound(func(c *Context) {
		s.set("not-found")
	})
	w = mockRequest(r, GET, "/not-exist", nil)
	is.Equal("not-found", s.str)
	is.Equal(200, w.Code)

	// enable handle method not allowed
	r = New(HandleMethodNotAllowed)
	r.GET("/users/{id}", emptyHandler)

	// no handler for NotAllowed
	s.reset()
	w = mockRequest(r, POST, "/users/21", nil)
	is.Equal("", s.str)
	is.Equal(405, w.Code)
	is.Contains(w.Header().Get("allow"), "GET")

	// but allow OPTIONS request
	w = mockRequest(r, OPTIONS, "/users/21", nil)
	is.Equal(200, w.Code)

	// add handler
	r.NotAllowed(func(c *Context) {
		s.set("not-allowed")
	})
	s.reset()
	mockRequest(r, POST, "/users/23", nil)
	is.Equal("not-allowed", s.str)

	s.reset()
	mockRequest(r, OPTIONS, "/users/23", nil)
	is.Equal("not-allowed", s.str)
}

func TestRouter_WrapHttpHandlers(t *testing.T) {
	r := New()
	is := assert.New(t)

	r.GET("/", func(c *Context) {
		c.WriteString("-O-")
	})

	// create some http.Handler
	gh := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(503)
			_, _ = w.Write([]byte("a"))
			h.ServeHTTP(w, r)
			_, _ = w.Write([]byte("d"))
		})
	}
	gh1 := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("b"))
			h.ServeHTTP(w, r)
			_, _ = w.Write([]byte("c"))
		})
	}

	h := r.WrapHTTPHandlers(gh, gh1)
	w := mockRequest(h, "GET", "/", nil)
	is.Equal(503, w.Code)
	is.Equal("ab-O-cd", w.Body.String())
}

func TestContext(t *testing.T) {
	is := assert.New(t)
	r := New()

	route := r.GET("/ctx", namedHandler) // main handler
	route.Use(func(c *Context) {         // middle 1
		// -> STEP 1:
		is.NotEmpty(c.Handler())
		is.NotEmpty(c.Router())
		is.NotEmpty(c.Copy())
		is.False(c.IsWebSocket())
		is.False(c.IsAjax())
		is.True(c.IsMethod("GET"))
		is.Equal("github.com/gookit/rux.namedHandler", c.HandlerName())
		// set a new context data
		c.Set("newKey", "val")

		c.Next()

		// STEP 4 ->:
		name, _ := c.Get("name")
		is.Equal("namedHandler1", name.(string))
	}, func(c *Context) { // middle 2
		// -> STEP 2:
		_, ok := c.Data()["newKey"]
		is.True(ok)
		is.Nil(c.Err())
		is.Equal("val", c.MustGet("newKey").(string))
		is.Equal("val", c.Value("newKey").(string))

		c.Next()

		// STEP 3 ->:
		is.Equal("namedHandler", c.MustGet("name").(string))
		c.Set("name", "namedHandler1") // change value
	})

	// Call sequence: middle 1 -> middle 2 -> main handler -> middle 2 -> middle 1
	mockRequest(r, GET, "/ctx", nil)

	r.GET("/ws", func(c *Context) {
		is.True(c.IsWebSocket())
	})
	mockRequest(r, GET, "/ws", &md{H: m{
		"Connection": "upgrade",
		"Upgrade":    "websocket",
	}})
}

func TestContext_ClientIP(t *testing.T) {
	is := assert.New(t)
	r := New()

	uri := "/ClientIP"
	r.GET(uri, func(c *Context) {
		c.WriteString(c.ClientIP())
	})
	w := mockRequest(r, GET, uri, &md{H: m{"X-Forwarded-For": "127.0.0.1"}})
	is.Equal(200, w.Code)
	is.Equal("127.0.0.1", w.Body.String())

	uri = "/ClientIP1"
	r.GET(uri, func(c *Context) {
		c.WriteString(c.ClientIP())
	})
	w = mockRequest(r, GET, uri, &md{H: m{"X-Forwarded-For": "127.0.0.2,localhost"}})
	is.Equal(200, w.Code)
	is.Equal("127.0.0.2", w.Body.String())

	uri = "/ClientIP2"
	r.GET(uri, func(c *Context) {
		c.WriteString(c.ClientIP())
	})
	w = mockRequest(r, GET, uri, &md{H: m{"X-Real-Ip": "127.0.0.3"}})
	is.Equal(200, w.Code)
	is.Equal("127.0.0.3", w.Body.String())

	uri = "/ClientIP3"
	r.GET(uri, func(c *Context) {
		c.WriteString(c.ClientIP())
	})
	w = mockRequest(r, GET, uri, &md{H: m{}})
	is.Equal(200, w.Code)
	is.Equal("", w.Body.String())
}

func TestContext_Write(t *testing.T) {
	is := assert.New(t)
	r := New()

	uri := "/Write"
	r.GET(uri, func(c *Context) {
		c.WriteBytes([]byte("hello"))
	})
	w := mockRequest(r, GET, uri, nil)
	is.Equal(200, w.Code)
	is.Equal("hello", w.Body.String())

	uri = "/WriteString"
	r.GET(uri, func(c *Context) {
		c.WriteString("hello")
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(200, w.Code)
	is.Equal("hello", w.Body.String())

	uri = "/Text"
	r.GET(uri, func(c *Context) {
		c.Text(200, "hello")
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(200, w.Code)
	is.Equal("hello", w.Body.String())
	is.Equal("text/plain; charset=UTF-8", w.Header().Get("content-type"))

	uri = "/HTML"
	r.GET(uri, func(c *Context) {
		c.HTML(200, []byte("html"))
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(200, w.Code)
	is.Equal("text/html; charset=UTF-8", w.Header().Get("content-type"))
	is.Equal(`html`, w.Body.String())

	uri = "/JSON"
	r.GET(uri, func(c *Context) {
		c.JSON(200, M{"name": "inhere"})
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(200, w.Code)
	is.Equal("application/json; charset=UTF-8", w.Header().Get("content-type"))
	is.Equal(`{"name":"inhere"}`, w.Body.String())

	uri = "/JSONBytes"
	r.GET(uri, func(c *Context) {
		c.JSONBytes(200, []byte(`{"name": "inhere"}`))
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(200, w.Code)
	is.Equal("application/json; charset=UTF-8", w.Header().Get("content-type"))
	is.Equal(`{"name": "inhere"}`, w.Body.String())

	uri = "/NoContent"
	r.GET(uri, func(c *Context) {
		c.NoContent()
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(204, w.Code)

	uri = "/HTTPError"
	r.GET(uri, func(c *Context) {
		c.HTTPError("error", 503)
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(503, w.Code)
	is.Equal("error\n", w.Body.String())

	uri = "/SetHeader"
	r.GET(uri, func(c *Context) {
		c.SetHeader("new-key", "val")
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(200, w.Code)
	is.Equal("val", w.Header().Get("new-key"))

	uri = "/SetStatus"
	r.GET(uri, func(c *Context) {
		c.SetStatus(504)
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(504, w.Code)
}

func TestContext_Cookie(t *testing.T) {
	ris := assert.New(t)

	r := New()
	r.GET("/test", func(c *Context) {
		val, err := c.Cookie("req-cke")
		ris.Nil(err)
		ris.Equal("req-val", val)

		c.FastSetCookie("res-cke", "val1", 300)
	})

	w := mockRequest(r, GET, "/test", nil, func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: "req-cke", Value: "req-val"})
	})

	ris.Equal(200, w.Code)

	resCke := w.Header().Get("Set-Cookie")
	ris.Equal("res-cke=val1; Path=/; Max-Age=300; Secure", resCke)
}
