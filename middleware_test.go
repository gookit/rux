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

	w := mockRequest(r, GET, "/middle", "")
	art.Equal("a-O-b", w.String())

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
	w = mockRequest(r, GET, "/middle2", "")
	art.Equal("ab-O-BA", w.String())

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
	w = mockRequest(r, GET, "/middle3", "")
	art.Equal("aAbB-O-", w.String())

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
	w = mockRequest(r, GET, "/middle4", "")
	art.Equal("ab-O-BA", w.String())
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
	w := mockRequest(r, GET, "/abort", "")
	art.Equal("aA", w.String())

	// use middleware, will termination execution early by AbortThen()
	r.GET("/abort1", func(c *Context) { // Will not execute
		c.WriteString("-O-")
	}, func(c *Context) {
		c.WriteString("a")
		// c.Next()
		// Will abort at the end of this middleware run
		c.AbortThen().Redirect("/other")
		c.WriteString("A")
	}, func(c *Context) { // Will not execute
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})
	// Call sequence: middle 1
	w = mockRequest(r, GET, "/abort1", "")
	art.NotEqual("aA", w.String())
	art.Equal(301, w.Status())
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
	w := mockRequest(r, GET, "/middle", "")
	art.Equal("z-O-Z", w.String())

	// eg2: global + route middle
	r.GET("/middle1", func(c *Context) { // main handler
		c.WriteString("-O-")
	}).Use(func(c *Context) {
		c.WriteString("b")
		c.Next()
		c.WriteString("B")
	})
	w = mockRequest(r, GET, "/middle1", "")
	art.Equal("zb-O-BZ", w.String())

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
	w = mockRequest(r, GET, "/grp/middle", "")
	art.Equal("zb-O-BZ", w.String())
	w = mockRequest(r, GET, "/grp/middle1", "")
	art.Equal("zbc-O-CBZ", w.String())

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
	w := mockRequest(r, GET, "/grp/middle", "")
	art.Equal("zy-O-YZ", w.String())
	w = mockRequest(r, GET, "/grp/middle1", "")
	art.Equal("zya-O-AYZ", w.String())
	w = mockRequest(r, GET, "/grp/sub-grp/middle", "")
	art.Equal("zyx-O-XYZ", w.String())
	w = mockRequest(r, GET, "/grp/sub-grp/middle1", "")
	art.Equal("zyxa-O-AXYZ", w.String())
}

func TestWarpHttpHandler(t *testing.T) {
	r := New()
	art := assert.New(t)
	gh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("he"))
	})

	r.GET("/path", func(c *Context) {
		c.WriteString("o")
	}).Use(
		WarpHttpHandler(gh),
		WarpHttpHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ll"))
		})))
	w := mockRequest(r, GET, "/path", "")
	art.Equal("hello", w.String())

	r.GET("/path1", func(c *Context) {
		c.WriteString("o")
	}).Use(
		WarpHttpHandlerFunc(gh),
		WarpHttpHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ll"))
		}))
	w = mockRequest(r, GET, "/path1", "")
	art.Equal("hello", w.String())
}
