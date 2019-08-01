package rux

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseWriter_WriteHeader(t *testing.T) {
	c := mockContext(GET, "/", nil, nil)

	c.WriteBytes([]byte("hi"))
	c.SetStatus(200)
	c.SetStatus(201)

	assert.Equal(t, 201, c.StatusCode())
}

func TestResponseWriter_Flush(t *testing.T) {
	c := mockContext(GET, "/", nil, nil)
	c.WriteBytes([]byte("hi"))
	c.writer.Flush()

	assert.True(t, c.writer.Writer.(*httptest.ResponseRecorder).Flushed)
}
