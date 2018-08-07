package middleware

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gookit/sux"
	"time"
)

// FavIcon uri for favicon.ico
const FavIcon = "/favicon.ico"

// SkipFavIcon middleware
func SkipFavIcon() sux.HandlerFunc {
	return func(c *sux.Context) {
		if c.URL().Path == FavIcon {
			c.NoContent()
			c.Abort()
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

// PanicsHandler middleware
func PanicsHandler() sux.HandlerFunc {
	// if h.printStack {
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
