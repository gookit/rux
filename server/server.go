// Package server provides a production-ready HTTP server wrapping a rux.Router.
//
// The Server adds sensible timeouts, graceful shutdown, lifecycle hooks, and
// liveness/readiness endpoints on top of the bare router. Defaults are tuned
// for containerized deployments; override any field before calling Run.
package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gookit/color/colorp"
	"github.com/gookit/rux/v2"
	"github.com/gookit/rux/v2/pkg/handlers"
)

// Default configuration values applied by New. All durations are conservative
// enough to defend against common slow-attacks while remaining friendly to
// typical request/response cycles.
const (
	DefaultAddr              = ":8080"
	DefaultReadHeaderTimeout = 2 * time.Second
	DefaultReadTimeout       = 10 * time.Second
	DefaultWriteTimeout      = 30 * time.Second
	DefaultIdleTimeout       = 120 * time.Second
	DefaultMaxHeaderBytes    = 1 << 20 // 1 MiB
	DefaultShutdownTimeout   = 25 * time.Second
	DefaultDrainDelay        = 5 * time.Second
)

// Server is a production-ready HTTP server wrapping a rux.Router.
// Defaults are sane for containerized deployments; override fields before Run().
type Server struct {
	*rux.Router

	// Addr is the listen address ("host:port" or ":port"). Defaults to ":8080".
	Addr string

	// Host is kept for backward compatibility; New does not populate it.
	// SetHostPort writes both Host/Port and Addr.
	Host string
	Port uint

	// Optional TLS. If both files are set, Start uses ListenAndServeTLS.
	TLSCertFile string
	TLSKeyFile  string

	// HTTP server timeouts. Zero means "use net/http default" (NOT recommended).
	ReadHeaderTimeout time.Duration // default: 2s   (slowloris defense)
	ReadTimeout       time.Duration // default: 10s
	WriteTimeout      time.Duration // default: 30s
	IdleTimeout       time.Duration // default: 120s
	MaxHeaderBytes    int           // default: 1 << 20 (1 MiB)

	// ShutdownTimeout bounds how long Shutdown waits for in-flight requests.
	ShutdownTimeout time.Duration // default: 25s

	// DrainDelay is how long after receiving a stop signal Run keeps serving
	// before calling Shutdown. During drain, /readyz reports 503 so the
	// upstream LB can drain traffic. Set to 0 to skip drain.
	DrainDelay time.Duration // default: 5s

	// StopSignals received by Run trigger graceful shutdown.
	StopSignals []os.Signal // default: SIGINT, SIGTERM

	// Lifecycle hooks. Hooks of the same kind run in slice order.
	// PreStart errors abort startup; other hook errors are logged.
	PreStart     []func(ctx context.Context) error
	PostStart    []func(ctx context.Context) error
	PreShutdown  []func(ctx context.Context) error
	PostShutdown []func(ctx context.Context) error

	// ReadyChecks evaluate /readyz. /readyz returns 503 if any returns error
	// OR if the server is draining. Liveness (/healthz) is independent — it's
	// 200 as long as the process is alive. Use MountHealthChecks() to attach.
	ReadyChecks []func(ctx context.Context) error

	// Logger receives lifecycle and error messages. Defaults to log.Printf.
	Logger func(format string, args ...any)

	// Internal state.
	httpServer *http.Server
	httpMu     sync.Mutex  // guards httpServer
	err        error       // last lifecycle error (returned by Err)
	ready      atomic.Bool // true between PostStart and PreShutdown drain start
	draining   atomic.Bool // true during DrainDelay between signal and Shutdown
	shutdown   atomic.Bool // true once Shutdown has been initiated

	// stopCh is closed by Stop() to trigger graceful shutdown from Run()
	// without relying on OS signals. Useful for tests on platforms where
	// process self-signalling is not supported (e.g. Windows).
	stopCh   chan struct{}
	stopOnce sync.Once
}

// New constructs a Server with sane defaults plus PanicsHandler middleware.
// When debugMode is true, also installs RequestLogger and enables rux.Debug.
func New(debugMode bool) *Server {
	rux.Debug(debugMode)
	r := rux.New()

	r.Use(handlers.PanicsHandler())
	if debugMode {
		r.Use(handlers.RequestLogger())
	}

	// Default error handler — keeps backward compatibility with the v1 stub.
	r.OnError = func(c *rux.Context) {
		if err := c.FirstError(); err != nil {
			colorp.Errorln(err)
			c.HTTPError(err.Error(), 400)
			return
		}
	}

	return &Server{
		Router:            r,
		Addr:              DefaultAddr,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		ReadTimeout:       DefaultReadTimeout,
		WriteTimeout:      DefaultWriteTimeout,
		IdleTimeout:       DefaultIdleTimeout,
		MaxHeaderBytes:    DefaultMaxHeaderBytes,
		ShutdownTimeout:   DefaultShutdownTimeout,
		DrainDelay:        DefaultDrainDelay,
		StopSignals:       []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		Logger:            log.Printf,
	}
}

// SetAddr is a convenience setter that composes host:port into Addr.
func (s *Server) SetAddr(host string, port uint) {
	s.Host = host
	s.Port = port
	s.Addr = host + ":" + strconv.FormatUint(uint64(port), 10)
}

// SetHostPort is kept for backward compatibility — same as SetAddr.
func (s *Server) SetHostPort(host string, port uint) { s.SetAddr(host, port) }

// String returns the server's listen address for logging.
func (s *Server) String() string {
	s.httpMu.Lock()
	addr := s.Addr
	host := s.Host
	port := s.Port
	s.httpMu.Unlock()
	if addr != "" {
		return addr
	}
	if port > 0 {
		return fmt.Sprintf("%s:%d", host, port)
	}
	return host
}

// Err returns the last lifecycle error captured. Same value as Run's return.
func (s *Server) Err() error {
	s.httpMu.Lock()
	defer s.httpMu.Unlock()
	return s.err
}

// buildHTTPServer materializes the underlying *http.Server using current
// Server field values. Falls back to DefaultAddr if Addr is empty so a
// zero-value Server (e.g. constructed via struct literal in echo_server.go)
// still works.
func (s *Server) buildHTTPServer() *http.Server {
	addr := s.Addr
	if addr == "" {
		addr = DefaultAddr
	}
	maxHeader := s.MaxHeaderBytes
	if maxHeader == 0 {
		maxHeader = DefaultMaxHeaderBytes
	}
	return &http.Server{
		Addr:              addr,
		Handler:           s.Router,
		ReadHeaderTimeout: s.ReadHeaderTimeout,
		ReadTimeout:       s.ReadTimeout,
		WriteTimeout:      s.WriteTimeout,
		IdleTimeout:       s.IdleTimeout,
		MaxHeaderBytes:    maxHeader,
	}
}

// Start begins serving in the current goroutine. Blocks until the underlying
// http.Server.Serve returns (typically only after Shutdown).
//
// Returns http.ErrServerClosed on clean shutdown, or another error on failure.
//
// Note: Start does NOT call any hooks. Use Run() for the full lifecycle.
func (s *Server) Start() error {
	s.httpMu.Lock()
	srv := s.buildHTTPServer()
	s.httpServer = srv
	s.httpMu.Unlock()

	// Bind explicitly so callers can read back the actual port when Addr ends
	// in ":0" (commonly used in tests).
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		s.setErr(err)
		return err
	}
	// Reflect the resolved address back into the server so String() and tests
	// can observe the chosen port.
	s.httpMu.Lock()
	srv.Addr = ln.Addr().String()
	s.Addr = srv.Addr
	cert, key := s.TLSCertFile, s.TLSKeyFile
	s.httpMu.Unlock()

	if cert != "" && key != "" {
		err = srv.ServeTLS(ln, cert, key)
	} else {
		err = srv.Serve(ln)
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.setErr(err)
	}
	return err
}

// setErr safely records the last lifecycle error.
func (s *Server) setErr(err error) {
	s.httpMu.Lock()
	s.err = err
	s.httpMu.Unlock()
}

// Shutdown gracefully stops the server, bounded by ShutdownTimeout if ctx has
// no deadline. Idempotent. Does NOT call any hooks; use Run() for full lifecycle.
func (s *Server) Shutdown(ctx context.Context) error {
	if !s.shutdown.CompareAndSwap(false, true) {
		return nil
	}
	s.httpMu.Lock()
	srv := s.httpServer
	s.httpMu.Unlock()
	if srv == nil {
		return nil
	}

	// If caller's ctx has no deadline, apply ShutdownTimeout.
	if _, ok := ctx.Deadline(); !ok {
		timeout := s.ShutdownTimeout
		if timeout <= 0 {
			timeout = DefaultShutdownTimeout
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	if err := srv.Shutdown(ctx); err != nil {
		s.setErr(err)
		return err
	}
	return nil
}

// Stop requests a graceful shutdown from a running Run() loop. Safe to call
// from any goroutine and any number of times. Useful when the caller wants to
// trigger shutdown without sending an OS signal (tests, programmatic stop).
func (s *Server) Stop() {
	s.httpMu.Lock()
	ch := s.stopCh
	once := &s.stopOnce
	s.httpMu.Unlock()
	if ch == nil {
		return
	}
	once.Do(func() { close(ch) })
}

// Run is the one-shot lifecycle: PreStart → Start (background) → PostStart →
// wait for stop signal → PreShutdown → drain → Shutdown → PostShutdown.
//
// Returns:
//   - the first PreStart error (startup aborted before listening), OR
//   - the Start error (if Start returned anything other than http.ErrServerClosed), OR
//   - the Shutdown error, OR
//   - nil on clean shutdown.
//
// Hook errors after PreStart are logged but do not affect the return value.
func (s *Server) Run() error {
	// Reset transient state so Run is idempotent across calls.
	s.httpMu.Lock()
	s.err = nil
	s.stopCh = make(chan struct{})
	s.stopOnce = sync.Once{}
	s.httpMu.Unlock()
	s.shutdown.Store(false)
	s.ready.Store(false)
	s.draining.Store(false)

	bg := context.Background()

	// 1) PreStart hooks — first error aborts startup.
	if err := runHooks(bg, s.PreStart, true); err != nil {
		s.setErr(err)
		return err
	}

	// 2) Start in a background goroutine.
	startErrCh := make(chan error, 1)
	go func() { startErrCh <- s.Start() }()

	// Give Start a moment to bind so PostStart hooks see a usable Addr.
	// We wait for either: a Start error, or a brief delay for net.Listen.
	select {
	case err := <-startErrCh:
		// Start failed before serving.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.setErr(err)
			return err
		}
		// Closed before we got going — treat as clean.
		s.runHooksLogged(bg, s.PostShutdown)
		return nil
	case <-time.After(50 * time.Millisecond):
		// Listener should be bound by now in the common case.
	}

	// 3) PostStart hooks (errors logged, not fatal).
	s.runHooksLogged(bg, s.PostStart)
	s.ready.Store(true)
	s.logf("server listening on %s", s.String())

	// In debug mode, dump the registered routes so the operator sees
	// the route table without having to hit /__routes or similar.
	if rux.IsDebug() {
		s.logf("registered routes:\n%s", s.Router.String())
	}

	// 4) Wait for either a Start error, a stop signal, or a Stop() call.
	sigCh := newSignalChan(s.StopSignals)
	defer stopSignalChan(sigCh)

	var startErr error
	select {
	case startErr = <-startErrCh:
		// Server exited on its own (rare unless ListenAndServe failed mid-flight).
		s.ready.Store(false)
	case sig := <-sigCh:
		s.logf("received signal %s, beginning graceful shutdown", sig)
		s.draining.Store(true)
		s.ready.Store(false)
		if s.DrainDelay > 0 {
			time.Sleep(s.DrainDelay)
		}
	case <-s.stopCh:
		s.logf("Stop() called, beginning graceful shutdown")
		s.draining.Store(true)
		s.ready.Store(false)
		if s.DrainDelay > 0 {
			time.Sleep(s.DrainDelay)
		}
	}

	// 5) PreShutdown hooks (errors logged).
	s.runHooksLogged(bg, s.PreShutdown)

	// 6) Graceful shutdown.
	shutdownErr := s.Shutdown(bg)
	// Drain Start goroutine if we initiated shutdown via signal.
	if startErr == nil {
		select {
		case startErr = <-startErrCh:
		case <-time.After(time.Second):
		}
	}

	// 7) PostShutdown hooks (errors logged).
	s.runHooksLogged(bg, s.PostShutdown)

	// 8) Determine return value with documented priority.
	if startErr != nil && !errors.Is(startErr, http.ErrServerClosed) {
		s.setErr(startErr)
		return startErr
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	return nil
}

// logf is a nil-safe logger wrapper.
func (s *Server) logf(format string, args ...any) {
	if s.Logger != nil {
		s.Logger(format, args...)
		return
	}
	log.Printf(format, args...)
}
