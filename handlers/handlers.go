package handlers

import "github.com/gookit/sux"

// DumpRoutesHandler
func DumpRoutesHandler() sux.HandlerFunc {
	return func(c *sux.Context) {
		c.Text(200, c.Router().String())
	}
}
