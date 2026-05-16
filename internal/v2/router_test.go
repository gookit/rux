package v2

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestNewRouter_Defaults(t *testing.T) {
	r := New()
	assert.NotNil(t, r)
	assert.Eq(t, "default", r.Name)
	assert.False(t, r.Frozen())
}

func TestNewRouter_WithOptions(t *testing.T) {
	r := New(StrictLastSlash, HandleMethodNotAllowed)
	assert.True(t, r.strictLastSlash)
	assert.True(t, r.handleMethodNotAllowed)
}
