package server

import (
	"fmt"

	"github.com/gookit/color/colorp"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/rux"
	"github.com/gookit/rux/pkg/handlers"
)

// Server struct
type Server struct {
	*rux.Router
	// server start error
	err error

	// Host server host. eg: localhost, 127.0.0.1
	Host string
	Port uint
}

// New server instance
func New(debugMode bool) *Server {
	rux.Debug(debugMode)
	r := rux.New(rux.EnableCaching)

	r.Use(handlers.PanicsHandler())
	if debugMode {
		r.Use(handlers.RequestLogger())
	}

	// handle error
	r.OnError = func(c *rux.Context) {
		if err := c.FirstError(); err != nil {
			colorp.Errorln(err)
			c.HTTPError(err.Error(), 400)
			return
		}
	}

	return &Server{Router: r}
}

// SetHostPort set host and port
func (s *Server) SetHostPort(host string, port uint) {
	s.Host = host
	s.Port = port
}

// Start server with addr
func (s *Server) Start() {
	portStr := strutil.SafeString(s.Port)
	// addr := s.Host + ":" + portStr
	// s.Listen(addr)
	s.Listen(s.Host, portStr)
}

// String get string
func (s *Server) String() string {
	if s.Port > 0 {
		return fmt.Sprintf("%s:%d", s.Host, s.Port)
	}
	return s.Host
}
