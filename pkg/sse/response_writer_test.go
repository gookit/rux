package sse_test

// Tests for how sse.Stream interacts with the response writer: flushing through
// wrappers (rux's own writer, middlewares that hide Flusher) and clearing the
// server-level write deadline so a stream is not cut off mid-flight.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/rux/v2"
	"github.com/gookit/rux/v2/pkg/sse"
)

// noFlushWriter implements only the three http.ResponseWriter methods, the way
// a middleware that forgot to preserve Flusher would.
type noFlushWriter struct {
	h      http.Header
	status int
	body   strings.Builder
}

func (w *noFlushWriter) Header() http.Header         { return w.h }
func (w *noFlushWriter) WriteHeader(status int)      { w.status = status }
func (w *noFlushWriter) Write(b []byte) (int, error) { return w.body.Write(b) }

// unwrapWriter hides Flusher but exposes Unwrap, the way
// http.ResponseController-aware middlewares are supposed to wrap.
type unwrapWriter struct {
	http.ResponseWriter
}

func (w *unwrapWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// deadlineRecorder records SetWriteDeadline calls, i.e. proves that the SSE
// handler clears the server-level write timeout for its response.
type deadlineRecorder struct {
	*httptest.ResponseRecorder
	mu       sync.Mutex
	deadline time.Time
	calls    int
}

func (w *deadlineRecorder) SetWriteDeadline(t time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.deadline, w.calls = t, w.calls+1
	return nil
}

func newSSERouter(producer sse.Producer) *rux.Router {
	r := rux.New()
	r.GET("/events", func(c *rux.Context) {
		_ = sse.Stream(c, nil, producer)
	})
	return r
}

func noopProducer(send sse.SendFunc, done <-chan struct{}) error { return nil }

// A writer that cannot flush must surface ErrFlushNotSupported. rux's writer
// implements Flusher itself, so a naive assertion on c.Resp would pass and the
// failure would show up as a panic inside Flush instead.
func TestStream_NonFlushableWriter_ReturnsErrFlushNotSupported(t *testing.T) {
	var got error
	r := rux.New()
	r.GET("/events", func(c *rux.Context) {
		got = sse.Stream(c, nil, noopProducer)
	})

	w := &noFlushWriter{h: http.Header{}}
	r.ServeHTTP(w, httptest.NewRequest("GET", "/events", nil))

	assert.Eq(t, sse.ErrFlushNotSupported, got)
	// Nothing SSE-ish was written: the handler can still send a real error.
	assert.Eq(t, "", w.body.String())
}

// Flushing must work through a wrapper that hides Flusher but exposes Unwrap.
func TestStream_FlushesThroughUnwrapWrapper(t *testing.T) {
	r := newSSERouter(func(send sse.SendFunc, done <-chan struct{}) error {
		return send(sse.Event{Data: "hello"})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(&unwrapWriter{ResponseWriter: rec}, httptest.NewRequest("GET", "/events", nil))

	assert.Eq(t, 200, rec.Code)
	assert.StrContains(t, rec.Body.String(), ": connected")
	assert.StrContains(t, rec.Body.String(), "data: hello")
	assert.True(t, rec.Flushed)
}

// StreamWith must clear the write deadline for its response, otherwise
// server.Server's default 30s WriteTimeout cuts long-lived streams.
func TestStream_ClearsWriteDeadline(t *testing.T) {
	w := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	newSSERouter(noopProducer).ServeHTTP(w, httptest.NewRequest("GET", "/events", nil))

	w.mu.Lock()
	calls, deadline := w.calls, w.deadline
	w.mu.Unlock()

	assert.Eq(t, 1, calls)
	assert.True(t, deadline.IsZero(), "write deadline should be cleared, got %v", deadline)
	assert.Eq(t, 200, w.Code)
}
