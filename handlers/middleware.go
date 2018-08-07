package handlers

import "github.com/gookit/sux"

// FavIcon uri for favicon.ico
const FavIcon = "/favicon.ico"

// RequestLogger middleware
func RequestLogger(c *sux.Context) {

	c.Next()

}

// SkipFavIcon middleware
func SkipFavIcon(c *sux.Context) {
	c.NoContent()
}
