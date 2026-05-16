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

/*************************************************************
 * Optional Segment Tests
 *************************************************************/

func TestParseOptionalSegments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple optional param",
			input:    "/posts[/{id}]",
			expected: []string{"/posts", "/posts/:id"},
		},
		{
			name:     "optional param with prefix",
			input:    "/api/users[/{name}]",
			expected: []string{"/api/users", "/api/users/:name"},
		},
		{
			name:     "optional with content after",
			input:    "/api/users[/{name}]/profile",
			expected: []string{"/api/users/profile", "/api/users/:name/profile"},
		},
		{
			name:     "no optional segment",
			input:    "/posts/{id}",
			expected: []string{"/posts/:id"},
		},
		{
			name:     "empty optional",
			input:    "/posts[]",
			expected: []string{"/posts", "/posts"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOptionalSegments(tt.input)
			assert.Eq(t, tt.expected, result)
		})
	}
}
