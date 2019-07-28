package rux

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelper(t *testing.T) {

	// resolveAddress
	addr := resolveAddress(nil)
	assert.Equal(t, "0.0.0.0:8080", addr)

	addr = resolveAddress([]string{"9090"})
	assert.Equal(t, "0.0.0.0:9090", addr)

	addr = resolveAddress([]string{"127.0.0.1:9090"})
	assert.Equal(t, "127.0.0.1:9090", addr)

	// use ENV

	// debugPrintError
	debugPrintError(errors.New("error msg"))

	// parseAccept
	ss := parseAccept("")
	assert.Len(t, ss, 0)

	ss = parseAccept(",")
	assert.Len(t, ss, 0)

	ss = parseAccept("application/json")
	assert.Len(t, ss, 1)
	assert.Equal(t, []string{"application/json"}, ss)
}
