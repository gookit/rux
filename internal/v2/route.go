package v2

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gookit/goutil"
)

// HandlerFunc is the standard handler signature.
type HandlerFunc func(c *Context)

// HandlersChain is a list of handlers (middlewares + final handler).
// The final handler is the last element — there is no separate field.
type HandlersChain []HandlerFunc

// abortIndex marks an aborted handler chain (set by Context.Abort).
const abortIndex int8 = 63

// RouteInfo is a snapshot of a Route's public information.
type RouteInfo struct {
	Name        string
	Path        string
	HandlerName string
	Methods     []string
	HandlerNum  int
}

// Route describes a single registered route.
//
// chain is the user-supplied middlewares followed by the main handler.
// finalChain is populated by Router.Freeze() and is what dispatch reads.
type Route struct {
	name    string
	path    string
	methods []string

	// originalPath preserves the registered path with {name} placeholders for
	// URL-building. path is rewritten to colon syntax at registration time.
	originalPath string

	chain      HandlersChain
	finalChain HandlersChain

	Opts map[string]any
}

// newRoute creates a Route with the main handler appended to chain.
// Panics if handler is nil. Defaults methods to []string{GET} when empty.
func newRoute(path string, handler HandlerFunc, methods []string) *Route {
	if handler == nil {
		panic(fmt.Sprintf("rux: route handler cannot be nil (path: %q)", path))
	}
	if len(methods) == 0 {
		methods = []string{GET}
	} else {
		methods = formatMethods(methods)
	}
	validateMethods(methods)
	return &Route{
		path:    simpleFmtPath(path),
		methods: methods,
		chain:   HandlersChain{handler},
	}
}

// newNamedRoute is like newRoute but assigns a name.
func newNamedRoute(name, path string, handler HandlerFunc, methods []string) *Route {
	r := newRoute(path, handler, methods)
	r.name = strings.TrimSpace(name)
	return r
}

// Use prepends middleware handlers to this route.
// Order: middlewares run before the main handler.
func (r *Route) Use(middlewares ...HandlerFunc) *Route {
	if len(middlewares) == 0 {
		return r
	}
	if len(r.chain)+len(middlewares) >= int(abortIndex) {
		goutil.Panicf("rux: too many handlers (limit %d)", abortIndex)
	}
	main := r.chain[len(r.chain)-1]
	// Drop main, append middlewares, re-append main at the end.
	r.chain = append(r.chain[:len(r.chain)-1], middlewares...)
	r.chain = append(r.chain, main)
	return r
}

// Name returns the route's name.
func (r *Route) Name() string { return r.name }

// Path returns the route's path.
func (r *Route) Path() string { return r.path }

// Methods returns the route's allowed methods.
func (r *Route) Methods() []string { return r.methods }

// MethodString joins allowed methods with sep.
func (r *Route) MethodString(sep string) string { return strings.Join(r.methods, sep) }

// Handler returns the main handler (last element of chain).
func (r *Route) Handler() HandlerFunc {
	if len(r.chain) == 0 {
		return nil
	}
	return r.chain[len(r.chain)-1]
}

// Handlers returns the user-supplied middlewares (excluding main handler).
func (r *Route) Handlers() HandlersChain {
	if len(r.chain) <= 1 {
		return nil
	}
	return r.chain[:len(r.chain)-1]
}

// HandlerName returns the symbolic name of the main handler.
func (r *Route) HandlerName() string {
	return goutil.FuncName(r.Handler())
}

// String returns a debug representation of the route.
func (r *Route) String() string {
	return fmt.Sprintf("%-15s %-38s --> %s (%d middleware)",
		r.MethodString(","), r.path, r.HandlerName(), len(r.chain)-1)
}

// Info returns a RouteInfo snapshot.
func (r *Route) Info() RouteInfo {
	return RouteInfo{r.name, r.path, r.HandlerName(), r.methods, len(r.chain) - 1}
}

// validateMethods panics if any method in m is not a recognized HTTP verb.
func validateMethods(m []string) {
	for _, method := range m {
		if methodIndex(method) < 0 {
			goutil.Panicf("rux: invalid HTTP method %q", method)
		}
	}
}

// formatMethods uppercases and trims each method string.
func formatMethods(methods []string) []string {
	out := make([]string, 0, len(methods))
	for _, m := range methods {
		m = strings.TrimSpace(m)
		if m != "" {
			out = append(out, strings.ToUpper(m))
		}
	}
	return out
}

// simpleFmtPath does the minimal path normalization needed at Route construction.
// Full normalization happens in Router.appendRoute.
func simpleFmtPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	return path
}

// ToURL builds a request URL for this route using the optional build args.
// buildArgs may be:
//   - empty:                 return the path as-is.
//   - a single *BuildRequestURL: drive scheme/host/queries from the builder.
//   - a single M:            populate path params and query string.
//   - alternating k,v pairs: same as a single M built from the pairs.
//
// Path params are expressed using the v1 placeholder form (e.g. "{id}").
// originalPath is used so the {name} placeholders are still present even
// though the registered path has been rewritten to colon syntax.
func (r *Route) ToURL(buildArgs ...any) *url.URL {
	pathTpl := r.originalPath
	if pathTpl == "" {
		pathTpl = r.path
	}

	n := len(buildArgs)
	if n == 0 {
		return NewBuildRequestURL().Path(pathTpl).Build()
	}

	var URLBuilder *BuildRequestURL
	var withParams = make(M)

	if n == 1 {
		switch v := buildArgs[0].(type) {
		case *BuildRequestURL:
			URLBuilder = v
		case M:
			URLBuilder = NewBuildRequestURL()
			withParams = v
		default:
			panic("rux: BuildURL: unsupported single argument type")
		}
	} else {
		if n%2 == 1 {
			panic("rux: BuildURL: odd argument count for k,v pairs")
		}
		for i := 0; i < n; i += 2 {
			withParams[fmt.Sprint(buildArgs[i])] = buildArgs[i+1]
		}
		URLBuilder = NewBuildRequestURL()
	}

	return URLBuilder.Path(pathTpl).Build(withParams)
}
