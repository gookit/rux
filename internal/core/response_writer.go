package core

import (
	"bufio"
	"net"
	"net/http"
)

const noWritten = -1

// responseWriter wraps http.ResponseWriter to defer status emission until
// Write or ensureWriteHeader is called. This lets the dispatch layer set a
// default 200 status if no handler wrote one explicitly.
type responseWriter struct {
	Writer http.ResponseWriter
	status int
	length int
}

func (w *responseWriter) reset(w2 http.ResponseWriter) {
	w.Writer = w2
	w.status = 0
	w.length = noWritten
}

func (w *responseWriter) Status() int   { return w.status }
func (w *responseWriter) Length() int   { return w.length }
func (w *responseWriter) Written() bool { return w.length != noWritten }

func (w *responseWriter) Header() http.Header { return w.Writer.Header() }

func (w *responseWriter) WriteHeader(status int) {
	if status > 0 && w.status != status {
		w.status = status
	}
	// Don't write yet — ensureWriteHeader does it.
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.ensureWriteHeader()
	n, err := w.Writer.Write(b)
	w.length += n
	return n, err
}

func (w *responseWriter) Flush() {
	if err := w.FlushError(); err != nil {
		panic("rux: underlying http.ResponseWriter does not implement http.Flusher")
	}
}

// FlushError flushes buffered data and reports whether the underlying writer
// supports flushing instead of panicking. http.ResponseController prefers this
// over Flush, so wrappers like sse.Stream can report a clean
// ErrFlushNotSupported even though this wrapper always implements Flusher.
//
// Delegating to the controller also covers middlewares that wrap the writer
// without promoting Flusher but do expose Unwrap.
func (w *responseWriter) FlushError() error {
	return http.NewResponseController(w.Writer).Flush()
}

// Unwrap exposes the wrapped writer so http.ResponseController (and anything
// else that walks Unwrap chains) can reach SetWriteDeadline / Flush / Hijack on
// the real http.ResponseWriter. Long-lived responses (SSE, WebSocket, big
// downloads) need SetWriteDeadline to escape a server-level WriteTimeout.
func (w *responseWriter) Unwrap() http.ResponseWriter { return w.Writer }

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.Writer.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	// Flush an explicitly recorded status (e.g. WebSocket 101) to the underlying
	// writer before detaching the connection. Otherwise the deferred WriteHeader
	// is lost — the handshake never reaches the socket and clients hang. A status
	// of 0 means a raw hijack, so nothing is written (matches native semantics).
	if w.status != 0 && !w.Written() {
		w.Writer.WriteHeader(w.status)
	}
	if w.length < 0 {
		w.length = 0
	}
	return hj.Hijack()
}

// ensureWriteHeader emits the actual status code (defaults to 200) and
// initializes length tracking. Idempotent via the Written() guard.
func (w *responseWriter) ensureWriteHeader() {
	if !w.Written() {
		if w.status == 0 {
			w.status = 200
		}
		w.length = 0
		w.Writer.WriteHeader(w.status)
	}
}
