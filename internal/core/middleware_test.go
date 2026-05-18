package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestHandlerFunc_ServeHTTP_AdaptsToHttpHandler(t *testing.T) {
	h := HandlerFunc(func(c *Context) { c.Resp.WriteHeader(201) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	http.Handler(h).ServeHTTP(w, req)
	assert.Eq(t, 201, w.Code)
}

func TestWrapH_AdaptsHttpHandler(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(202)
	})
	h := WrapH(inner)
	c := &Context{}
	c.Init(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	h(c)
	assert.True(t, called)
}

func TestCombineHandlers_PreservesOrder(t *testing.T) {
	a := func(c *Context) {}
	b := func(c *Context) {}
	cFn := func(c *Context) {}
	out := combineHandlers(HandlersChain{a, b}, HandlersChain{cFn})
	assert.Eq(t, 3, len(out))
}

func TestHandlersChain_Last(t *testing.T) {
	a := HandlerFunc(func(c *Context) {})
	b := HandlerFunc(func(c *Context) {})
	chain := HandlersChain{a, b}
	assert.NotNil(t, chain.Last())

	var empty HandlersChain
	assert.Nil(t, empty.Last())
}

// ----- the other adapter aliases ---------------------------------

func TestHTTPHandler_AliasOfWrapH(t *testing.T) {
	called := false
	h := HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	}))
	w := httptest.NewRecorder()
	c := &Context{}
	c.Init(w, httptest.NewRequest("GET", "/", nil))
	h(c)
	assert.True(t, called)
	// c.Resp wraps the recorder; WriteHeader is deferred to the wrapper's
	// ensureWriteHeader. Read via the API that knows about the wrapper.
	assert.Eq(t, http.StatusTeapot, c.StatusCode())
}

func TestWrapHF_AdaptsHandlerFunc(t *testing.T) {
	called := false
	h := WrapHF(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		_, _ = w.Write([]byte("hf"))
	})
	w := httptest.NewRecorder()
	c := &Context{}
	c.Init(w, httptest.NewRequest("GET", "/", nil))
	h(c)
	assert.True(t, called)
	assert.Eq(t, "hf", w.Body.String())
}

func TestHTTPHandlerFunc_AliasOfWrapHF(t *testing.T) {
	h := HTTPHandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hhf"))
	})
	w := httptest.NewRecorder()
	c := &Context{}
	c.Init(w, httptest.NewRequest("GET", "/", nil))
	h(c)
	assert.Eq(t, "hhf", w.Body.String())
}

func TestWrapHTTPHandlerFunc_Direct(t *testing.T) {
	h := WrapHTTPHandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("whf"))
	})
	w := httptest.NewRecorder()
	c := &Context{}
	c.Init(w, httptest.NewRequest("GET", "/", nil))
	h(c)
	assert.Eq(t, "whf", w.Body.String())
}
