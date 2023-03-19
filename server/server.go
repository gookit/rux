package server

import (
	"github.com/gookit/rux"
)

// Server struct
type Server struct {
	rux.Router
	// server start error
	err  error
	Host string
	Port int
}

// MustRun server
func (s *Server) MustRun(addr string) {
	s.Listen(addr)
}
