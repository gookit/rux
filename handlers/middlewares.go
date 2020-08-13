package handlers

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

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
func GenRequestID() rux.HandlerFunc {
	return func(c *rux.Context) {
		reqID := genMd5(fmt.Sprintf("rux-%d", time.Now().Nanosecond()))
		// add reqID to context
		c.Set("reqID", reqID)
	}
}

// HTTPBasicAuth for the request
func HTTPBasicAuth(users map[string]string) rux.HandlerFunc {
	return func(c *rux.Context) {
		user, pwd, ok := c.Req.BasicAuth()
		if !ok {
			c.SetHeader("WWW-Authenticate", `Basic realm="THE REALM"`)
			c.AbortWithStatus(401, "Unauthorized")
			return
		}

		if len(users) > 0 {
			srcPwd, ok := users[user]
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
// the method is refer from "github.com/go-chi/chi/middleware"
//
// It's required that you select the ctx.Done() channel to check for the signal
// if the context has reached its deadline and return, otherwise the timeout
// signal will be just ignored.
//
// ie. a route/handler may look like:
//
//  r.GET("/long", func(c *rux.Context) {
// 	 ctx := c.Req.Context()
// 	 processTime := time.Duration(rand.Intn(4)+1) * time.Second
//
// 	 select {
// 	 case <-ctx.Done():
// 	 	return
//
// 	 case <-time.After(processTime):
// 	 	 // The above channel simulates some hard work.
// 	 }
//
// 	 c.WriteBytes([]byte("done"))
//  })
func Timeout(timeout time.Duration) rux.HandlerFunc  {
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

// genMd5 生成32位md5字串
func genMd5(s string) string {
	h := md5.New()
	h.Write([]byte(s))

	return hex.EncodeToString(h.Sum(nil))
}
