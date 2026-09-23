package core

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what fn
// printed. The Listen family reports the resolved address through fmt.Printf.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	rp, wp, err := os.Pipe()
	assert.NoErr(t, err)
	os.Stdout = wp
	defer func() { os.Stdout = old }()

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(rp)
		done <- string(b)
	}()

	fn()

	assert.NoErr(t, wp.Close())
	return <-done
}

func waitListening(t *testing.T, r *Router) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for r.Listener() == nil {
		if time.Now().After(deadline) {
			t.Fatal("router never bound a listener")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestListenAddr_EmptyBeforeBind(t *testing.T) {
	r := New()

	assert.Eq(t, "", r.ListenAddr())
	assert.Eq(t, 0, r.ListenPort())
	assert.Nil(t, r.Listener())
	assert.NoErr(t, r.Err())
}

// Bind is the port-0 story for the core router: the real port is available
// before any traffic arrives.
func TestBind_ResolvesEphemeralPort(t *testing.T) {
	r := New()
	r.GET("/ping", func(c *Context) { c.Text(200, "pong") })

	ln, err := r.Bind("127.0.0.1:0")
	assert.NoErr(t, err)

	port := ln.Addr().(*net.TCPAddr).Port
	assert.True(t, port > 0)
	assert.Eq(t, ln.Addr().String(), r.ListenAddr())
	assert.Eq(t, port, r.ListenPort())
	assert.Eq(t, ln, r.Listener())

	done := make(chan struct{})
	go func() {
		r.ServeListener(ln)
		close(done)
	}()

	resp, err := http.Get("http://" + r.ListenAddr() + "/ping")
	assert.NoErr(t, err)
	defer resp.Body.Close()
	assert.Eq(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoErr(t, err)
	assert.Eq(t, "pong", string(body))

	// Closing the listener ends the serve loop.
	assert.NoErr(t, ln.Close())
	<-done
	assert.Nil(t, r.Listener())
	assert.Err(t, r.Err())
}

func TestBind_WildcardPort(t *testing.T) {
	r := New()
	ln, err := r.Bind(":0")
	assert.NoErr(t, err)
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	assert.True(t, port > 0)
	assert.Eq(t, port, r.ListenPort())
	assert.Eq(t, ln.Addr().String(), r.ListenAddr())
}

// Listen prints the resolved address, not the requested ":0".
func TestListen_PrintsResolvedAddr(t *testing.T) {
	r := New()
	r.GET("/ping", func(c *Context) { c.Text(200, "pong") })

	var port int
	out := captureStdout(t, func() {
		done := make(chan struct{})
		go func() {
			r.Listen("127.0.0.1:0")
			close(done)
		}()

		waitListening(t, r)
		port = r.ListenPort()
		assert.NoErr(t, r.Listener().Close())
		<-done
	})

	assert.True(t, port > 0)
	assert.StrContains(t, out, fmt.Sprintf("Serve listen on 127.0.0.1:%d", port))
}

func TestServeListener_ReflectsProvidedListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)

	r := New()
	r.GET("/x", func(c *Context) { c.Text(200, "ok") })

	done := make(chan struct{})
	go func() {
		r.ServeListener(ln)
		close(done)
	}()

	addr := ln.Addr().String()
	deadline := time.Now().Add(3 * time.Second)
	var resp *http.Response
	for {
		resp, err = http.Get("http://" + addr + "/x")
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("listener never served: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	assert.Eq(t, 200, resp.StatusCode)
	assert.NoErr(t, resp.Body.Close())

	assert.Eq(t, addr, r.ListenAddr())
	assert.Eq(t, ln.Addr().(*net.TCPAddr).Port, r.ListenPort())

	assert.NoErr(t, ln.Close())
	<-done
	assert.Nil(t, r.Listener())
}

func TestBind_BindErrorLeavesNoListener(t *testing.T) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)
	defer held.Close()

	r := New()
	ln, err := r.Bind(held.Addr().String())
	assert.Err(t, err)
	assert.Nil(t, ln)
	assert.Eq(t, "", r.ListenAddr())
	assert.Eq(t, 0, r.ListenPort())
	assert.Nil(t, r.Listener())
	assert.ErrIs(t, r.Err(), err)
}

// Listen must keep recording bind errors the way it always did.
func TestListen_BindErrorKeepsErr(t *testing.T) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)
	defer held.Close()

	r := New()
	r.GET("/x", func(c *Context) { c.Text(200, "x") })
	r.Listen(held.Addr().String())
	assert.Err(t, r.Err())
	assert.Eq(t, "", r.ListenAddr())
}
