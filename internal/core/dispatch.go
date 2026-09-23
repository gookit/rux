package core

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

// CTXAllowedMethods is the context key carrying the []string of HTTP methods
// other than the request's method that would match the same path. The
// internal 405 handler reads it to build the Allow header.
const CTXAllowedMethods = "_allowedMethods"

// CTXRecoverResult is the context key carrying the recover() return value
// when OnPanic fires.
const CTXRecoverResult = "_recoverResult"

var internal404Handler HandlerFunc = func(c *Context) {
	http.NotFound(c.Resp, c.Req)
}

var internal405Handler HandlerFunc = func(c *Context) {
	if v, ok := c.Get(CTXAllowedMethods); ok {
		if list, ok := v.([]string); ok {
			sort.Strings(list)
			c.SetHeader("Allow", strings.Join(list, ", "))
		}
	}
	if c.Req.Method == OPTIONS {
		c.Resp.WriteHeader(200)
	} else {
		http.Error(c.Resp, "Method not allowed", 405)
	}
}

// fallbackChain returns the composed 404/405 chain (global middleware + user or
// internal handler). Freeze() always builds it; def is only used if a Context
// is dispatched through a router that was never frozen.
func fallbackChain(chain HandlersChain, def HandlerFunc) HandlersChain {
	if len(chain) == 0 {
		return HandlersChain{def}
	}
	return chain
}

// Bind binds the TCP listener for the resolved address without serving, so
// callers can read the real address before any traffic arrives. This is what
// makes an OS-assigned port (":0", "host:0") usable:
//
//	ln, err := r.Bind("127.0.0.1:0")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println("listening on", r.ListenAddr()) // 127.0.0.1:50447
//	r.ServeListener(ln)                         // blocks
//
// addr follows the same rules as Listen (default "0.0.0.0:8080", honors $PORT).
func (r *Router) Bind(addr ...string) (net.Listener, error) {
	ln, err := net.Listen("tcp", resolveAddress(addr))
	if err != nil {
		r.setListenState(nil, "", err)
		return nil, err
	}
	r.setListenState(ln, ln.Addr().String(), nil)
	return ln, nil
}

// Listen starts an HTTP server on the resolved address and blocks until the
// server exits; the error is stored in r.Err.
//
// The listener is bound before serving, so when addr asks for an OS-assigned
// port the resolved address is the one that gets printed and the one reported
// by ListenAddr / ListenPort / Listener (from another goroutine, or after Bind).
func (r *Router) Listen(addr ...string) {
	ln, err := r.Bind(addr...)
	if err != nil {
		return
	}
	fmt.Printf("Serve listen on %s\n", r.ListenAddr())
	r.serve(ln, "", "")
}

// ListenTLS starts an HTTPS server on the resolved address (see Listen).
func (r *Router) ListenTLS(addr, certFile, keyFile string) {
	ln, err := r.Bind(addr)
	if err != nil {
		return
	}
	fmt.Printf("Serve listen on %s (TLS)\n", r.ListenAddr())
	r.serve(ln, certFile, keyFile)
}

// ListenUnix starts an HTTP server on a Unix domain socket.
func (r *Router) ListenUnix(file string) {
	if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
		r.setListenState(nil, "", err)
		return
	}
	ln, err := net.Listen("unix", file)
	if err != nil {
		r.setListenState(nil, "", err)
		return
	}
	r.setListenState(ln, ln.Addr().String(), nil)
	r.serve(ln, "", "")
}

// ServeListener serves HTTP on an already-bound listener and blocks until the
// server exits; the error is stored in r.Err. The listener's address is
// reflected into ListenAddr / ListenPort.
//
// Use it with Bind, or when the socket is owned by someone else (systemd
// socket activation, a port pre-bound by a test, a Unix socket). For TLS,
// call http.ServeTLS(ln, r, certFile, keyFile) directly: Router is an
// http.Handler.
func (r *Router) ServeListener(ln net.Listener) {
	r.setListenState(ln, ln.Addr().String(), nil)
	r.serve(ln, "", "")
}

// Listener returns the listener the router is serving on, or nil when it is not
// serving (before Bind/Listen, or after the server exited).
func (r *Router) Listener() net.Listener {
	r.listenMu.Lock()
	defer r.listenMu.Unlock()
	return r.ln
}

// ListenAddr returns the resolved "host:port" the router is serving on, e.g.
// "127.0.0.1:50447" or "[::]:50447" for a wildcard bind, or "" when nothing has
// been bound.
func (r *Router) ListenAddr() string {
	r.listenMu.Lock()
	defer r.listenMu.Unlock()
	return r.listenAddr
}

// ListenPort returns the resolved TCP port, or 0 when nothing is bound.
func (r *Router) ListenPort() int {
	r.listenMu.Lock()
	addr := r.listenAddr
	r.listenMu.Unlock()

	if _, portStr, err := net.SplitHostPort(addr); err == nil {
		if port, err := strconv.Atoi(portStr); err == nil {
			return port
		}
	}
	return 0
}

// serve runs the HTTP (or TLS) server on ln until it exits, then records the
// error and drops the listener reference.
func (r *Router) serve(ln net.Listener, certFile, keyFile string) {
	var err error
	if certFile != "" || keyFile != "" {
		err = http.ServeTLS(ln, r, certFile, keyFile)
	} else {
		err = http.Serve(ln, r) // http.Serve closes ln on return
	}
	r.setListenState(nil, "", err)
}

// setListenState updates the serving state under the router's listen mutex. An
// empty addr keeps the previously resolved address.
func (r *Router) setListenState(ln net.Listener, addr string, err error) {
	r.listenMu.Lock()
	r.ln = ln
	if addr != "" {
		r.listenAddr = addr
	}
	r.err = err
	r.listenMu.Unlock()
}

// WrapHTTPHandlers wraps the router in zero or more net/http middlewares.
// The leftmost wrapper runs first.
func (r *Router) WrapHTTPHandlers(preHandlers ...func(http.Handler) http.Handler) http.Handler {
	var wrapped http.Handler = r
	for i := len(preHandlers) - 1; i >= 0; i-- {
		wrapped = preHandlers[i](wrapped)
	}
	return wrapped
}

// ServeHTTP triggers lazy Freeze on first call.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !r.frozen.Load() {
		r.Freeze()
	}
	ctx := r.ctxPool.Get().(*Context)
	ctx.Init(w, req)
	r.handle(ctx)
	r.ctxPool.Put(ctx)
}

// HandleContext re-uses an externally constructed Context.
func (r *Router) HandleContext(c *Context) {
	if !r.frozen.Load() {
		r.Freeze()
	}
	r.handle(c)
	r.ctxPool.Put(c)
}

// handle is the core dispatch — runs middleware/route chain, falls back to
// 404 / 405 handlers, and finally ensures a status code is written.
func (r *Router) handle(ctx *Context) {
	// Always flush status, even on a recovered panic path.
	defer ctx.writer.ensureWriteHeader()

	if r.OnPanic != nil {
		defer func() {
			if rec := recover(); rec != nil {
				ctx.Set(CTXRecoverResult, rec)
				r.OnPanic(ctx)
			}
		}()
	}

	path := ctx.Req.URL.Path
	if r.useEncodedPath {
		path = ctx.Req.URL.EscapedPath()
	}
	if r.interceptAll != "" {
		path = r.interceptAll
	} else {
		path = r.formatPath(path)
	}

	method := ctx.Req.Method
	idx := methodIndex(method)

	var route *Route
	if idx >= 0 {
		if m := r.staticRoutes[idx]; m != nil {
			route = m[path]
		}
		if route == nil {
			if tree := r.dynamicTrees[idx]; tree != nil {
				if r2, ok := tree.lookup(path, &ctx.params); ok {
					route = r2
				}
			}
		}
	}

	if route != nil {
		ctx.matchedRoute = route
		ctx.matchedPath = path
		ctx.SetHandlers(route.finalChain)
		ctx.Next()
	} else {
		dispatched := false
		if r.handleFallbackRoute && idx >= 0 {
			if m := r.staticRoutes[idx]; m != nil {
				if fb, ok := m["/*"]; ok {
					ctx.SetHandlers(fb.finalChain)
					ctx.Next()
					dispatched = true
				}
			}
		}
		if !dispatched && r.handleMethodNotAllowed {
			allowed := r.findAllowedMethods(method, path)
			if len(allowed) > 0 {
				ctx.Set(CTXAllowedMethods, allowed)
				ctx.SetHandlers(fallbackChain(r.noAllowedChain, internal405Handler))
				ctx.Next()
				dispatched = true
			}
		}
		if !dispatched {
			ctx.SetHandlers(fallbackChain(r.noRouteChain, internal404Handler))
			ctx.Next()
		}
	}

	if r.OnError != nil && len(ctx.Errors) > 0 {
		r.OnError(ctx)
	}
}

// findAllowedMethods returns the set of HTTP methods (other than the
// rejected method) that would match path. Used for the Allow header on 405.
func (r *Router) findAllowedMethods(method, path string) []string {
	var allowed []string
	for _, m := range []string{GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS, CONNECT, TRACE} {
		if m == method {
			continue
		}
		idx := methodIndex(m)
		if idx < 0 {
			continue
		}
		if r.staticRoutes[idx] != nil {
			if _, ok := r.staticRoutes[idx][path]; ok {
				allowed = append(allowed, m)
				continue
			}
		}
		if tree := r.dynamicTrees[idx]; tree != nil {
			var ps Params
			if _, ok := tree.lookup(path, &ps); ok {
				allowed = append(allowed, m)
			}
		}
	}
	return allowed
}

// resolveAddress turns user-supplied addr arguments into a single "ip:port".
func resolveAddress(addr []string) string {
	ip := "0.0.0.0"
	switch len(addr) {
	case 0:
		if port := os.Getenv("PORT"); port != "" {
			return ip + ":" + port
		}
		return ip + ":8080"
	case 1:
		if strings.IndexByte(addr[0], ':') != -1 {
			ss := strings.SplitN(addr[0], ":", 2)
			if ss[0] != "" {
				return addr[0]
			}
			return ip + ":" + ss[1]
		}
		return ip + ":" + addr[0]
	case 2:
		return addr[0] + ":" + addr[1]
	default:
		panic("rux: too many addr arguments")
	}
}
