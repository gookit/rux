package pprof

import (
	"testing"

	"github.com/gookit/goutil/testutil"
	"github.com/gookit/rux"
	"github.com/stretchr/testify/assert"
)

func TestRouter_PProf(t *testing.T) {
	r := rux.New(UsePProf)
	is := assert.New(t)

	w := testutil.MockRequest(r, "GET", "/debug/pprof/", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/pprof/heap", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/pprof/goroutine", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/pprof/block", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/pprof/threadcreate", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/pprof/cmdline", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/pprof/profile", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/pprof/symbol", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/pprof/mutex", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/pprof/trace", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/pprof/404", nil)
	is.Equal(404, w.Code)
}
