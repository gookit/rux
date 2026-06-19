package core

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
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
func (f *fakeHijacker) WriteHeader(code int)         { f.wroteStatus = code }

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
