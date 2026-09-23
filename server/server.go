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

	"github.com/gookit/goutil/x/ccolor"
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
	httpMu     sync.Mutex   // guards httpServer, ln, Addr/Host/Port and boundCh
	ln         net.Listener // listener currently serving; nil when not listening
	err        error        // last lifecycle error (returned by Err)
	listening  atomic.Bool  // true while a bound listener is serving
	ready      atomic.Bool  // true between PostStart and PreShutdown drain start
	draining   atomic.Bool  // true during DrainDelay between signal and Shutdown
	shutdown   atomic.Bool  // true once Shutdown has been initiated

	// boundCh is closed (and cleared) by Start as soon as the listener is
	// bound, so Run and WaitListening know the resolved address is final
	// instead of guessing how long net.Listen takes. nil means "already
	// signalled, or no run in progress".
	boundCh chan struct{}

	// stopCh is closed by Stop() to trigger graceful shutdown from Run()
	// without relying on OS signals. Useful for tests on platforms where
	// process self-signalling is not supported (e.g. Windows).
	stopCh   chan struct{}
	stopOnce sync.Once
}

// ErrNotListening is returned by WaitListening when there is no listener to
// wait for: startup aborted before binding, or a run already came and went, so
// the address can never become final in this run.
var ErrNotListening = errors.New("rux: server is not listening")

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
			ccolor.Errorln("Server error: ", err)
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
		// Non-nil so a WaitListening caller that starts before Run blocks
		// until the (first) run binds instead of returning immediately.
		boundCh: make(chan struct{}),
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

// ListenAddr returns the address the server listens on.
//
// Before the listener is bound it is the configured Addr (which may still
// carry port 0, i.e. "127.0.0.1:0"). Once bound, Start reflects the resolved
// listener address back here, so an OS-assigned port shows up as a real one
// ("127.0.0.1:50447", or "[::]:50447" / "0.0.0.0:50447" for wildcard binds).
// Use IsListening/WaitListening to know when the value is final, and LocalURL
// when you need a URL that is usable in a browser.
//
// Safe to call from any goroutine (unlike reading the Addr field directly).
func (s *Server) ListenAddr() string {
	s.httpMu.Lock()
	defer s.httpMu.Unlock()
	return s.Addr
}

// ListenPort returns the resolved TCP port of the listener. It returns 0 when
// the listener is not bound yet (including the ":0" request for an
// OS-assigned port) or when Addr carries no port.
func (s *Server) ListenPort() int {
	s.httpMu.Lock()
	addr := s.Addr
	legacyPort := s.Port
	s.httpMu.Unlock()

	if _, portStr, err := net.SplitHostPort(addr); err == nil {
		if p, err := strconv.Atoi(portStr); err == nil {
			return p
		}
	}
	return int(legacyPort)
}

// LocalURL returns a loopback URL for the bound listener, e.g.
// "http://127.0.0.1:50447". Wildcard hosts ("", "0.0.0.0", "::") are mapped
// onto 127.0.0.1 so the result can be handed straight to a browser or an
// open-in-browser flag. The scheme is "https" when TLS is configured.
//
// It returns "" while the port is not usable yet (no listener, or Addr still
// asking for port 0), which lets callers wait on WaitListening instead of
// opening a bogus URL.
func (s *Server) LocalURL() string {
	s.httpMu.Lock()
	addr := s.Addr
	tls := s.TLSCertFile != "" && s.TLSKeyFile != ""
	s.httpMu.Unlock()

	if addr == "" {
		return ""
	}

	scheme := "http"
	if tls {
		scheme = "https"
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// No port component at all (e.g. a Unix socket path): hand it back.
		return scheme + "://" + addr
	}
	if port == "" || port == "0" {
		return ""
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "0:0:0:0:0:0:0:0" {
		host = "127.0.0.1"
	}
	return scheme + "://" + net.JoinHostPort(host, port)
}

// IsListening reports whether a listener is currently bound and serving.
// It is false before Run/Start binds and again once Shutdown completes.
func (s *Server) IsListening() bool { return s.listening.Load() }

// WaitListening blocks until the listener is bound and its address is final
// (the port-0 / `--open` case), or until ctx is done. It returns:
//
//   - nil once the server is listening,
//   - the error that aborted startup, if the run failed before binding,
//   - ErrNotListening if there is no listener to wait for,
//   - ctx.Err() otherwise (e.g. context deadline exceeded).
//
// Waiting before Run is safe: it blocks until that run binds. Typical use:
//
//	go func() { _ = srv.Run() }()
//	if err := srv.WaitListening(ctx); err == nil {
//		openInBrowser(srv.LocalURL()) // e.g. http://127.0.0.1:50447
//	}
func (s *Server) WaitListening(ctx context.Context) error {
	for {
		if s.listening.Load() {
			return nil
		}
		if err := s.Err(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		s.httpMu.Lock()
		ch := s.boundCh
		s.httpMu.Unlock()
		if ch == nil {
			// The bind signal may have landed between our checks. Taking
			// httpMu here happens-after the signalling side (which stored
			// listening=true before clearing boundCh under the same mutex), so
			// this read is authoritative.
			if s.listening.Load() {
				return nil
			}
			// Signalled and no longer listening, or the run ended without ever
			// binding: nothing left to wait for.
			return ErrNotListening
		}

		select {
		case <-ch:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
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
// The listener is bound explicitly before serving, so when Addr asks for an
// OS-assigned port ("host:0") the resolved address is reflected back into
// Addr/Host/Port and reported by ListenAddr/ListenPort/LocalURL. As soon as
// binding succeeds, waiting WaitListening callers (and Run) are woken with the
// final address instead of polling for it.
//
// If SetListener was called, the supplied listener is served instead of binding
// Addr.
//
// Returns http.ErrServerClosed on clean shutdown, or another error on failure.
//
// Note: Start does NOT call any hooks. Use Run() for the full lifecycle.
func (s *Server) Start() error {
	s.httpMu.Lock()
	srv := s.buildHTTPServer()
	s.httpServer = srv
	ln := s.ln
	s.httpMu.Unlock()

	if ln == nil {
		// Bind explicitly so callers can read back the actual port when Addr
		// ends in ":0" (commonly used in tests).
		var err error
		ln, err = net.Listen("tcp", srv.Addr)
		if err != nil {
			s.setErr(err)
			return err
		}
	}
	return s.serve(srv, ln)
}

// ServeListener serves on an already-bound listener in the current goroutine,
// using the configured timeouts, TLS files, address reflection and lifecycle
// signals. Like Start it runs no hooks; use SetListener + Run for the full
// lifecycle.
//
// Use it when the caller owns the socket: socket activation (systemd
// LISTEN_FDS), a pre-bound port handed over by a test, or a Unix socket.
func (s *Server) ServeListener(ln net.Listener) error {
	s.httpMu.Lock()
	srv := s.buildHTTPServer()
	s.httpServer = srv
	s.httpMu.Unlock()
	return s.serve(srv, ln)
}

// SetListener makes the next Start/Run serve on an already-bound listener
// instead of binding Addr. The listener's address is reflected into
// Addr/Host/Port exactly like a self-bound one.
//
// A listener is good for one run: the underlying http.Server closes it on
// shutdown, so calling Run again on the same Server needs a fresh listener.
func (s *Server) SetListener(ln net.Listener) {
	s.httpMu.Lock()
	s.ln = ln
	s.httpMu.Unlock()
}

// Listener returns the listener the server is currently serving on, or nil when
// it is not listening.
func (s *Server) Listener() net.Listener {
	s.httpMu.Lock()
	defer s.httpMu.Unlock()
	return s.ln
}

// serve publishes the resolved address of a bound listener and serves on it
// until the server is shut down.
func (s *Server) serve(srv *http.Server, ln net.Listener) error {
	// Reflect the resolved address back into the server so String(), the
	// documented Host/Port fields and the address accessors all agree on the
	// port the OS actually assigned.
	s.httpMu.Lock()
	s.ln = ln
	srv.Addr = ln.Addr().String()
	s.Addr = srv.Addr
	s.Host, s.Port = splitHostPort(s.Addr)
	cert, key := s.TLSCertFile, s.TLSKeyFile
	s.httpMu.Unlock()

	// Bound: publish the address and wake Run/WaitListening.
	s.listening.Store(true)
	s.signalBound()

	var err error
	if cert != "" && key != "" {
		err = srv.ServeTLS(ln, cert, key)
	} else {
		err = srv.Serve(ln)
	}
	s.listening.Store(false)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.setErr(err)
	}
	return err
}

// signalBound closes the bind channel of the current run (and clears it) so
// Run and WaitListening stop waiting. Idempotent, safe from any goroutine.
func (s *Server) signalBound() {
	s.httpMu.Lock()
	ch := s.boundCh
	s.boundCh = nil
	s.httpMu.Unlock()
	if ch != nil {
		close(ch)
	}
}

// resetBoundCh installs a fresh bind channel for a new run. The previous
// channel is closed rather than dropped so a WaitListening caller that grabbed
// it before Run started wakes up, re-reads the current channel and keeps
// waiting for the new run instead of blocking forever on an abandoned one.
func (s *Server) resetBoundCh() {
	s.httpMu.Lock()
	old := s.boundCh
	s.boundCh = make(chan struct{})
	s.httpMu.Unlock()
	if old != nil {
		close(old)
	}
}

// splitHostPort splits a resolved listener address into host and port,
// tolerating addresses that carry no port (the port is 0 then).
func splitHostPort(addr string) (string, uint) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 0
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return host, 0
	}
	return host, uint(port)
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
	// No new connections are accepted from here on.
	s.listening.Store(false)

	s.httpMu.Lock()
	srv := s.httpServer
	s.ln = nil
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

// Run is the one-shot lifecycle: PreStart → Start (background) → wait for the
// listener to bind → PostStart → wait for stop signal → PreShutdown → drain →
// Shutdown → PostShutdown.
//
// The listener is bound before PostStart hooks run, so hooks (and anyone using
// WaitListening) observe the resolved address. When Addr requests an
// OS-assigned port ("host:0"), ListenAddr/ListenPort/LocalURL report the port
// the kernel picked.
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
	s.resetBoundCh()
	s.listening.Store(false)
	s.shutdown.Store(false)
	s.ready.Store(false)
	s.draining.Store(false)

	// If this run ends before binding, release any WaitListening caller.
	defer s.signalBound()

	bg := context.Background()

	// 1) PreStart hooks — first error aborts startup.
	if err := runHooks(bg, s.PreStart, true); err != nil {
		s.setErr(err)
		return err
	}

	// 2) Start in a background goroutine.
	startErrCh := make(chan error, 1)
	go func() { startErrCh <- s.Start() }()

	s.httpMu.Lock()
	boundCh := s.boundCh
	s.httpMu.Unlock()

	// 3) Wait for the listener to actually bind — either Start reports an
	// error, or Start closes boundCh once net.Listen succeeded. This is a real
	// signal (no sleep-and-hope), so PostStart hooks and the log line below
	// always see the resolved address, including an OS-assigned port.
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
	case <-boundCh:
	}

	// 4) PostStart hooks (errors logged, not fatal).
	s.runHooksLogged(bg, s.PostStart)
	s.ready.Store(true)
	if u := s.LocalURL(); u != "" && u != "http://"+s.ListenAddr() && u != "https://"+s.ListenAddr() {
		s.logf("server listening on %s (local url %s)", s.ListenAddr(), u)
	} else {
		s.logf("server listening on %s", s.ListenAddr())
	}

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
