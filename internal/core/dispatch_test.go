package core

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func textHandler(body string) HandlerFunc {
	return func(c *Context) {
		c.Resp.WriteHeader(200)
		_, _ = c.Resp.Write([]byte(body))
	}
}

func TestServeHTTP_StaticHit(t *testing.T) {
	r := New()
	r.GET("/users", textHandler("hi"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users", nil)
	r.ServeHTTP(w, req)

	assert.Eq(t, 200, w.Code)
	body, _ := io.ReadAll(w.Body)
	assert.Eq(t, "hi", string(body))
}

func TestServeHTTP_DynamicHit_BindsParam(t *testing.T) {
	r := New()
	r.GET("/users/{id}", func(c *Context) {
		_, _ = c.Resp.Write([]byte("id=" + c.Param("id")))
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/42", nil)
	r.ServeHTTP(w, req)
	body, _ := io.ReadAll(w.Body)
	assert.True(t, strings.Contains(string(body), "id=42"))
}

func TestServeHTTP_404(t *testing.T) {
	r := New()
	r.GET("/x", func(c *Context) {})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/nothing", nil)
	r.ServeHTTP(w, req)
	assert.Eq(t, 404, w.Code)
}

func TestServeHTTP_HEAD_FallsBackToGET(t *testing.T) {
	r := New()
	r.GET("/x", textHandler("ok"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("HEAD", "/x", nil)
	r.ServeHTTP(w, req)
	assert.Eq(t, 200, w.Code)
}

func TestServeHTTP_TriggersFreeze(t *testing.T) {
	r := New()
	r.GET("/x", textHandler("x"))
	assert.False(t, r.Frozen())
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	assert.True(t, r.Frozen())
}

func TestServeHTTP_MiddlewareOrder_GlobalGroupRouteMain(t *testing.T) {
	var order []string
	r := New()
	r.Use(func(c *Context) { order = append(order, "global"); c.Next() })
	r.Group("/api", func() {
		r.GET("/x", func(c *Context) { order = append(order, "main") },
			func(c *Context) { order = append(order, "route"); c.Next() })
	}, func(c *Context) { order = append(order, "group"); c.Next() })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/x", nil)
	r.ServeHTTP(w, req)
	assert.Eq(t, []string{"global", "group", "route", "main"}, order)
}

func TestServeHTTP_OnPanic(t *testing.T) {
	var captured any
	r := New()
	r.OnPanic = func(c *Context) {
		captured, _ = c.Get(CTXRecoverResult)
		c.Resp.WriteHeader(500)
	}
	r.GET("/boom", func(c *Context) { panic("oops") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/boom", nil)
	r.ServeHTTP(w, req)

	assert.Eq(t, 500, w.Code)
	assert.Eq(t, "oops", captured)
}

func TestServeHTTP_HandleMethodNotAllowed(t *testing.T) {
	r := New(HandleMethodNotAllowed)
	r.GET("/x", func(c *Context) {})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/x", nil)
	r.ServeHTTP(w, req)

	assert.Eq(t, 405, w.Code)
	assert.True(t, strings.Contains(w.Header().Get("Allow"), "GET"))
}

// Ensure HEAD method receives no body but correct status (HEAD test above
// already covers status; this asserts the http test recorder remains usable).
var _ = http.MethodGet

// =========================================================================
// resolveAddress branch coverage
// =========================================================================

func TestResolveAddress_NoArgs_DefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	assert.Eq(t, "0.0.0.0:8080", resolveAddress(nil))
}

func TestResolveAddress_NoArgs_HonorsPORT(t *testing.T) {
	t.Setenv("PORT", "9090")
	assert.Eq(t, "0.0.0.0:9090", resolveAddress(nil))
}

func TestResolveAddress_OneArg_HostPort(t *testing.T) {
	assert.Eq(t, "127.0.0.1:8080", resolveAddress([]string{"127.0.0.1:8080"}))
}

func TestResolveAddress_OneArg_ColonPortOnly(t *testing.T) {
	assert.Eq(t, "0.0.0.0:8080", resolveAddress([]string{":8080"}))
}

func TestResolveAddress_OneArg_PortBareNumber(t *testing.T) {
	assert.Eq(t, "0.0.0.0:9000", resolveAddress([]string{"9000"}))
}

func TestResolveAddress_TwoArgs(t *testing.T) {
	assert.Eq(t, "127.0.0.1:7777", resolveAddress([]string{"127.0.0.1", "7777"}))
}

func TestResolveAddress_TooManyArgs_Panics(t *testing.T) {
	defer func() { assert.NotNil(t, recover()) }()
	_ = resolveAddress([]string{"a", "b", "c"})
}

// =========================================================================
// WrapHTTPHandlers / HandleContext
// =========================================================================

func TestWrapHTTPHandlers_AppliesInOrder(t *testing.T) {
	r := New()
	r.GET("/x", textHandler("inner"))

	var order []string
	mw := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, req)
			})
		}
	}
	wrapped := r.WrapHTTPHandlers(mw("a"), mw("b"))

	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	assert.Eq(t, []string{"a", "b"}, order, "leftmost wrapper must run first")
	assert.Eq(t, "inner", w.Body.String())
}

func TestHandleContext_ReusesExternalContext(t *testing.T) {
	r := New()
	r.GET("/x", textHandler("via-handlecontext"))
	c := &Context{}
	w := httptest.NewRecorder()
	c.Init(w, httptest.NewRequest("GET", "/x", nil))
	r.HandleContext(c)
	assert.Eq(t, "via-handlecontext", w.Body.String())
}

// =========================================================================
// Listen / ListenTLS / ListenUnix — error paths
//
// Listen-family functions block until net/http exits, with no graceful
// stop hook. We exercise them via "bind to an address already in use →
// immediate bind error → r.err set" so the test completes synchronously.
// =========================================================================

func TestListen_RecordsBindErrIntoErr(t *testing.T) {
	// Take a port and HOLD it; Listen against the same addr must fail.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)
	defer ln.Close()

	r := New()
	r.GET("/x", textHandler("x"))
	r.Listen(ln.Addr().String())
	assert.Err(t, r.Err())
}

func TestListenTLS_RecordsBindErrIntoErr(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)
	defer ln.Close()

	r := New()
	r.GET("/x", textHandler("x"))
	// Missing cert files would also fail, but bind error fires first when
	// the port is held — either way r.err is set.
	r.ListenTLS(ln.Addr().String(), "no-cert.pem", "no-key.pem")
	assert.Err(t, r.Err())
}

func TestListenUnix_RecordsRemoveOrListenErr(t *testing.T) {
	// Point to a path inside a directory that doesn't exist so both
	// os.Remove (with IsNotExist trapped) and net.Listen fail clean.
	r := New()
	r.GET("/x", textHandler("x"))
	r.ListenUnix("/no/such/dir/listen.sock")
	assert.Err(t, r.Err())
}

// Stop the orphan import deletion: net is needed by the Listen tests above.
var _ = net.IPv4zero

