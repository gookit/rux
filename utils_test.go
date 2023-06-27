package rux

import (
	"errors"
	"os"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestHelper(t *testing.T) {
	// resolveAddress
	_ = os.Setenv("PORT", "")
	addr := resolveAddress(nil)
	assert.Eq(t, "0.0.0.0:8080", addr)

	addr = resolveAddress([]string{"9090"})
	assert.Eq(t, "0.0.0.0:9090", addr)

	addr = resolveAddress([]string{":9090"})
	assert.Eq(t, "0.0.0.0:9090", addr)

	addr = resolveAddress([]string{"127.0.0.1:9090"})
	assert.Eq(t, "127.0.0.1:9090", addr)

	// use ENV for resolveAddress
	mockEnvValue("PORT", "1234", func() {
		addr = resolveAddress(nil)
		assert.Eq(t, "0.0.0.0:1234", addr)
	})

	// debugPrintError
	debugPrintError(errors.New("error msg"))

	// parseAccept
	ss := parseAccept("")
	assert.Len(t, ss, 0)

	ss = parseAccept(",")
	assert.Len(t, ss, 0)

	ss = parseAccept("application/json")
	assert.Len(t, ss, 1)
	assert.Eq(t, []string{"application/json"}, ss)

}
