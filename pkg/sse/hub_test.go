package sse_test

import (
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

// hubReady starts a real httptest.NewServer and waits for n clients to
// register under the given hub before returning. Without this barrier,
// Send races the GET handler's registration and the test gets flaky.
func hubReady(t *testing.T, hub *sse.Hub, want int, deadline time.Duration) {
	t.Helper()
	start := time.Now()
	for {
		got, _ := hub.Count()
		if got >= want {
			return
		}
		if time.Since(start) > deadline {
			t.Fatalf("timed out waiting for %d clients (have %d)", want, got)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestHub_RegisterUnregister(t *testing.T) {
	h := sse.NewHub(8)
	assert.False(t, h.Has("a"))

	c1 := h.Register("a")
	c2 := h.Register("a")
	c3 := h.Register("b")

	clients, ids := h.Count()
	assert.Eq(t, 3, clients)
	assert.Eq(t, 2, ids)
	assert.True(t, h.Has("a"))
	assert.True(t, h.Has("b"))

	h.Unregister(c1)
	h.Unregister(c1) // double-unregister must be safe
	clients, ids = h.Count()
	assert.Eq(t, 2, clients)
	assert.Eq(t, 2, ids) // "a" still has c2

	h.Unregister(c2)
	clients, ids = h.Count()
	assert.Eq(t, 1, clients)
	assert.Eq(t, 1, ids)
	assert.False(t, h.Has("a"))

	h.Unregister(c3)
	clients, ids = h.Count()
	assert.Eq(t, 0, clients)
	assert.Eq(t, 0, ids)
}

func TestHub_Send_FansOutToAllClientsOfID(t *testing.T) {
	hub := sse.NewHub(16)
	srv := startHubServer(t, hub)

	// Open two SSE streams under the same ID (multi-tab user).
	resp1, br1 := openSSE(t, srv.URL+"/events?uid=u1")
	resp2, br2 := openSSE(t, srv.URL+"/events?uid=u1")
	defer resp1.Body.Close()
	defer resp2.Body.Close()

	hubReady(t, hub, 2, 2*time.Second)

	delivered, dropped := hub.Send("u1", sse.Event{Data: "hello"})
	assert.Eq(t, 2, delivered)
	assert.Eq(t, 0, dropped)

	// Both clients should receive the same payload.
	assert.True(t, strings.Contains(readUntilFrame(t, br1), "data: hello"))
	assert.True(t, strings.Contains(readUntilFrame(t, br2), "data: hello"))
}

func TestHub_Send_UnknownID_NoOp(t *testing.T) {
	hub := sse.NewHub(8)
	delivered, dropped := hub.Send("nope", sse.Event{Data: "x"})
	assert.Eq(t, 0, delivered)
	assert.Eq(t, 0, dropped)
}

func TestHub_Broadcast_HitsEveryClient(t *testing.T) {
	hub := sse.NewHub(16)
	srv := startHubServer(t, hub)

	r1, br1 := openSSE(t, srv.URL+"/events?uid=u1")
	r2, br2 := openSSE(t, srv.URL+"/events?uid=u2")
	r3, br3 := openSSE(t, srv.URL+"/events?uid=u2")
	defer r1.Body.Close()
	defer r2.Body.Close()
	defer r3.Body.Close()

	hubReady(t, hub, 3, 2*time.Second)

	delivered, _ := hub.Broadcast(sse.Event{Data: "bcast"})
	assert.Eq(t, 3, delivered)
	assert.True(t, strings.Contains(readUntilFrame(t, br1), "data: bcast"))
	assert.True(t, strings.Contains(readUntilFrame(t, br2), "data: bcast"))
	assert.True(t, strings.Contains(readUntilFrame(t, br3), "data: bcast"))
}

func TestHub_DropsWhenBufferFull(t *testing.T) {
	// Tiny buffer + a producer that intentionally doesn't drain so the
	// chan fills up immediately.
	hub := sse.NewHub(2)

	dropEvents := make(chan sse.Event, 16)
	hub.SetOnDrop(func(c *sse.Client, e sse.Event) {
		select {
		case dropEvents <- e:
		default:
		}
	})

	// Register a client directly without a producer — its chan won't drain.
	client := hub.Register("slow")
	defer hub.Unregister(client)

	// Fill the buffer: bufSize=2 → first 2 deliver, rest drop.
	for i := 0; i < 5; i++ {
		hub.Send("slow", sse.Event{Data: "x"})
	}

	assert.Eq(t, int64(3), client.Dropped())

	// OnDrop fired 3 times.
	assert.Eq(t, 3, len(dropEvents))
}

func TestHub_PushAfterUnregister_ReturnsFalse(t *testing.T) {
	hub := sse.NewHub(4)
	c := hub.Register("x")
	hub.Unregister(c)

	delivered, dropped := hub.Send("x", sse.Event{Data: "after"})
	// "x" was removed from the map, so Send finds no clients.
	assert.Eq(t, 0, delivered)
	assert.Eq(t, 0, dropped)
}

func TestHub_HubProducer_EndToEnd(t *testing.T) {
	hub := sse.NewHub(8)
	srv := startHubServer(t, hub)

	resp, br := openSSE(t, srv.URL+"/events?uid=alice")
	defer resp.Body.Close()
	hubReady(t, hub, 1, 2*time.Second)

	hub.Send("alice", sse.Event{Name: "ping", Data: "1"})
	frame := readUntilFrame(t, br)
	assert.True(t, strings.Contains(frame, "event: ping"))
	assert.True(t, strings.Contains(frame, "data: 1"))
}

func TestHub_OnDrop_HookCanKickSlowClient(t *testing.T) {
	hub := sse.NewHub(1)
	var kicked atomic.Int32
	hub.SetOnDrop(func(c *sse.Client, _ sse.Event) {
		if kicked.Add(1) == 1 {
			hub.Unregister(c)
		}
	})

	c := hub.Register("slow")
	hub.Send("slow", sse.Event{Data: "1"}) // delivered (buffer was empty)
	hub.Send("slow", sse.Event{Data: "2"}) // dropped → onDrop fires, kicks client

	assert.Eq(t, int32(1), kicked.Load())

	// After kick, more sends route to nobody.
	delivered, _ := hub.Send("slow", sse.Event{Data: "3"})
	assert.Eq(t, 0, delivered)

	_ = c // satisfy unused
}

func TestHub_ConcurrentRegisterSendUnregister(t *testing.T) {
	hub := sse.NewHub(8)
	const N = 50
	var wg sync.WaitGroup

	// Half: register/send/unregister loop.
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := hub.Register("u")
			for j := 0; j < 5; j++ {
				hub.Send("u", sse.Event{Data: "x"})
			}
			hub.Unregister(c)
			_ = i
		}(i)
	}
	// Half: broadcast loop.
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				hub.Broadcast(sse.Event{Data: "b"})
			}
		}()
	}

	wg.Wait()

	// All goroutines unregister at the end → hub must be empty.
	clients, ids := hub.Count()
	assert.Eq(t, 0, clients)
	assert.Eq(t, 0, ids)
}

// --- helpers --------------------------------------------------------

func startHubServer(t *testing.T, hub *sse.Hub) *httptest.Server {
	t.Helper()
	r := rux.New()
	r.GET("/events", func(c *rux.Context) {
		uid := c.Query("uid")
		_ = sse.StreamWith(c, &sse.Options{SendConnected: false}, sse.HubProducer(hub, uid))
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func openSSE(t *testing.T, url string) (*http.Response, *bufReader) {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	assert.NoErr(t, err)
	return resp, newBufReader(resp.Body)
}

// readUntilFrame reads until a blank line terminates an SSE frame and
// returns the frame's payload. Times out at 2s to keep tests snappy.
func readUntilFrame(t *testing.T, br *bufReader) string {
	t.Helper()
	deadline := time.After(2 * time.Second)
	frame := ""
	for {
		select {
		case <-deadline:
			t.Fatalf("frame timeout — got so far: %q", frame)
			return frame
		default:
		}
		line, err := br.readLine(200 * time.Millisecond)
		if err == nil {
			frame += line + "\n"
			if line == "" {
				return frame
			}
		}
	}
}

// bufReader wraps an io.ReadCloser with a goroutine that funnels lines
// onto a channel so readLine can deadline-poll without blocking forever.
type bufReader struct {
	lines chan string
	stop  chan struct{}
}

func newBufReader(r io.ReadCloser) *bufReader {
	br := &bufReader{
		lines: make(chan string, 64),
		stop:  make(chan struct{}),
	}
	go func() {
		buf := make([]byte, 4096)
		var carry []byte
		for {
			n, err := r.Read(buf)
			if n > 0 {
				carry = append(carry, buf[:n]...)
				for {
					i := indexByte(carry, '\n')
					if i < 0 {
						break
					}
					select {
					case br.lines <- string(carry[:i]):
					case <-br.stop:
						return
					}
					carry = carry[i+1:]
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return br
}

func (b *bufReader) readLine(timeout time.Duration) (string, error) {
	select {
	case s := <-b.lines:
		return s, nil
	case <-time.After(timeout):
		return "", io.EOF
	}
}

func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}
