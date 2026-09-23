package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/rux/v2"
)

// Addr that cannot be bound, so a test only passes when the provided listener
// is actually used.
const badAddr = "not-a-real-host.invalid:1"

func TestServeListener_ServesProvidedListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)

	s := addrTestServer(badAddr)
	s.GET("/ping", func(c *rux.Context) { c.Text(200, "pong") })

	done := make(chan error, 1)
	go func() { done <- s.ServeListener(ln) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	assert.NoErr(t, s.WaitListening(ctx))

	assert.Eq(t, ln.Addr().String(), s.ListenAddr())
	assert.Eq(t, ln, s.Listener())
	assert.True(t, s.IsListening())

	resp, err := http.Get(s.LocalURL() + "/ping")
	assert.NoErr(t, err)
	defer resp.Body.Close()
	assert.Eq(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoErr(t, err)
	assert.Eq(t, "pong", string(body))

	shCtx, shCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shCancel()
	assert.NoErr(t, s.Shutdown(shCtx))

	// Start/ServeListener report http.ErrServerClosed on clean shutdown.
	assert.ErrIs(t, <-done, http.ErrServerClosed)
	assert.Nil(t, s.Listener())
	assert.False(t, s.IsListening())
}

func TestSetListener_RunServesProvidedListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)

	s := addrTestServer(badAddr)
	s.SetListener(ln)
	s.GET("/ping", func(c *rux.Context) { c.Text(200, "pong") })

	done := startInBackground(t, s)
	defer stopAndWait(t, s, done)

	assert.Eq(t, ln.Addr().String(), s.ListenAddr())
	assert.Eq(t, ln, s.Listener())

	resp, err := http.Get(s.LocalURL() + "/ping")
	assert.NoErr(t, err)
	defer resp.Body.Close()
	assert.Eq(t, 200, resp.StatusCode)
}

func TestSetListener_AddressIsReflected(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)

	s := addrTestServer(badAddr)
	s.SetListener(ln)

	done := startInBackground(t, s)
	defer stopAndWait(t, s, done)

	port := ln.Addr().(*net.TCPAddr).Port
	assert.Eq(t, port, s.ListenPort())
	assert.Eq(t, uint(port), s.Port)
	assert.Eq(t, "127.0.0.1", s.Host)
	assert.Eq(t, ln.Addr().String(), s.String())
}
