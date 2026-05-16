package core

import (
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
