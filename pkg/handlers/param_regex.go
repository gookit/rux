package handlers

import (
	"net/http"
	"regexp"

	"github.com/gookit/rux/v2"
)

// ParamRegex returns a route middleware that requires the named path parameter
// to match pattern. It replaces the v1 inline constraint syntax
// (`{id:\d+}`), which v2 removed:
//
//	r.GET("/users/{id}", showUser, handlers.ParamRegex("id", `\d+`))
//
// The pattern is matched against the whole parameter value: it is compiled as
// `^(?:pattern)$`, so `\d+` rejects "12abc". Writing the anchors yourself
// (`^\d+$`) behaves the same, so v1 patterns port over unchanged.
//
// A missing or non-matching parameter aborts the request with 400 and a short
// text body. The pattern is compiled once, when the middleware is built, so an
// invalid pattern panics at registration time, like route registration does.
//
// Use it per route, or on a group whose routes share the parameter:
//
//	r.Group("/users/{id}", func() {
//		r.GET("", showUser)
//		r.PUT("", updateUser)
//	}, handlers.ParamRegex("id", `\d+`))
//
// For a custom failure response (JSON body, different status), write a small
// middleware of your own instead.
func ParamRegex(name, pattern string) rux.HandlerFunc {
	re := regexp.MustCompile("^(?:" + pattern + ")$")

	return func(c *rux.Context) {
		if !re.MatchString(c.Param(name)) {
			c.AbortWithStatus(http.StatusBadRequest, "rux: invalid path param: "+name)
			return
		}
		c.Next()
	}
}
