package sse_test

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/rux/v2"
	"github.com/gookit/rux/v2/pkg/sse"
)

// startSSE wires producer/hooks behind a real httptest.NewServer so the
// underlying ResponseWriter actually supports Flusher and request
// cancellation flows end-to-end. Returns the test server URL.
func startSSE(t *testing.T, hooks *sse.Hooks, producer sse.Producer) *httptest.Server {
	t.Helper()
	r := rux.New()
	r.GET("/events", func(c *rux.Context) {
		_ = sse.Stream(c, hooks, producer)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func TestEncode_Basic(t *testing.T) {
	srv := startSSE(t, nil, func(send sse.SendFunc, done <-chan struct{}) error {
		_ = send(sse.Event{Data: "hello"})
		return nil
	})

	resp, err := http.Get(srv.URL + "/events")
	assert.NoErr(t, err)
	defer resp.Body.Close()

	assert.Eq(t, "text/event-stream", resp.Header.Get("Content-Type"))
	assert.Eq(t, "no-cache", resp.Header.Get("Cache-Control"))
	assert.Eq(t, "no", resp.Header.Get("X-Accel-Buffering"))

	body, _ := io.ReadAll(resp.Body)
	assert.Eq(t, "data: hello\n\n", string(body))
}

func TestEncode_AllFields(t *testing.T) {
	srv := startSSE(t, nil, func(send sse.SendFunc, done <-chan struct{}) error {
		_ = send(sse.Event{
			ID:    "42",
			Name:  "tick",
			Data:  "payload",
			Retry: 5000,
		})
		return nil
	})

	resp, _ := http.Get(srv.URL + "/events")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	got := string(body)
	// SSE fields ordered as: id, event, retry, data, blank line.
	want := "id: 42\nevent: tick\nretry: 5000\ndata: payload\n\n"
	assert.Eq(t, want, got)
}

func TestEncode_MultiLineData(t *testing.T) {
	srv := startSSE(t, nil, func(send sse.SendFunc, done <-chan struct{}) error {
		_ = send(sse.Event{Data: "line1\nline2\nline3"})
		return nil
	})

	resp, _ := http.Get(srv.URL + "/events")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	want := "data: line1\ndata: line2\ndata: line3\n\n"
	assert.Eq(t, want, string(body))
}

func TestEncode_EmptyDataIsHeartbeat(t *testing.T) {
	srv := startSSE(t, nil, func(send sse.SendFunc, done <-chan struct{}) error {
		_ = send(sse.Event{})
		return nil
	})

	resp, _ := http.Get(srv.URL + "/events")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Eq(t, "data: \n\n", string(body))
}

func TestHook_OnConnect_Reject(t *testing.T) {
	var producerCalled atomic.Bool
	var disconnectReason error
	var mu sync.Mutex

	hooks := &sse.Hooks{
		OnConnect: func(c *rux.Context) error {
			return errors.New("unauthorized")
		},
		OnDisconnect: func(c *rux.Context, reason error) {
			mu.Lock()
			disconnectReason = reason
			mu.Unlock()
		},
	}
	srv := startSSE(t, hooks, func(send sse.SendFunc, done <-chan struct{}) error {
		producerCalled.Store(true)
		return nil
	})

	resp, _ := http.Get(srv.URL + "/events")
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	assert.False(t, producerCalled.Load(), "producer must not run after OnConnect rejection")
	mu.Lock()
	defer mu.Unlock()
	assert.Err(t, disconnectReason)
	assert.True(t, strings.Contains(disconnectReason.Error(), "unauthorized"))
}

func TestHook_OnConnect_Reject_AllowsCustom4xx(t *testing.T) {
	// Because OnConnect runs BEFORE SSE headers are written, a rejecting
	// hook can issue its own error response with any status code.
	hooks := &sse.Hooks{
		OnConnect: func(c *rux.Context) error {
			http.Error(c.Resp, "no token", http.StatusUnauthorized)
			return errors.New("unauthorized")
		},
	}
	srv := startSSE(t, hooks, func(send sse.SendFunc, done <-chan struct{}) error {
		t.Fatal("producer must not run")
		return nil
	})

	resp, _ := http.Get(srv.URL + "/events")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	assert.Eq(t, 401, resp.StatusCode)
	assert.True(t, strings.Contains(string(body), "no token"))
	// And we did NOT switch into SSE mode.
	assert.True(t, !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream"))
}

func TestHook_OnDisconnect_CleanExit(t *testing.T) {
	var reasonCh = make(chan error, 1)
	hooks := &sse.Hooks{
		OnDisconnect: func(c *rux.Context, reason error) { reasonCh <- reason },
	}
	srv := startSSE(t, hooks, func(send sse.SendFunc, done <-chan struct{}) error {
		_ = send(sse.Event{Data: "x"})
		return nil
	})

	resp, _ := http.Get(srv.URL + "/events")
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	select {
	case err := <-reasonCh:
		assert.NoErr(t, err) // clean producer return → nil reason
	case <-time.After(2 * time.Second):
		t.Fatal("OnDisconnect not invoked")
	}
}

func TestHook_OnSend_SkipsEvent(t *testing.T) {
	var skippedErrCh = make(chan error, 1)
	hooks := &sse.Hooks{
		OnSend: func(c *rux.Context, e *sse.Event) error {
			if e.Data == "skip-me" {
				return errors.New("filtered")
			}
			return nil
		},
		OnError: func(c *rux.Context, err error) {
			select {
			case skippedErrCh <- err:
			default:
			}
		},
	}
	srv := startSSE(t, hooks, func(send sse.SendFunc, done <-chan struct{}) error {
		err := send(sse.Event{Data: "skip-me"})
		assert.True(t, errors.Is(err, sse.ErrEventSkipped))
		_ = send(sse.Event{Data: "keep-me"})
		return nil
	})

	resp, _ := http.Get(srv.URL + "/events")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	assert.True(t, strings.Contains(string(body), "data: keep-me"))
	assert.False(t, strings.Contains(string(body), "skip-me"))

	select {
	case err := <-skippedErrCh:
		assert.True(t, strings.Contains(err.Error(), "filtered"))
	case <-time.After(time.Second):
		t.Fatal("OnError not called for skipped event")
	}
}

func TestHook_OnSend_MutatesEvent(t *testing.T) {
	hooks := &sse.Hooks{
		OnSend: func(c *rux.Context, e *sse.Event) error {
			e.ID = "rewritten"
			return nil
		},
	}
	srv := startSSE(t, hooks, func(send sse.SendFunc, done <-chan struct{}) error {
		_ = send(sse.Event{Data: "hi"})
		return nil
	})

	resp, _ := http.Get(srv.URL + "/events")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.True(t, strings.Contains(string(body), "id: rewritten"))
}

func TestProducer_ClientDisconnect_StopsProducer(t *testing.T) {
	producerReturned := make(chan struct{})

	srv := startSSE(t, nil, func(send sse.SendFunc, done <-chan struct{}) error {
		defer close(producerReturned)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return nil
			case <-ticker.C:
				if err := send(sse.Event{Data: "tick"}); err != nil {
					return err
				}
			}
		}
	})

	// Client cancels before reading much.
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/events", nil)
	resp, err := http.DefaultClient.Do(req)
	assert.NoErr(t, err)

	// Read one frame, then cancel.
	br := bufio.NewReader(resp.Body)
	_, _ = br.ReadString('\n')
	cancel()
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	select {
	case <-producerReturned:
		// good — producer noticed done and returned
	case <-time.After(3 * time.Second):
		t.Fatal("producer did not exit after client disconnect")
	}
}

func TestStream_NilHooks_OK(t *testing.T) {
	// Passing nil hooks must not panic — it's the documented shortcut
	// for "no callbacks needed".
	srv := startSSE(t, nil, func(send sse.SendFunc, done <-chan struct{}) error {
		return send(sse.Event{Data: "ok"})
	})
	resp, _ := http.Get(srv.URL + "/events")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.True(t, strings.Contains(string(body), "data: ok"))
}

func TestStream_MultipleEvents(t *testing.T) {
	srv := startSSE(t, nil, func(send sse.SendFunc, done <-chan struct{}) error {
		for i := 0; i < 3; i++ {
			if err := send(sse.Event{Data: "msg"}); err != nil {
				return err
			}
		}
		return nil
	})

	resp, _ := http.Get(srv.URL + "/events")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Each event ends with a blank line — 3 events → 3 occurrences.
	count := strings.Count(string(body), "data: msg\n\n")
	assert.Eq(t, 3, count)
}

func TestSendFunc_AfterClientGone_ReportsError(t *testing.T) {
	// Once the underlying connection is closed, send() returns a write
	// error and the producer is expected to bail out.
	var lastErr atomic.Value // error
	var producerExited = make(chan struct{})

	srv := startSSE(t, nil, func(send sse.SendFunc, done <-chan struct{}) error {
		defer close(producerExited)
		// Spin until the client goes away or send fails.
		for {
			select {
			case <-done:
				return nil
			default:
			}
			if err := send(sse.Event{Data: strings.Repeat("x", 1024)}); err != nil {
				lastErr.Store(err)
				return err
			}
			time.Sleep(time.Millisecond)
		}
	})

	resp, _ := http.Get(srv.URL + "/events")
	_ = resp.Body.Close() // hang up immediately

	select {
	case <-producerExited:
	case <-time.After(3 * time.Second):
		t.Fatal("producer did not notice closed client")
	}
	// Either a write error or done-channel exit is acceptable; both
	// signal a clean shutdown. We only assert the producer left the loop.
}
