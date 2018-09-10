package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gookit/sux"
	"time"
)

// FavIcon uri for favicon.ico
const FavIcon = "/favicon.ico"

// IgnoreFavIcon middleware
func IgnoreFavIcon() sux.HandlerFunc {
	return func(c *sux.Context) {
		if c.URL().Path == FavIcon {
			c.AbortThen().NoContent()
			return
		}
	}
}

// GenRequestID for the request
func GenRequestID() sux.HandlerFunc {
	return func(c *sux.Context) {
		reqId := genMd5(fmt.Sprintf("sux-%d", time.Now().Nanosecond()))
		// add reqID to context
		c.Set("reqID", reqId)
	}
}

// HTTPBasicAuth for the request
func HTTPBasicAuth(users map[string]string) sux.HandlerFunc {
	return func(c *sux.Context) {
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
func PanicsHandler() sux.HandlerFunc {
	// if debug {
	// 	debug.PrintStack()
	// }

	return func(c *sux.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.Resp.WriteHeader(500)
			}
		}()

		c.Next()
	}
}

// genMd5 生成32位md5字串
func genMd5(s string) string {
	h := md5.New()
	h.Write([]byte(s))

	return hex.EncodeToString(h.Sum(nil))
}
