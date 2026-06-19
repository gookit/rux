//go:build !windows

package server

import (
	"context"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
)

// TestRun_RealSignalTriggersShutdown exercises the SIGTERM path. Windows
// cannot deliver SIGTERM/Interrupt via os.Process.Signal, so this test is
// gated to POSIX systems; the Stop()-based equivalent is exercised on all
// platforms by TestRun_SignalTriggersShutdown.
func TestRun_RealSignalTriggersShutdown(t *testing.T) {
	s := New(false)
	s.Addr = "127.0.0.1:0"
	s.DrainDelay = 0
	s.ShutdownTimeout = 2 * time.Second
	s.Logger = func(format string, args ...any) {}

	var postStarted, preShutdown atomic.Bool
	s.PostStart = append(s.PostStart, func(ctx context.Context) error {
		postStarted.Store(true)
		return nil
	})
	s.PreShutdown = append(s.PreShutdown, func(ctx context.Context) error {
		preShutdown.Store(true)
		return nil
	})

	done := make(chan error, 1)
	go func() { done <- s.Run() }()

	deadline := time.Now().Add(3 * time.Second)
	for !s.ready.Load() {
		if time.Now().After(deadline) {
			t.Fatal("server never became ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(t, postStarted.Load())

	p, err := os.FindProcess(os.Getpid())
	assert.NoErr(t, err)
	assert.NoErr(t, p.Signal(syscall.SIGTERM))

	select {
	case err := <-done:
		assert.NoErr(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after signal")
	}
	assert.True(t, preShutdown.Load())
}
