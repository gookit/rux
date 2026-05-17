// Package sse provides Server-Sent Events helpers for rux handlers.
//
// SSE is one of the simplest server-push protocols supported by every
// modern browser. This package wraps the wire-format and lifecycle so a
// handler only has to focus on producing events:
//
//	r.GET("/events", func(c *rux.Context) {
//	    _ = sse.Stream(c, nil, func(send sse.SendFunc, done <-chan struct{}) error {
//	        ticker := time.NewTicker(time.Second)
//	        defer ticker.Stop()
//	        for {
//	            select {
//	            case <-done:
//	                return nil
//	            case t := <-ticker.C:
//	                if err := send(sse.Event{Data: t.Format(time.RFC3339)}); err != nil {
//	                    return err
//	                }
//	            }
//	        }
//	    })
//	})
//
// Lifecycle hooks (auth, logging, metrics) plug in via the Hooks struct.
//
// NOTE: server.Server.WriteTimeout defaults to 30 s, which will kill a
// long-running SSE connection. For dedicated SSE servers either set
// WriteTimeout = 0, or route SSE handlers through a server instance with
// a longer budget.
package sse

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gookit/rux/v2"
)

// Event is a single SSE message frame. Empty fields are omitted from
// the wire output, so the zero value Event{} encodes to a heartbeat
// (an empty data: line followed by the terminating blank line).
type Event struct {
	// ID populates the "id:" field. Browsers echo the last seen ID back
	// in the Last-Event-ID header on reconnect.
	ID string
	// Name populates the "event:" field — used by EventSource.addEventListener.
	Name string
	// Data is the event body. Multi-line values are split into one
	// "data:" line per segment as required by the SSE spec.
	Data string
	// Retry, when > 0, populates the "retry:" field (reconnection delay
	// in milliseconds). Typically only set on the first event.
	Retry int
}

// SendFunc writes an event to the active stream. Returns an error if
// the underlying write fails (usually because the client disconnected),
// or if a hook chose to abort.
type SendFunc func(Event) error

// Producer is the user callback driving an SSE stream. It receives a
// send function and a done channel — when done fires (typically because
// the client disconnected), the producer should return promptly.
//
// Returning a non-nil error is reported via Hooks.OnDisconnect.
type Producer func(send SendFunc, done <-chan struct{}) error

// Hooks are optional callbacks fired at well-defined points in the
// stream lifecycle. Any field may be nil. Hooks run on the handler
// goroutine, so they must not block indefinitely.
type Hooks struct {
	// OnConnect runs after the SSE headers are written but before any
	// event is emitted. Return a non-nil error to abort the stream —
	// OnDisconnect still fires with that error, but the producer is
	// never invoked. Typical uses: authentication, rate-limit checks,
	// channel subscription bookkeeping.
	OnConnect func(c *rux.Context) error

	// OnDisconnect runs exactly once after the stream ends, whether
	// from a clean producer return, OnConnect rejection, write error,
	// or client disconnect. reason is nil only on a clean producer
	// return. Typical uses: subscription cleanup, audit logging.
	OnDisconnect func(c *rux.Context, reason error)

	// OnSend runs before each event is written. Modify *e in place to
	// adjust the outgoing event; return a non-nil error to skip it
	// (the error is reported via OnError; the stream continues).
	// Typical uses: tagging events with a request ID, filtering, metrics.
	OnSend func(c *rux.Context, e *Event) error

	// OnError reports per-event encode/skip errors that do not terminate
	// the stream. Fatal errors (write failures, client gone) bypass this
	// and go directly to OnDisconnect.
	OnError func(c *rux.Context, err error)
}

// ErrFlushNotSupported is returned by Stream when the underlying
// http.ResponseWriter does not implement http.Flusher (e.g. a buggy
// middleware wrapped it without preserving the interface).
var ErrFlushNotSupported = errors.New("sse: ResponseWriter does not support http.Flusher")

// ErrEventSkipped is the sentinel returned by SendFunc when an OnSend
// hook chose to skip the event. Callers usually treat this as a non-fatal.
var ErrEventSkipped = errors.New("sse: event skipped by OnSend hook")

// Stream upgrades c to an SSE connection and drives producer.
//
// Sequence:
//  1. Verify the writer supports Flusher (return ErrFlushNotSupported if not)
//  2. Call hooks.OnConnect; if it returns an error, abort BEFORE any SSE
//     headers are sent — the hook is free to write its own error response
//     (e.g. http.Error(c.Resp, ..., 401)) using c.Resp directly.
//  3. Write SSE response headers (Content-Type, Cache-Control, etc.) and
//     flush them so the client transitions out of "connecting" state.
//  4. Run producer until it returns or the client disconnects.
//  5. Call hooks.OnDisconnect with the final error (nil on clean exit).
//
// Passing nil hooks is fine — equivalent to &Hooks{}.
func Stream(c *rux.Context, hooks *Hooks, producer Producer) (retErr error) {
	if hooks == nil {
		hooks = &Hooks{}
	}

	flusher, ok := c.Resp.(http.Flusher)
	if !ok {
		retErr = ErrFlushNotSupported
		if hooks.OnDisconnect != nil {
			hooks.OnDisconnect(c, retErr)
		}
		return retErr
	}

	// OnConnect runs BEFORE we lock the status to 200, so a rejecting
	// hook can write its own 4xx/5xx error response via c.Resp.
	if hooks.OnConnect != nil {
		if err := hooks.OnConnect(c); err != nil {
			retErr = err
			if hooks.OnDisconnect != nil {
				hooks.OnDisconnect(c, retErr)
			}
			return retErr
		}
	}

	h := c.Resp.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	// Disable buffering for nginx / envoy / etc. — without this, proxies
	// hold events back until the buffer fills, defeating the point of SSE.
	h.Set("X-Accel-Buffering", "no")
	c.Resp.WriteHeader(http.StatusOK)

	// Write zero bytes to push the deferred status down through rux's
	// responseWriter wrapper into the underlying http.ResponseWriter.
	// Without this, the wrapper holds the status in memory and Flush
	// issues its own implicit WriteHeader(200), then our first real
	// Write triggers a "superfluous WriteHeader" warning.
	if _, err := c.Resp.Write(nil); err != nil {
		retErr = err
		if hooks.OnDisconnect != nil {
			hooks.OnDisconnect(c, retErr)
		}
		return retErr
	}

	// Flush the headers immediately so the client transitions out of
	// "connecting" state even before the first event lands.
	flusher.Flush()

	done := c.Req.Context().Done()

	send := func(e Event) error {
		if hooks.OnSend != nil {
			if err := hooks.OnSend(c, &e); err != nil {
				// Skip-but-continue: report via OnError, signal to producer.
				if hooks.OnError != nil {
					hooks.OnError(c, err)
				}
				return ErrEventSkipped
			}
		}
		if err := writeEvent(c.Resp, e); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	retErr = producer(send, done)
	if hooks.OnDisconnect != nil {
		hooks.OnDisconnect(c, retErr)
	}
	return retErr
}

// writeEvent serializes one Event to w in SSE wire format. Format:
//
//	id: <id>\n           (if ID non-empty)
//	event: <name>\n      (if Name non-empty)
//	retry: <ms>\n        (if Retry > 0)
//	data: <line1>\n      (one data: line per newline-separated segment)
//	data: <line2>\n
//	\n                   (terminating blank line marks frame boundary)
func writeEvent(w io.Writer, e Event) error {
	var b strings.Builder
	if e.ID != "" {
		b.WriteString("id: ")
		b.WriteString(e.ID)
		b.WriteByte('\n')
	}
	if e.Name != "" {
		b.WriteString("event: ")
		b.WriteString(e.Name)
		b.WriteByte('\n')
	}
	if e.Retry > 0 {
		b.WriteString("retry: ")
		b.WriteString(strconv.Itoa(e.Retry))
		b.WriteByte('\n')
	}
	// Always emit at least one data: line so heartbeats stay valid frames.
	if e.Data == "" {
		b.WriteString("data: \n")
	} else {
		for _, line := range strings.Split(e.Data, "\n") {
			b.WriteString("data: ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')

	_, err := io.WriteString(w, b.String())
	if err != nil {
		return fmt.Errorf("sse: write event: %w", err)
	}
	return nil
}
