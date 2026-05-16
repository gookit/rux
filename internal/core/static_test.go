package core

import (
	"net/http"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestRouter_StaticFile_RegistersGetRoute(t *testing.T) {
	r := New()
	r.StaticFile("/favicon.ico", "./testdata/favicon.ico")
	idx := methodIndex(GET)
	_, ok := r.staticRoutes[idx]["/favicon.ico"]
	assert.True(t, ok)
}

func TestRouter_StaticDir_RegistersWildcard(t *testing.T) {
	r := New()
	r.StaticDir("/assets", "./testdata")
	idx := methodIndex(GET)
	assert.NotNil(t, r.dynamicTrees[idx])
}

func TestRouter_StaticFS_RegistersWildcard(t *testing.T) {
	r := New()
	r.StaticFS("/files", http.Dir("./testdata"))
	idx := methodIndex(GET)
	assert.NotNil(t, r.dynamicTrees[idx])
}

func TestRouter_StaticFiles_RegistersWildcard(t *testing.T) {
	r := New()
	r.StaticFiles("/static", "./testdata", "")
	idx := methodIndex(GET)
	assert.NotNil(t, r.dynamicTrees[idx])
}
