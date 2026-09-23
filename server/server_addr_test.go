package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/rux/v2"
)

// addrTestServer builds a Server with test-friendly lifecycle timings.
func addrTestServer(addr string) *Server {
	s := New(false)
	s.Addr = addr
	s.DrainDelay = 0
	s.ShutdownTimeout = 2 * time.Second
	s.Logger = func(string, ...any) {}
	return s
}

// startInBackground runs the server and waits until the listener is bound.
func startInBackground(t *testing.T, s *Server) chan error {
	t.Helper()

	done := make(chan error, 1)
	go func() { done <- s.Run() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.WaitListening(ctx); err != nil {
		t.Fatalf("server never started listening: %v", err)
	}
	return done
}

// waitPostStart waits until the PostStart hooks have finished.
func waitPostStart(t *testing.T, s *Server) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for !s.ready.Load() {
		if time.Now().After(deadline) {
			t.Fatal("server never finished PostStart")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// stopAndWait stops the server and asserts Run returned cleanly.
func stopAndWait(t *testing.T, s *Server, done chan error) {
	t.Helper()

	s.Stop()
	select {
	case err := <-done:
		assert.NoErr(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after Stop")
	}
}

// TestRun_EphemeralPort_ReportsResolvedAddress is the regression test for the
// "--port 0 plus --open" story: when Addr asks for an OS-assigned port, the
// resolved port must be observable (and browser-usable) instead of leaving the
// caller with ":0".
func TestRun_EphemeralPort_ReportsResolvedAddress(t *testing.T) {
	s := addrTestServer("127.0.0.1:0")
	s.GET("/ping", func(c *rux.Context) { c.Text(200, "pong") })

	done := startInBackground(t, s)
	defer stopAndWait(t, s, done)

	port := s.ListenPort()
	assert.Gt(t, port, 0)

	assert.Eq(t, fmt.Sprintf("127.0.0.1:%d", port), s.ListenAddr())
	assert.Eq(t, fmt.Sprintf("http://127.0.0.1:%d", port), s.LocalURL())
	assert.True(t, s.IsListening())

	// The documented Host/Port fields must not keep reporting port 0.
	assert.Eq(t, "127.0.0.1", s.Host)
	assert.Eq(t, uint(port), s.Port)

	// The reported URL must really serve.
	resp, err := http.Get(s.LocalURL() + "/ping")
	assert.NoErr(t, err)
	defer resp.Body.Close()
	assert.Eq(t, 200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoErr(t, err)
	assert.Eq(t, "pong", string(body))
}

// TestRun_WildcardEphemeralPort_LocalURLIsLoopback covers the ":0" (all
// interfaces) form: the listener address stays a wildcard, but LocalURL must
// hand back something a browser can open.
func TestRun_WildcardEphemeralPort_LocalURLIsLoopback(t *testing.T) {
	s := addrTestServer(":0")
	s.GET("/ping", func(c *rux.Context) { c.Text(200, "pong") })

	done := startInBackground(t, s)
	defer stopAndWait(t, s, done)

	port := s.ListenPort()
	assert.Gt(t, port, 0)
	assert.StrContains(t, s.ListenAddr(), fmt.Sprintf(":%d", port))
	assert.Eq(t, fmt.Sprintf("http://127.0.0.1:%d", port), s.LocalURL())
	assert.StrNotContains(t, s.LocalURL(), "[::]")
	assert.StrNotContains(t, s.LocalURL(), "0.0.0.0")

	resp, err := http.Get(s.LocalURL() + "/ping")
	assert.NoErr(t, err)
	defer resp.Body.Close()
	assert.Eq(t, 200, resp.StatusCode)
}

// TestRun_PostStartHookSeesResolvedAddr locks in the contract that the
// lifecycle no longer guesses how long binding takes.
func TestRun_PostStartHookSeesResolvedAddr(t *testing.T) {
	s := addrTestServer("127.0.0.1:0")

	seen := make(chan [2]string, 1)
	s.PostStart = append(s.PostStart, func(ctx context.Context) error {
		seen <- [2]string{s.ListenAddr(), s.LocalURL()}
		return nil
	})

	done := startInBackground(t, s)
	defer stopAndWait(t, s, done)
	waitPostStart(t, s)

	select {
	case got := <-seen:
		assert.StrNotContains(t, got[0], ":0")
		assert.Eq(t, s.ListenAddr(), got[0])
		assert.Eq(t, s.LocalURL(), got[1])
		assert.StrContains(t, got[1], fmt.Sprintf(":%d", s.ListenPort()))
	case <-time.After(time.Second):
		t.Fatal("PostStart hook did not run")
	}
}

// TestWaitListening_ReportsBindError makes sure a waiter is not left hanging
// when the requested port is taken.
func TestWaitListening_ReportsBindError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)
	defer ln.Close()

	s := addrTestServer(ln.Addr().String())
	done := make(chan error, 1)
	go func() { done <- s.Run() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	werr := s.WaitListening(ctx)
	assert.Err(t, werr)
	assert.False(t, s.IsListening())

	select {
	case rerr := <-done:
		assert.ErrIs(t, rerr, werr)
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after a bind failure")
	}
}

func TestWaitListening_ReportsPreStartError(t *testing.T) {
	s := addrTestServer("127.0.0.1:0")
	s.PreStart = append(s.PreStart, func(ctx context.Context) error {
		return errors.New("boom: invalid config")
	})

	done := make(chan error, 1)
	go func() { done <- s.Run() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.WaitListening(ctx)
	assert.Err(t, err)
	assert.StrContains(t, err.Error(), "boom: invalid config")
	assert.False(t, s.IsListening())
	assert.Err(t, <-done)
}

func TestWaitListening_NotListening(t *testing.T) {
	s := addrTestServer("127.0.0.1:0")

	// No run in flight: a waiter waits for the next run and gives up with the
	// context error.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	assert.ErrIs(t, s.WaitListening(ctx), context.DeadlineExceeded)

	// After a run has come and gone there is nothing left to wait for.
	done := startInBackground(t, s)
	stopAndWait(t, s, done)
	assert.False(t, s.IsListening())

	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	assert.ErrIs(t, s.WaitListening(ctx2), ErrNotListening)
}

// TestRun_BindsWithoutFixedDelay guards the removal of the old fixed
// "give Start a moment to bind" sleep: readiness must follow the real bind,
// not an arbitrary timer.
func TestRun_BindsWithoutFixedDelay(t *testing.T) {
	s := addrTestServer("127.0.0.1:0")

	start := time.Now()
	done := startInBackground(t, s)
	elapsed := time.Since(start)
	defer stopAndWait(t, s, done)

	t.Logf("listener reported ready after %v", elapsed)
	assert.Lt(t, elapsed, 50*time.Millisecond)
}

func TestLocalURL_Formats(t *testing.T) {
	cases := []struct {
		addr, cert, key, want string
	}{
		{"", "", "", ""},
		{"127.0.0.1:8080", "", "", "http://127.0.0.1:8080"},
		{":8080", "", "", "http://127.0.0.1:8080"},
		{"0.0.0.0:8080", "", "", "http://127.0.0.1:8080"},
		{"[::]:8080", "", "", "http://127.0.0.1:8080"},
		{"[::1]:8080", "", "", "http://[::1]:8080"},
		{"example.test:8080", "", "", "http://example.test:8080"},
		// port 0 means "the OS has not assigned one yet": no usable URL.
		{"127.0.0.1:0", "", "", ""},
		{":0", "", "", ""},
		{"127.0.0.1:8443", "cert.pem", "key.pem", "https://127.0.0.1:8443"},
	}

	for _, c := range cases {
		s := &Server{Addr: c.addr, TLSCertFile: c.cert, TLSKeyFile: c.key}
		assert.Eq(t, c.want, s.LocalURL(), "addr=%q", c.addr)
	}
}

func TestListenPort_ResolvedOrLegacy(t *testing.T) {
	assert.Eq(t, 8080, (&Server{Addr: "127.0.0.1:8080"}).ListenPort())
	assert.Eq(t, 0, (&Server{Addr: ":0"}).ListenPort())
	assert.Eq(t, 0, (&Server{Addr: "127.0.0.1"}).ListenPort())
	// Legacy Host/Port fields are the fallback when Addr carries no port.
	assert.Eq(t, 9000, (&Server{Port: 9000}).ListenPort())
}
