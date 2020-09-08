package rux

import (
	"fmt"
	"net/http"
	"runtime"
	"testing"

	"github.com/gookit/goutil/netutil/httpctype"
	"github.com/gookit/goutil/testutil"
	"github.com/stretchr/testify/assert"
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

	if runtime.GOOS == "windows" {
		return
	}

	// httptest.NewServer().Start()
	testutil.RewriteStdout()
	r.Listen("invalid]")
	r.Listen(":invalid]")
	r.Listen("127.0.0.1:invalid]")
	r.ListenTLS("invalid]", "", "")
	r.ListenUnix("")
	r.ListenUnix("/not-exit-file")

	mockEnvValue("PORT", "invalid]", func() {
		r.Listen()
	})

	s := testutil.RestoreStdout()

	is.Contains(s, "[ERROR] listen tcp: address 0.0.0.0:invalid]")
	is.Contains(s, "[ERROR] listen tcp: address 127.0.0.1:invalid]")
	fmt.Println(s)

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
		is.False(c.IsGet())
		is.True(c.IsPost())
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

	// add middleware
	route.Use(func(c *Context) { // middle 1
		// -> STEP 1:
		is.NotEmpty(c.Handler())
		is.NotEmpty(c.Router())
		is.NotEmpty(c.Copy())
		is.False(c.IsWebSocket())
		is.False(c.IsAjax())
		is.False(c.IsPost())
		is.True(c.IsGet())
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
		is.Nil(c.Value("not-exists"))

		_, ok = c.Value(nil).(*http.Request)
		is.True(ok)

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
	is.Equal(httpctype.Text, w.Header().Get(httpctype.Key))

	uri = "/HTML"
	r.GET(uri, func(c *Context) {
		c.HTML(200, []byte("html"))
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(200, w.Code)
	is.Equal(httpctype.HTML, w.Header().Get(httpctype.Key))
	is.Equal(`html`, w.Body.String())

	uri = "/JSON"
	r.GET(uri, func(c *Context) {
		c.JSON(200, M{"name": "inhere"})
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(200, w.Code)
	is.Equal(httpctype.JSON, w.Header().Get(httpctype.Key))
	is.Equal("{\"name\":\"inhere\"}\n", w.Body.String())

	uri = "/JSONBytes"
	r.GET(uri, func(c *Context) {
		c.JSONBytes(200, []byte(`{"name": "inhere"}`))
	})
	w = mockRequest(r, GET, uri, nil)
	is.Equal(200, w.Code)
	is.Equal(httpctype.JSON, w.Header().Get(httpctype.Key))
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

func TestRepeatSetStatusCode(t *testing.T) {
	rux := New()
	is := assert.New(t)

	rux.GET("/test-status-code", func(c *Context) {
		c.SetStatusCode(200)
		c.SetStatusCode(201)
		_, _ = c.Resp.Write([]byte("hi"))
	})

	w := mockRequest(rux, GET, "/test-status-code", nil)
	is.Equal(201, w.Code)
}

func TestHandleError(t *testing.T) {
	r := New()
	is := assert.New(t)

	r.OnError = func(c *Context) {
		is.Error(c.FirstError())
		is.Equal("oo, has an error", c.FirstError().Error())
	}

	r.GET("/test-error", func(c *Context) {
		c.AddError(fmt.Errorf("oo, has an error"))
		c.SetStatusCode(200)
	})

	w := mockRequest(r, GET, "/test-error", nil)
	is.Equal(200, w.Code)
}

func TestHandlePanic(t *testing.T) {
	is := assert.New(t)

	r := New()
	r.OnPanic = func(c *Context) {
		err, ok := c.Get(CTXRecoverResult)
		is.True(ok)
		is.Equal("panic test", err)
	}

	r.GET("/test-panic", func(c *Context) {
		panic("panic test")
	})

	w := mockRequest(r, GET, "/test-panic", nil)
	is.Equal(200, w.Code)
}

func TestHandleIsPost(t *testing.T) {
	is := assert.New(t)

	r := New()
	r.Add("/test-is-post", func(c *Context) {
		if c.IsPost() {
			c.HTML(200, []byte("method is post"))
			return
		}

		c.NoContent()
	}, GET, POST)

	w := mockRequest(r, GET, "/test-is-post", nil)
	is.Equal(204, w.Code)
	w = mockRequest(r, POST, "/test-is-post", nil)
	is.Equal(200, w.Code)
}

func TestHandleBack(t *testing.T) {
	is := assert.New(t)

	r := New()

	r.GET("/test-back", func(c *Context) {
		c.Back()
	})

	r.GET("/test-back-301", func(c *Context) {
		c.Back(301)
	})

	w := mockRequest(r, GET, "/test-back", nil)
	is.Equal(302, w.Code)
	w = mockRequest(r, GET, "/test-back-301", nil)
	is.Equal(301, w.Code)
}

func TestHandleBinder(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.GET("/test-binder", func(c *Context) {
		var form struct {
		}

		if err := c.Bind(&form); err != nil {
			is.Equal(err.Error(), "binder not registered")
		}
	})

	mockRequest(r, GET, "/test-binder", nil)
}

func TestHandleRender(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.GET("/test-render", func(c *Context) {
		if err := c.Render(200, "", nil); err != nil {
			is.Equal(err.Error(), "renderer not registered")
		}
	})

	mockRequest(r, GET, "/test-render", nil)
}

func TestHandleValidate(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.GET("/test-validate", func(c *Context) {
		var form struct{}

		if err := c.Validate(&form); err != nil {
			is.Equal(err.Error(), "validator not registered")
		}
	})

	mockRequest(r, GET, "/test-validate", nil)
}

func TestHandleXML(t *testing.T) {
	is := assert.New(t)
	r := New()

	type User struct {
		Name string
	}

	u := &User{
		Name: "test",
	}

	r.GET("/test-xml", func(c *Context) {
		c.XML(200, u)
	})
	r.GET("/test-xml2", func(c *Context) {
		c.XML(200, u, "  ")
	})

	w := mockRequest(r, GET, "/test-xml", nil)
	is.Equal(httpctype.XML, w.Header().Get("Content-Type"))
	is.Equal(`<?xml version="1.0" encoding="UTF-8"?>
<User><Name>test</Name></User>`, w.Body.String())

	w = mockRequest(r, GET, "/test-xml2", nil)
	is.Equal(httpctype.XML, w.Header().Get("Content-Type"))
	is.Equal(`<?xml version="1.0" encoding="UTF-8"?>
<User>
  <Name>test</Name>
</User>`, w.Body.String())
}

func TestHandleJSONP(t *testing.T) {
	is := assert.New(t)
	r := New()

	r.GET("/test-jsonp", func(c *Context) {
		type User struct {
			Name string
		}

		u := &User{
			Name: "test",
		}

		// or rux.M{"Name": "test"}
		c.JSONP(200, "jquery-jsonp", &u)
	})

	w := mockRequest(r, GET, "/test-jsonp", nil)
	is.Equal(httpctype.JSONP, w.Header().Get(httpctype.Key))
	is.Equal(`jquery-jsonp({"Name":"test"}
);`, w.Body.String())
}
