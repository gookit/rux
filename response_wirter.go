package rux

import (
	"bufio"
	"net"
	"net/http"
)

const noWritten = -1

type responseWriter struct {
	Writer http.ResponseWriter
	status int
	length int
	// mark header is wrote
	// wroteHeader bool
}

// reset the writer
func (w *responseWriter) reset(w2 http.ResponseWriter) {
	w.status = 0
	w.length = noWritten
	w.Writer = w2
}

// Status get status code
func (w *responseWriter) Status() int {
	return w.status
}

// Written has written content ?
func (w *responseWriter) Written() bool {
	return w.length != noWritten
}

// Length get written length
func (w *responseWriter) Length() int {
	return w.length
}

// Header get Header
// Tips: implement the http.ResponseWriter interface.
func (w *responseWriter) Header() http.Header {
	return w.Writer.Header()
}

// WriteHeader write status code
// Tips: implement the http.ResponseWriter interface.
func (w *responseWriter) WriteHeader(status int) {
	if status > 0 && w.status != status {
		if w.Written() {
			debugPrint(
				"[WARNING] Headers were already written. Wanted to override status code %d with %d",
				w.status,
				status,
			)
		}

		w.status = status
	}

	// Don't write, real write on ensureWriteHeader()
	// w.Writer.WriteHeader(status)
}

// Write data to response writer
// Tips: implement the http.ResponseWriter interface.
func (w *responseWriter) Write(b []byte) (n int, err error) {
	w.ensureWriteHeader()

	n, err = w.Writer.Write(b)
	w.length += n
	return
}

// WriteString write string.
// func (w *responseWriter) WriteString(s string) (n int, err error) {
// 	w.ensureWriteHeader()
//
// 	n, err = io.WriteString(w.Writer, s)
// 	w.length += n
// 	return
// }

// Flush get status code
// Tips: implement the http.Flusher interface.
func (w *responseWriter) Flush() {
	w.Writer.(http.Flusher).Flush()
}

// Hijack implements the http.Hijacker interface.
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.length < 0 {
		w.length = 0
	}

	return w.Writer.(http.Hijacker).Hijack()
}

// ensureWriteHeader ensure write header status
func (w *responseWriter) ensureWriteHeader() {
	if !w.Written() {
		if w.status == 0 {
			w.status = 200
		}

		w.length = 0
		w.Writer.WriteHeader(w.status)
	}
}
