package server

import (
	"fmt"

	"github.com/gookit/rux"
)

// Server struct
type Server struct {
	*rux.Router
	// server start error
	err error

	// Host server host. eg: localhost, 127.0.0.1
	Host string
	Port int
}

// New server instance
func New() *Server {
	return &Server{
		Router: rux.New(),
	}
}

// Start server with addr
func (s *Server) Start(addr string) {
	s.Listen(addr)
}

// String get string
func (s *Server) String() string {
	if s.Port > 0 {
		return fmt.Sprintf("%s:%d", s.Host, s.Port)
	}
	return s.Host
}
