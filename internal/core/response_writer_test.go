package core

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestResponseWriter_Reset(t *testing.T) {
	var rw responseWriter
	w := httptest.NewRecorder()
	rw.reset(w)
	assert.Same(t, w, rw.Writer)
	assert.Eq(t, 0, rw.status)
}

func TestResponseWriter_WriteHeader_TracksStatus(t *testing.T) {
	var rw responseWriter
	rw.reset(httptest.NewRecorder())
	rw.WriteHeader(404)
	assert.Eq(t, 404, rw.status)
}

func TestResponseWriter_EnsureWriteHeader_DefaultsTo200(t *testing.T) {
	var rw responseWriter
	w := httptest.NewRecorder()
	rw.reset(w)
	rw.ensureWriteHeader()
	assert.Eq(t, 200, w.Code)
}

func TestResponseWriter_EnsureWriteHeader_RespectsExplicitStatus(t *testing.T) {
	var rw responseWriter
	w := httptest.NewRecorder()
	rw.reset(w)
	rw.WriteHeader(404)
	rw.ensureWriteHeader()
	assert.Eq(t, 404, w.Code)
}

type flushStatusWriter struct {
	http.ResponseWriter
	statuses []int
}

func (w *flushStatusWriter) WriteHeader(status int) {
	w.statuses = append(w.statuses, status)
	w.ResponseWriter.WriteHeader(status)
}

func (w *flushStatusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

type flushErrorStatusWriter struct {
	*flushStatusWriter
	err error
}

func (w *flushErrorStatusWriter) FlushError() error {
	_ = http.NewResponseController(w.ResponseWriter).Flush()
	return w.err
}

func (w *flushErrorStatusWriter) Flush() { panic("FlushError must take precedence") }

func TestResponseWriter_Flush_CommitsStatus(t *testing.T) {
	for _, status := range []int{0, http.StatusCreated} {
		name := "implicit OK"
		if status != 0 {
			name = http.StatusText(status)
		}
		for _, method := range []string{"Flush", "FlushError", "ResponseController"} {
			t.Run(name+"/"+method, func(t *testing.T) {
				rec := httptest.NewRecorder()
				writer := &flushStatusWriter{ResponseWriter: rec}
				var rw responseWriter
				rw.reset(writer)
				rw.WriteHeader(status)
				switch method {
				case "Flush":
					rw.Flush()
				case "FlushError":
					assert.NoErr(t, rw.FlushError())
				case "ResponseController":
					assert.NoErr(t, http.NewResponseController(&rw).Flush())
				}
				want := status
				if want == 0 {
					want = http.StatusOK
				}
				assert.Eq(t, want, rec.Code)
				assert.Eq(t, []int{want}, writer.statuses)
				assert.Eq(t, want, rw.Status())
				assert.True(t, rec.Flushed)
				assert.True(t, rw.Written())
				assert.Eq(t, 0, rw.Length())
				n, err := rw.Write([]byte("body"))
				assert.NoErr(t, err)
				assert.Eq(t, 4, n)
				assert.NoErr(t, rw.FlushError())
				assert.Eq(t, 4, rw.Length())
				assert.Eq(t, "body", rec.Body.String())
				assert.Eq(t, []int{want}, writer.statuses)
			})
		}
	}
}

func TestResponseWriter_FlushError_PreservesErrors(t *testing.T) {
	t.Run("supported flush returns its error", func(t *testing.T) {
		errFlush := errors.New("flush failed")
		writer := &flushErrorStatusWriter{
			flushStatusWriter: &flushStatusWriter{ResponseWriter: httptest.NewRecorder()},
			err:               errFlush,
		}
		var rw responseWriter
		rw.reset(writer)
		rw.WriteHeader(http.StatusCreated)
		assert.Same(t, errFlush, rw.FlushError())
		assert.Eq(t, []int{http.StatusCreated}, writer.statuses)
		assert.True(t, rw.Written())
	})
	t.Run("unsupported flush leaves the status deferred", func(t *testing.T) {
		writer := &flushStatusWriter{ResponseWriter: struct{ http.ResponseWriter }{httptest.NewRecorder()}}
		var rw responseWriter
		rw.reset(writer)
		rw.WriteHeader(http.StatusCreated)
		assert.True(t, errors.Is(rw.FlushError(), http.ErrNotSupported))
		assert.Empty(t, writer.statuses)
		assert.False(t, rw.Written())
	})
}

func TestResponseWriter_Flush_HTTP(t *testing.T) {
	for _, controller := range []bool{false, true} {
		t.Run(map[bool]string{false: "Flusher", true: "ResponseController"}[controller], func(t *testing.T) {
			var serverLog bytes.Buffer
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				var rw responseWriter
				rw.reset(w)
				rw.WriteHeader(http.StatusCreated)
				if controller {
					assert.NoErr(t, http.NewResponseController(&rw).Flush())
				} else {
					rw.Flush()
				}
				_, err := rw.Write([]byte("body"))
				assert.NoErr(t, err)
			}))
			server.Config.ErrorLog = log.New(&serverLog, "", 0)
			server.Start()
			defer server.Close()
			resp, err := server.Client().Get(server.URL)
			assert.NoErr(t, err)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			assert.NoErr(t, err)
			assert.Eq(t, http.StatusCreated, resp.StatusCode)
			assert.Eq(t, "body", string(body))
			assert.Eq(t, "", serverLog.String())
		})
	}
}

// fakeHijacker is a ResponseWriter test double that records WriteHeader calls
// and the status seen at the moment Hijack is invoked — letting us assert the
// recorded status is flushed to the underlying writer before connection takeover.
type fakeHijacker struct {
	header             http.Header
	wroteStatus        int // last status passed to WriteHeader; 0 if none
	hijacked           bool
	statusBeforeHijack int // wroteStatus captured at the moment Hijack ran
}

func (f *fakeHijacker) Header() http.Header {
	if f.header == nil {
		f.header = make(http.Header)
	}
	return f.header
}
func (f *fakeHijacker) Write(b []byte) (int, error) { return len(b), nil }
func (f *fakeHijacker) WriteHeader(code int)        { f.wroteStatus = code }

func (f *fakeHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	f.hijacked = true
	f.statusBeforeHijack = f.wroteStatus
	c1, _ := net.Pipe()
	brw := bufio.NewReadWriter(bufio.NewReader(c1), bufio.NewWriter(c1))
	return c1, brw, nil
}

func TestResponseWriter_Hijack(t *testing.T) {
	// WebSocket upgrade path: coder/websocket does WriteHeader(101) then Hijack().
	// The recorded 101 must reach the underlying writer before detaching the conn,
	// otherwise the handshake is lost and the client Dial hangs.
	t.Run("flushes recorded 101 before detaching", func(t *testing.T) {
		fh := &fakeHijacker{}
		var rw responseWriter
		rw.reset(fh)
		rw.WriteHeader(http.StatusSwitchingProtocols)

		conn, brw, err := rw.Hijack()
		assert.NoErr(t, err)
		assert.NotNil(t, conn)
		assert.NotNil(t, brw)
		assert.True(t, fh.hijacked)
		// 101 was written, and written BEFORE the underlying Hijack ran
		assert.Eq(t, http.StatusSwitchingProtocols, fh.wroteStatus)
		assert.Eq(t, http.StatusSwitchingProtocols, fh.statusBeforeHijack)
		_ = conn.Close()
	})

	// Raw TCP takeover with no explicit status must not synthesize a 200,
	// matching native http.ResponseWriter Hijack semantics.
	t.Run("raw hijack writes no status", func(t *testing.T) {
		fh := &fakeHijacker{}
		var rw responseWriter
		rw.reset(fh)

		conn, _, err := rw.Hijack()
		assert.NoErr(t, err)
		assert.True(t, fh.hijacked)
		assert.Eq(t, 0, fh.wroteStatus)
		assert.Eq(t, 0, rw.length) // length initialized on hijack
		_ = conn.Close()
	})
}
