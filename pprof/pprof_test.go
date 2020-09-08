package pprof

import (
	"os"
	"testing"

	"github.com/gookit/goutil/testutil"
	"github.com/gookit/rux"
	"github.com/stretchr/testify/assert"
)

func TestRouter_PProf(t *testing.T) {
	// skip run on local
	if os.Getenv("GOPROXY") != "" {
		return
	}

	r := rux.New(UsePProf)
	is := assert.New(t)

	w := testutil.MockRequest(r, "GET", "/debug/pprof/", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/heap", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/goroutine", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/block", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/threadcreate", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/cmdline", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/profile", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/symbol", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/mutex", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/trace", nil)
	is.Equal(200, w.Code)
	w = testutil.MockRequest(r, "GET", "/debug/404", nil)
	is.Equal(404, w.Code)
}
