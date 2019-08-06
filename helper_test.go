package rux

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelper(t *testing.T) {
	// resolveAddress
	_ = os.Setenv("PORT", "")
	addr := resolveAddress(nil)
	assert.Equal(t, "0.0.0.0:8080", addr)

	addr = resolveAddress([]string{"9090"})
	assert.Equal(t, "0.0.0.0:9090", addr)

	addr = resolveAddress([]string{":9090"})
	assert.Equal(t, "0.0.0.0:9090", addr)

	addr = resolveAddress([]string{"127.0.0.1:9090"})
	assert.Equal(t, "127.0.0.1:9090", addr)

	// use ENV for resolveAddress
	mockEnvValue("PORT", "1234", func() {
		addr = resolveAddress(nil)
		assert.Equal(t, "0.0.0.0:1234", addr)
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
	assert.Equal(t, []string{"application/json"}, ss)

	ts := toString("test-string")
	assert.Equal(t, ts, "test-string")

	ts = toString(20)
	assert.Equal(t, ts, "20")

	ts = toString(int(30))
	assert.Equal(t, ts, "30")

	ts = toString(int8(30))
	assert.Equal(t, ts, "30")

	ts = toString(int16(30))
	assert.Equal(t, ts, "30")

	ts = toString(int32(30))
	assert.Equal(t, ts, "30")

	ts = toString(int64(30))
	assert.Equal(t, ts, "30")

	ts = toString(uint(30))
	assert.Equal(t, ts, "30")

	ts = toString(uint8(30))
	assert.Equal(t, ts, "30")

	ts = toString(uint16(30))
	assert.Equal(t, ts, "30")

	ts = toString(uint32(30))
	assert.Equal(t, ts, "30")

	ts = toString(uint64(30))
	assert.Equal(t, ts, "30")

	ts = toString(float32(30.00))
	assert.Equal(t, ts, "30")

	ts = toString(float64(30.00))
	assert.Equal(t, ts, "30")

	ts = toString(true)
	assert.Equal(t, ts, "true")

	ts = toString(false)
	assert.Equal(t, ts, "false")

	ts = toString([]byte{'t', 'e', 's', 't'})
	assert.Equal(t, ts, "test")

	ts = toString(nil)
	assert.Equal(t, ts, "")

	testUsername := struct {
		Username string
	}{
		Username: "Test",
	}

	testUsernameJson, err := json.Marshal(testUsername)

	ts = toString(testUsername)
	assert.Nil(t, err)
	assert.Equal(t, ts, string(testUsernameJson))
}
