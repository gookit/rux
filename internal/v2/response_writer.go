package v2

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
	w.Writer.(http.Flusher).Flush()
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.length < 0 {
		w.length = 0
	}
	return w.Writer.(http.Hijacker).Hijack()
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
