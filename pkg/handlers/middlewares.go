package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/rux"
)

// FavIcon uri for favicon.ico
const FavIcon = "/favicon.ico"

// IgnoreFavIcon middleware
func IgnoreFavIcon() rux.HandlerFunc {
	return func(c *rux.Context) {
		if c.URL().Path == FavIcon {
			c.AbortThen().NoContent()
			return
		}
	}
}

// GenRequestID for the request
func GenRequestID(key string) rux.HandlerFunc {
	return func(c *rux.Context) {
		now := time.Now()
		val := fmt.Sprintf("r%xq%d", now.UnixMicro(), mathutil.RandIntWithSeed(1000, 9999, int64(now.Nanosecond())))
		// add reqID to context
		c.Set(key, val)
	}
}

// HTTPBasicAuth for the request
//
// Usage:
//
//	r.GET("/auth", func(c *rux.Context) {
//		c.WriteString("hello")
//	}, HTTPBasicAuth(map[string]string{"testuser": "123"}))
func HTTPBasicAuth(accounts map[string]string) rux.HandlerFunc {
	return func(c *rux.Context) {
		user, pwd, ok := c.Req.BasicAuth()
		if !ok {
			c.SetHeader("WWW-Authenticate", `Basic realm="THE REALM"`)
			c.AbortWithStatus(401, "Unauthorized")
			return
		}

		if len(accounts) > 0 {
			srcPwd, ok := accounts[user]
			if !ok || srcPwd != pwd {
				c.AbortWithStatus(403)
			}
		}

		c.Set("username", user)
		c.Set("password", pwd)
	}
}

// PanicsHandler middleware
func PanicsHandler() rux.HandlerFunc {
	// if debug {
	// 	debug.PrintStack()
	// }

	return func(c *rux.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.Resp.WriteHeader(500)
			}
		}()

		c.Next()
	}
}

// Timeout is a middleware for handle logic.
// the method is referred from "github.com/go-chi/chi/middleware"
//
// It's required that you select the ctx.Done() channel to check for the signal
// if the context has reached its deadline and return, otherwise the timeout
// signal will be just ignored.
//
// a route/handler may look like:
//
//	 r.GET("/long", func(c *rux.Context) {
//		 ctx := c.Req.Context()
//		 processTime := time.Duration(rand.Intn(4)+1) * time.Second
//
//		 select {
//		 case <-ctx.Done():
//		 	return
//
//		 case <-time.After(processTime):
//		 	 // The above channel simulates some hard work.
//		 }
//
//		 c.WriteBytes([]byte("done"))
//	 })
func Timeout(timeout time.Duration) rux.HandlerFunc {
	return func(c *rux.Context) {
		ctx, cancel := context.WithTimeout(c.Req.Context(), timeout)
		defer func() {
			cancel()
			if ctx.Err() == context.DeadlineExceeded {
				c.Resp.WriteHeader(http.StatusGatewayTimeout)
			}
		}()

		// override Request
		c.Req = c.Req.WithContext(ctx)
		c.Next()
	}
}
