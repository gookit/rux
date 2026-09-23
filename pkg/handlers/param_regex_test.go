package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/rux/v2"
)

// hit runs one request through a route guarded by mw and returns the recorder.
func hit(t *testing.T, path string, mw rux.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	r := rux.New()
	r.GET("/users/{id}", func(c *rux.Context) {
		c.Text(200, "id="+c.Param("id"))
	}, mw)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	return w
}

func TestParamRegex_AcceptsMatch(t *testing.T) {
	w := hit(t, "/users/42", ParamRegex("id", `\d+`))
	assert.Eq(t, 200, w.Code)
	assert.Eq(t, "id=42", w.Body.String())
}

func TestParamRegex_RejectsNonMatch(t *testing.T) {
	w := hit(t, "/users/abc", ParamRegex("id", `\d+`))
	assert.Eq(t, 400, w.Code)
	assert.StrContains(t, w.Body.String(), "invalid path param: id")
}

// The pattern covers the whole value, so a partially numeric id is rejected.
func TestParamRegex_WholeValueMustMatch(t *testing.T) {
	assert.Eq(t, 400, hit(t, "/users/12abc", ParamRegex("id", `\d+`)).Code)
	assert.Eq(t, 400, hit(t, "/users/abc12", ParamRegex("id", `\d+`)).Code)
	// v1-style explicit anchors behave identically.
	assert.Eq(t, 200, hit(t, "/users/12", ParamRegex("id", `^\d+$`)).Code)
	assert.Eq(t, 400, hit(t, "/users/12abc", ParamRegex("id", `^\d+$`)).Code)
}

func TestParamRegex_Alternation(t *testing.T) {
	mw := ParamRegex("id", `\d+|all`)
	assert.Eq(t, 200, hit(t, "/users/all", mw).Code)
	assert.Eq(t, 200, hit(t, "/users/7", mw).Code)
	assert.Eq(t, 400, hit(t, "/users/ALL", mw).Code)
	assert.Eq(t, 400, hit(t, "/users/all7", mw).Code)
}

// A route without the guarded parameter fails closed.
func TestParamRegex_MissingParamFailsClosed(t *testing.T) {
	r := rux.New()
	r.GET("/health", func(c *rux.Context) { c.Text(200, "ok") }, ParamRegex("id", `\d+`))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
	assert.Eq(t, 400, w.Code)
}

func TestParamRegex_InvalidPatternPanicsAtRegistration(t *testing.T) {
	assert.Panics(t, func() { ParamRegex("id", "(") })
}

// The middleware must not leak the parameter value into the error body.
func TestParamRegex_ErrorBodyDoesNotEchoValue(t *testing.T) {
	w := hit(t, "/users/%3Cscript%3E", ParamRegex("id", `\d+`))
	assert.Eq(t, 400, w.Code)
	assert.False(t, strings.Contains(w.Body.String(), "<script>"))
}
