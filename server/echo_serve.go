package server

import (
	"github.com/gookit/goutil/testutil"
	"github.com/gookit/rux"
	"github.com/gookit/rux/pkg/render"
)

// NewEchoServer instance
func NewEchoServer() *Server {
	s := &Server{
		Router: rux.New(),
	}

	s.Any("/{all}", func(c *rux.Context) {
		data := testutil.BuildEchoReply(c.Req)

		// c.JSON(200, data)
		c.Respond(200, data, render.NewJSONIndented())
	})

	return s
}
