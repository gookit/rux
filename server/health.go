package server

import (
	"fmt"
	"net/http"

	"github.com/gookit/rux/v2"
)

// MountHealthChecks attaches GET /healthz (liveness) and GET /readyz (readiness)
// to the router. Call this before Run() — the router freezes on first request.
//
// /healthz: always 200 "ok" if the process is alive.
// /readyz: 200 if ready && !draining && all ReadyChecks pass; else 503 with details.
func (s *Server) MountHealthChecks() {
	s.Router.GET("/healthz", livenessHandler)
	s.Router.GET("/readyz", s.readinessHandler)
}

// livenessHandler responds 200 OK as long as the process can handle requests.
// Kubernetes uses this to decide whether to restart the pod.
func livenessHandler(c *rux.Context) {
	c.Resp.WriteHeader(http.StatusOK)
	_, _ = c.Resp.Write([]byte("ok"))
}

// readinessHandler responds 200 only when the server has finished startup
// and all user-supplied ReadyChecks pass. Returns 503 during drain so the
// upstream load balancer stops sending new traffic.
func (s *Server) readinessHandler(c *rux.Context) {
	if s.draining.Load() {
		c.Resp.WriteHeader(http.StatusServiceUnavailable)
		_, _ = c.Resp.Write([]byte("draining"))
		return
	}
	if !s.ready.Load() {
		c.Resp.WriteHeader(http.StatusServiceUnavailable)
		_, _ = c.Resp.Write([]byte("not ready"))
		return
	}
	for i, check := range s.ReadyChecks {
		if check == nil {
			continue
		}
		if err := check(c.Req.Context()); err != nil {
			c.Resp.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(c.Resp, "check %d failed: %v", i, err)
			return
		}
	}
	c.Resp.WriteHeader(http.StatusOK)
	_, _ = c.Resp.Write([]byte("ready"))
}
