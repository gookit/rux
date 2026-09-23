package core

import (
	"fmt"
	"net/http"
)

// Group is a route group value: a path prefix plus the middleware every route
// registered through it inherits.
//
// Router.Group(prefix, fn, middles...) is the closure form; Router.NewGroup
// returns this value form, which reads closer to gin:
//
//	api := r.NewGroup("/api", handlers.BearerAuth())
//	api.GET("/users", listUsers) // GET /api/users, auth runs first
//
//	admin := api.NewGroup("/admin", isAdmin())
//	admin.DELETE("/users/{id}", deleteUser) // DELETE /api/admin/users/{id}
//
// Middleware order inside a request: global -> group (outer to inner) -> route
// -> handler.
//
// A Group captures the prefix and middleware of the moment it is created, so it
// can be stored and used later. Registration must still happen before the router
// freezes, exactly like Router.GET.
type Group struct {
	router  *Router
	prefix  string
	middles HandlersChain
}

// NewGroup returns a Group for prefix. It inherits the enclosing group context
// (so it can be called inside a closure-style Group) and appends middles after
// the inherited middleware.
func (r *Router) NewGroup(prefix string, middles ...HandlerFunc) *Group {
	return &Group{
		router:  r,
		prefix:  r.currentGroupPrefix + r.formatPath(prefix),
		middles: mergeChain(r.currentGroupHandlers, middles...),
	}
}

// NewGroup derives a nested group: prefixes concatenate, middleware stacks.
func (g *Group) NewGroup(prefix string, middles ...HandlerFunc) *Group {
	return &Group{
		router:  g.router,
		prefix:  g.prefix + g.router.formatPath(prefix),
		middles: mergeChain(g.middles, middles...),
	}
}

// Router returns the router this group registers routes on.
func (g *Group) Router() *Router { return g.router }

// Prefix returns the group's path prefix, including any parent prefix.
func (g *Group) Prefix() string { return g.prefix }

// Use appends middleware to the group and returns it for chaining. Routes
// registered afterwards inherit it; routes registered earlier keep the
// middleware set they were created with.
func (g *Group) Use(middlewares ...HandlerFunc) *Group {
	g.middles = mergeChain(g.middles, middlewares...)
	return g
}

// add registers one route through this group's context.
func (g *Group) add(path string, h HandlerFunc, methods []string, mw ...HandlerFunc) *Route {
	route := newRoute(path, h, methods)
	route.groupPrefix = g.prefix
	route.groupChain = g.middles
	return g.router.AddRoute(route).Use(mw...)
}

// Add registers a route on the given methods (defaults to GET when empty).
func (g *Group) Add(path string, h HandlerFunc, methods ...string) *Route {
	return g.add(path, h, methods)
}

// AddNamed registers a named route through this group's context.
func (g *Group) AddNamed(name, path string, h HandlerFunc, methods ...string) *Route {
	route := newNamedRoute(name, path, h, methods)
	route.groupPrefix = g.prefix
	route.groupChain = g.middles
	return g.router.AddRoute(route)
}

// Verb shortcuts, mirroring the Router methods.

func (g *Group) GET(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return g.add(path, h, []string{GET}, mw...)
}
func (g *Group) HEAD(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return g.add(path, h, []string{HEAD}, mw...)
}
func (g *Group) POST(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return g.add(path, h, []string{POST}, mw...)
}
func (g *Group) PUT(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return g.add(path, h, []string{PUT}, mw...)
}
func (g *Group) PATCH(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return g.add(path, h, []string{PATCH}, mw...)
}
func (g *Group) DELETE(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return g.add(path, h, []string{DELETE}, mw...)
}
func (g *Group) OPTIONS(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return g.add(path, h, []string{OPTIONS}, mw...)
}
func (g *Group) CONNECT(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return g.add(path, h, []string{CONNECT}, mw...)
}
func (g *Group) TRACE(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return g.add(path, h, []string{TRACE}, mw...)
}

// Any registers a route on every supported HTTP method.
func (g *Group) Any(path string, h HandlerFunc, mw ...HandlerFunc) *Route {
	return g.add(path, h, anyMethods, mw...)
}

// Static helpers, mirroring the Router methods.

// StaticFile serves one file under the group.
func (g *Group) StaticFile(path, filePath string) *Route {
	return g.GET(path, func(c *Context) { c.File(filePath) })
}

// StaticDir serves files from rootDir under prefixURL inside the group.
func (g *Group) StaticDir(prefixURL, rootDir string) *Route {
	return g.addStatic(prefixURL, http.FileServer(http.Dir(rootDir)))
}

// StaticFS serves files from the given http.FileSystem under prefixURL.
func (g *Group) StaticFS(prefixURL string, fs http.FileSystem) *Route {
	return g.addStatic(prefixURL, http.FileServer(fs))
}

// StaticFiles serves files from rootDir under prefixURL. The exts argument is
// reserved for future extension filtering and is currently ignored.
func (g *Group) StaticFiles(prefixURL, rootDir, exts string) *Route {
	_ = exts // reserved for future extension filtering
	fs := http.FileServer(http.Dir(rootDir))
	return g.GET(fmt.Sprintf("%s/*file", prefixURL), func(c *Context) {
		c.Req.URL.Path = c.Param("file")
		fs.ServeHTTP(c.Resp, c.Req)
	})
}

// addStatic registers a file-server route, stripping the group-prefixed URL so
// the request path matches what the router actually routed.
func (g *Group) addStatic(prefixURL string, fileHandler http.Handler) *Route {
	fh := http.StripPrefix(g.fullPath(prefixURL), fileHandler)
	return g.GET(prefixURL+"/*file", func(c *Context) {
		fh.ServeHTTP(c.Resp, c.Req)
	})
}

// fullPath resolves a group-relative path against the group prefix.
func (g *Group) fullPath(path string) string {
	return g.router.formatPath(g.prefix + g.router.formatPath(path))
}

// mergeChain returns base followed by extra in a fresh slice, so the caller can
// keep appending without aliasing base.
func mergeChain(base HandlersChain, extra ...HandlerFunc) HandlersChain {
	if len(extra) == 0 {
		return base
	}
	out := make(HandlersChain, 0, len(base)+len(extra))
	out = append(out, base...)
	return append(out, extra...)
}
