package server

import (
	"github.com/gookit/rux"
	"github.com/gookit/rux/render"
)

// NewEchoServer instance
func NewEchoServer() *Server {
	s := &Server{
		Router: *rux.New(),
	}

	s.Any("/{all}", func(c *rux.Context) {
		bs, err := c.RawBodyData()
		if err != nil {
			c.AbortThen().AddError(err)
			return
		}

		data := rux.M{
			"headers": c.Req.Header,
			"uri":     c.Req.RequestURI,
			"query":   c.QueryValues(),
			"body":    string(bs),
		}

		// c.JSON(200, data)
		c.Respond(200, data, render.NewJSONIndented())
	})

	return s
}
