package v2

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
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

// Listen starts an HTTP server on the resolved address (errors stored in r.Err).
func (r *Router) Listen(addr ...string) {
	address := resolveAddress(addr)
	fmt.Printf("Serve listen on %s\n", address)
	r.err = http.ListenAndServe(address, r)
}

// ListenTLS starts an HTTPS server.
func (r *Router) ListenTLS(addr, certFile, keyFile string) {
	address := resolveAddress([]string{addr})
	fmt.Printf("Serve listen on %s (TLS)\n", address)
	r.err = http.ListenAndServeTLS(address, certFile, keyFile, r)
}

// ListenUnix starts an HTTP server on a Unix domain socket.
func (r *Router) ListenUnix(file string) {
	if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
		r.err = err
		return
	}
	listener, err := net.Listen("unix", file)
	if err != nil {
		r.err = err
		return
	}
	r.err = http.Serve(listener, r)
	_ = listener.Close()
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
				if len(r.noAllowed) == 0 {
					r.noAllowed = HandlersChain{internal405Handler}
				}
				ctx.Set(CTXAllowedMethods, allowed)
				ctx.SetHandlers(r.noAllowed)
				ctx.Next()
				dispatched = true
			}
		}
		if !dispatched {
			if len(r.noRoute) == 0 {
				r.noRoute = HandlersChain{internal404Handler}
			}
			ctx.SetHandlers(r.noRoute)
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
