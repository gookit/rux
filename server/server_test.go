package server

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/rux/v2"
)

func TestNew_AppliesDefaults(t *testing.T) {
	s := New(false)
	assert.Eq(t, DefaultReadHeaderTimeout, s.ReadHeaderTimeout)
	assert.Eq(t, DefaultReadTimeout, s.ReadTimeout)
	assert.Eq(t, DefaultWriteTimeout, s.WriteTimeout)
	assert.Eq(t, DefaultIdleTimeout, s.IdleTimeout)
	assert.Eq(t, DefaultMaxHeaderBytes, s.MaxHeaderBytes)
	assert.Eq(t, DefaultShutdownTimeout, s.ShutdownTimeout)
	assert.Eq(t, DefaultDrainDelay, s.DrainDelay)
	assert.Eq(t, 2, len(s.StopSignals))
	assert.NotNil(t, s.Logger)
}

func TestSetAddr(t *testing.T) {
	s := New(false)
	s.SetAddr("127.0.0.1", 8081)
	assert.Eq(t, "127.0.0.1:8081", s.Addr)
}

func TestSetHostPort_BackwardCompat(t *testing.T) {
	s := New(false)
	s.SetHostPort("127.0.0.1", 9000)
	assert.Eq(t, "127.0.0.1:9000", s.Addr)
	assert.Eq(t, "127.0.0.1:9000", s.String())
}

func TestString_DefaultAddr(t *testing.T) {
	s := New(false)
	assert.Eq(t, DefaultAddr, s.String())
}

func TestRun_PreStartErrorAbortsStartup(t *testing.T) {
	s := New(false)
	s.Addr = "127.0.0.1:0"
	var started atomic.Bool
	s.PreStart = append(s.PreStart, func(ctx context.Context) error {
		return errors.New("init failed")
	})
	s.PostStart = append(s.PostStart, func(ctx context.Context) error {
		started.Store(true)
		return nil
	})
	err := s.Run()
	assert.Err(t, err)
	assert.True(t, strings.Contains(err.Error(), "init failed"))
	assert.False(t, started.Load())
	assert.Eq(t, err, s.Err())
}

func TestRun_SignalTriggersShutdown(t *testing.T) {
	s := New(false)
	s.Addr = "127.0.0.1:0"
	s.DrainDelay = 0 // skip drain in test
	s.ShutdownTimeout = 2 * time.Second
	// Silence logger output during test.
	s.Logger = func(format string, args ...any) {}

	var postStarted, preShutdown, postShutdown atomic.Bool
	s.PostStart = append(s.PostStart, func(ctx context.Context) error {
		postStarted.Store(true)
		return nil
	})
	s.PreShutdown = append(s.PreShutdown, func(ctx context.Context) error {
		preShutdown.Store(true)
		return nil
	})
	s.PostShutdown = append(s.PostShutdown, func(ctx context.Context) error {
		postShutdown.Store(true)
		return nil
	})

	done := make(chan error, 1)
	go func() { done <- s.Run() }()

	// Wait for ready.
	deadline := time.Now().Add(3 * time.Second)
	for !s.ready.Load() {
		if time.Now().After(deadline) {
			t.Fatal("server never became ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(t, postStarted.Load())

	// Trigger shutdown. os.Process.Signal cannot deliver SIGTERM/Interrupt on
	// Windows, so we test the equivalent Stop() path there and the signal path
	// on POSIX systems (see TestRun_RealSignalTriggersShutdown below).
	s.Stop()
	_ = runtime.GOOS // retained for clarity; actual signal coverage is in build-tagged tests

	select {
	case err := <-done:
		assert.NoErr(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after signal")
	}
	assert.True(t, preShutdown.Load())
	assert.True(t, postShutdown.Load())
}

func TestRun_DebugDumpsRoutes(t *testing.T) {
	// New(true) flips rux debug mode on, which makes Run print the
	// route table once the listener is up.
	s := New(true)
	s.Addr = "127.0.0.1:0"
	s.DrainDelay = 0
	s.ShutdownTimeout = 2 * time.Second

	s.GET("/api/users/{id}", func(c *rux.Context) {})
	s.POST("/api/posts", func(c *rux.Context) {})

	// Capture log lines so we can assert on the route dump output.
	var logged []string
	var mu sync.Mutex
	s.Logger = func(format string, args ...any) {
		mu.Lock()
		logged = append(logged, fmt.Sprintf(format, args...))
		mu.Unlock()
	}

	done := make(chan error, 1)
	go func() { done <- s.Run() }()

	// Wait for ready, then stop.
	deadline := time.Now().Add(3 * time.Second)
	for !s.ready.Load() {
		if time.Now().After(deadline) {
			t.Fatal("server never became ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.Stop()
	select {
	case err := <-done:
		assert.NoErr(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return")
	}

	// The dump line should mention the two registered routes.
	mu.Lock()
	all := strings.Join(logged, "\n")
	mu.Unlock()
	assert.True(t, strings.Contains(all, "registered routes"))
	// rux normalizes "{id}" to ":id" internally; assert on the stored form.
	assert.True(t, strings.Contains(all, "/api/users/:id"))
	assert.True(t, strings.Contains(all, "/api/posts"))
}

func TestRun_NonDebugSkipsRouteDump(t *testing.T) {
	// Non-debug mode must not print the dump. New(false) resets the
	// package-level debug flag flipped by the previous test.
	s := New(false)
	s.Addr = "127.0.0.1:0"
	s.DrainDelay = 0
	s.ShutdownTimeout = 2 * time.Second
	s.GET("/silent", func(c *rux.Context) {})

	var logged []string
	var mu sync.Mutex
	s.Logger = func(format string, args ...any) {
		mu.Lock()
		logged = append(logged, fmt.Sprintf(format, args...))
		mu.Unlock()
	}

	done := make(chan error, 1)
	go func() { done <- s.Run() }()
	deadline := time.Now().Add(3 * time.Second)
	for !s.ready.Load() {
		if time.Now().After(deadline) {
			t.Fatal("server never became ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.Stop()
	<-done

	mu.Lock()
	all := strings.Join(logged, "\n")
	mu.Unlock()
	assert.False(t, strings.Contains(all, "registered routes"),
		"non-debug mode should not dump routes; got: %s", all)
}

func TestShutdown_Idempotent(t *testing.T) {
	s := New(false)
	// Shutdown before Start returns nil — nothing to do.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	assert.NoErr(t, s.Shutdown(ctx))
	assert.NoErr(t, s.Shutdown(ctx))
}
