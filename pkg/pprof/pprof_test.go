package pprof

import (
	"os"
	"testing"

	"github.com/gookit/goutil/testutil"
	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/rux"
)

func TestRouter_PProf(t *testing.T) {
	// skip run on local
	if os.Getenv("USER") == "inhere" {
		return
	}

	r := rux.New(UsePProf)
	is := assert.New(t)

	w := testutil.MockRequest(r, "GET", "/debug/pprof/", nil)
	is.Eq(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/heap", nil)
	is.Eq(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/goroutine", nil)
	is.Eq(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/block", nil)
	is.Eq(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/threadcreate", nil)
	is.Eq(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/cmdline", nil)
	is.Eq(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/profile", nil)
	is.Eq(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/symbol", nil)
	is.Eq(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/mutex", nil)
	is.Eq(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/trace", nil)
	is.Eq(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/404", nil)
	is.Eq(404, w.Code)
}
