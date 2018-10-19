package sux

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestRouteMiddleware(t *testing.T) {
	r := New()
	art := assert.New(t)

	// add one middleware
	r.GET("/middle", func(c *Context) {
		c.WriteString("-O-")
	}, func(c *Context) {
		c.WriteString("a")
		c.Next()
		c.WriteString("b")
	})

	w := mockRequest(r, GET, "/middle", nil)
	art.Equal("a-O-b", w.Body.String())

	// add multi middleware
	r.GET("/middle2", func(c *Context) { // main handler
		c.WriteString("-O-")
	}, func(c *Context) { // middle 1
		c.WriteString("a")
		c.Next()
		c.WriteString("A")
	}, func(c *Context) { // middle 2
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})
	// Call sequence: middle 1 -> middle 2 -> main handler -> middle 2 -> middle 1
	w = mockRequest(r, GET, "/middle2", nil)
	art.Equal("ab-O-BA", w.Body.String())

	// add multi middleware(don't call next)
	r.GET("/middle3", func(c *Context) { // main handler
		c.WriteString("-O-")
	}, func(c *Context) { // middle 1
		c.WriteString("a")
		// c.Next()
		c.WriteString("A")
	}, func(c *Context) { // middle 2
		c.WriteString("b")
		// c.Next()
		c.WriteString("B")
	})
	// Call sequence: middle 1 -> middle 2 -> main handler
	w = mockRequest(r, GET, "/middle3", nil)
	art.Equal("aAbB-O-", w.Body.String())

	// add middleware use method Use()
	route := r.GET("/middle4", func(c *Context) { // main handler
		c.WriteString("-O-")
	})
	route.Use(func(c *Context) { // middle 1
		c.WriteString("a")
		c.Next()
		c.WriteString("A")
	}, func(c *Context) { // middle 2
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})
	// Call sequence: middle 1 -> middle 2 -> main handler -> middle 2 -> middle 1
	w = mockRequest(r, GET, "/middle4", nil)
	art.Equal("ab-O-BA", w.Body.String())
}

func TestContext_Abort(t *testing.T) {
	art := assert.New(t)
	r := New()

	// use middleware, will termination execution early by Abort()
	r.GET("/abort", func(c *Context) { // Will not execute
		c.WriteString("-O-")
	}, func(c *Context) {
		c.WriteString("a")
		// c.Next()
		c.Abort() // Will abort at the end of this middleware run
		c.WriteString("A")
	}, func(c *Context) { // Will not execute
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})
	// Call sequence: middle 1
	w := mockRequest(r, GET, "/abort", nil)
	art.Equal("aA", w.Body.String())

	// use middleware, will termination execution early by AbortThen()
	r.GET("/abort1", func(c *Context) { // Will not execute
		c.WriteString("-O-")
	}, func(c *Context) {
		// Will abort at the end of this middleware run
		c.AbortThen().Redirect("/other", 302)
		c.WriteString("a")
		// c.Next()
		c.WriteString("A")
	}, func(c *Context) { // Will not execute
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})
	// Call sequence: middle 1
	w = mockRequest(r, GET, "/abort1", nil)
	// body: <a href="/other">Found</a>.\n\naA
	art.NotEqual("aA", w.Body.String())
	art.Equal(302, w.Code)

	// use middleware, will termination execution early by AbortWithStatus()
	r.GET("/abort2", func(c *Context) { // Will not execute
		c.WriteString("-O-")
	}, func(c *Context) {
		// Will abort at the end of this middleware run
		c.AbortWithStatus(404)
		c.WriteString("a")
		// c.Next()
		c.WriteString("A")
	}, func(c *Context) { // Will not execute
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})
	// Call sequence: middle 1
	w = mockRequest(r, GET, "/abort2", nil)
	art.Equal("aA", w.Body.String())
	art.Equal(404, w.Code)
}

func TestGlobalMiddleware(t *testing.T) {
	art := assert.New(t)
	r := New()
	art.NotEmpty(r)

	r.Use(func(c *Context) {
		c.WriteString("z")
		c.Next()
		c.WriteString("Z")
	})

	// eg1: only global middles
	r.GET("/middle", func(c *Context) { // main handler
		c.WriteString("-O-")
	})
	w := mockRequest(r, GET, "/middle", nil)
	art.Equal("z-O-Z", w.Body.String())

	// eg2: global + route middle
	r.GET("/middle1", func(c *Context) { // main handler
		c.WriteString("-O-")
	}).Use(func(c *Context) {
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})
	w = mockRequest(r, GET, "/middle1", nil)
	art.Equal("zb-O-BZ", w.Body.String())

	r.Group("/grp", func() {
		// eg3: global + group middles
		r.GET("/middle", func(c *Context) { // main handler
			c.WriteString("-O-")
		})

		// eg4: global + group + route middles
		r.GET("/middle1", func(c *Context) { // main handler
			c.WriteString("-O-")
		}).Use(func(c *Context) {
			c.WriteString("c")
			c.Next()
			c.WriteString("C")
		})
	}, func(c *Context) {
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})
	w = mockRequest(r, GET, "/grp/middle", nil)
	art.Equal("zb-O-BZ", w.Body.String())
	w = mockRequest(r, GET, "/grp/middle1", nil)
	art.Equal("zbc-O-CBZ", w.Body.String())

}

func TestGroupMiddleware(t *testing.T) {
	art := assert.New(t)
	r := New()
	art.NotEmpty(r)

	r.Group("/grp", func() {
		r.GET("/middle", func(c *Context) { // main handler
			c.WriteString("-O-")
		})

		r.GET("/middle1", func(c *Context) { // main handler
			c.WriteString("-O-")
		}).Use(func(c *Context) {
			c.WriteString("a")
			c.Next()
			c.WriteString("A")
		})

		// multi group level
		r.Group("/sub-grp", func() {
			r.GET("/middle", func(c *Context) { // main handler
				c.WriteString("-O-")
			})
			r.GET("/middle1", func(c *Context) { // main handler
				c.WriteString("-O-")
			}).Use(func(c *Context) {
				c.WriteString("a")
				c.Next()
				c.WriteString("A")
			})
		}, func(c *Context) {
			c.WriteString("x")
			c.Next()
			c.WriteString("X")
		})
	}, func(c *Context) {
		c.WriteString("z")
		c.Next()
		c.WriteString("Z")
	}, func(c *Context) {
		c.WriteString("y")
		c.Next()
		c.WriteString("Y")
	})
	w := mockRequest(r, GET, "/grp/middle", nil)
	art.Equal("zy-O-YZ", w.Body.String())
	w = mockRequest(r, GET, "/grp/middle1", nil)
	art.Equal("zya-O-AYZ", w.Body.String())
	w = mockRequest(r, GET, "/grp/sub-grp/middle", nil)
	art.Equal("zyx-O-XYZ", w.Body.String())
	w = mockRequest(r, GET, "/grp/sub-grp/middle1", nil)
	art.Equal("zyxa-O-AXYZ", w.Body.String())
}

func TestWrapHTTPHandler(t *testing.T) {
	r := New()
	art := assert.New(t)
	gh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("he"))
	})

	r.GET("/path", func(c *Context) {
		c.WriteString("o")
	}).Use(
		WrapHTTPHandler(gh),
		WrapHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ll"))
		})))
	w := mockRequest(r, GET, "/path", nil)
	art.Equal("hello", w.Body.String())

	r.GET("/path1", func(c *Context) {
		c.WriteString("o")
	}).Use(
		WrapHTTPHandlerFunc(gh),
		WrapHTTPHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ll"))
		}))
	w = mockRequest(r, GET, "/path1", nil)
	art.Equal("hello", w.Body.String())
}
