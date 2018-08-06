package middleware

import "github.com/gookit/sux"

// RequestLogger middleware
func RequestLogger(c *sux.Context) {

	c.Next()

}
