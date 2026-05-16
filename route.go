package rux

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gookit/goutil"
)

/*************************************************************
 * Route Params/Info
 *************************************************************/

// RouteInfo simple route info struct
type RouteInfo struct {
	Name, Path, HandlerName string
	// supported method of the route
	Methods    []string
	HandlerNum int
}

// Params for current route
type Params map[string]string

// Has param key in the Params
func (p Params) Has(key string) bool {
	_, ok := p[key]
	return ok
}

// String get string value by key
func (p Params) String(key string) (val string) {
	if val, ok := p[key]; ok {
		return val
	}
	return
}

// Int get int value by key
func (p Params) Int(key string) (val int) {
	if str, ok := p[key]; ok {
		val, err := strconv.Atoi(str)
		if err == nil {
			return val
		}
	}
	return
}

/*************************************************************
 * MatchResult - return type for Match/QuickMatch
 *************************************************************/

// MatchResult holds the result of a route match
type MatchResult struct {
	// Path matched
	Path string
	// Method matched
	Method string
	// Route matched
	Route *Route
	// Params extracted from the path
	Params Params
	// Allowed methods when method not allowed
	Allowed []string
}

/*************************************************************
 * Route definition
 *************************************************************/

// Route in the router
type Route struct {
	// route name.
	name string
	// path for the route. eg "/users" "/users/{id}"
	path string
	// allowed methods
	methods []string

	// the main handler for the route
	handler HandlerFunc
	// middleware handlers list for the route
	handlers HandlersChain

	// Opts some options data for the route
	Opts map[string]any
}

// NewRoute create a new route
func NewRoute(path string, handler HandlerFunc, methods ...string) *Route {
	return &Route{
		path: simpleFmtPath(path),
		// handler
		handler: handler,
		methods: formatMethodsWithDefault(methods, GET),
		// handlers: middleware,
	}
}

// NamedRoute create a new route with name. alias of NewNamedRoute()
func NamedRoute(name, path string, handler HandlerFunc, methods ...string) *Route {
	return NewNamedRoute(name, path, handler, methods...)
}

// NewNamedRoute create a new route with name
func NewNamedRoute(name, path string, handler HandlerFunc, methods ...string) *Route {
	return &Route{
		name: strings.TrimSpace(name),
		path: simpleFmtPath(path),
		// handler
		handler: handler,
		methods: formatMethodsWithDefault(methods, GET),
	}
}

// Use add middleware handlers to the route
func (r *Route) Use(middleware ...HandlerFunc) *Route {
	finalSize := len(r.handlers) + len(middleware)
	if finalSize >= int(abortIndex) {
		goutil.Panicf("too many handlers(number: %d)", finalSize)
	}

	r.handlers = append(r.handlers, middleware...)
	return r
}

// AttachTo register the route to router.
func (r *Route) AttachTo(router *Router) {
	router.AddRoute(r)
}

// NamedTo add name and register the route to router.
func (r *Route) NamedTo(name string, router *Router) {
	if name = strings.TrimSpace(name); name != "" {
		r.name = name
		// attach to router
		router.namedRoutes[name] = r
	}
}

// Name get route name
func (r *Route) Name() string {
	return r.name
}

// Path get route path string.
func (r *Route) Path() string {
	return r.path
}

// Methods get route allowed request methods
func (r *Route) Methods() []string {
	return r.methods
}

// MethodString join allowed methods to an string
func (r *Route) MethodString(char string) string {
	return strings.Join(r.methods, char)
}

// Handler returns the main handler.
func (r *Route) Handler() HandlerFunc {
	return r.handler
}

// Handlers returns handlers of the route.
func (r *Route) Handlers() HandlersChain {
	return r.handlers
}

// HandlerName get the main handler name
func (r *Route) HandlerName() string {
	return goutil.FuncName(r.handler)
}

// String route info to string
func (r *Route) String() string {
	method := r.MethodString(",")
	template := "%-15s %-38s --> %s (%d middleware)"

	// will print two line
	if len(method) > 14 {
		method = method + "\n" + strings.Repeat(" ", 27)
		template = "%s %-38s --> %s (%d middleware)"
	}

	return fmt.Sprintf(template, method, r.path, r.HandlerName(), len(r.handlers))
}

// Info get basic info of the route
func (r *Route) Info() RouteInfo {
	return RouteInfo{r.name, r.path, r.HandlerName(), r.methods, len(r.handlers)}
}

// ToURL build request URL, can with path vars
func (r *Route) ToURL(buildArgs ...any) *url.URL {
	var URLBuilder *BuildRequestURL
	//noinspection GoNilness
	path := r.path
	vlen := len(buildArgs)

	if vlen == 0 {
		return NewBuildRequestURL().Path(path).Build()
	}

	var withParams = make(M)
	if vlen == 1 {
		switch buildArgs[0].(type) {
		case *BuildRequestURL:
			URLBuilder = buildArgs[0].(*BuildRequestURL)
		case M:
			URLBuilder = NewBuildRequestURL()
			withParams = buildArgs[0].(M)
		default:
			panic("buildArgs odd argument count")
		}
	} else { // vlen > 1
		if vlen%2 == 1 {
			panic("buildArgs odd argument count")
		}

		for i := 0; i < len(buildArgs); i += 2 {
			withParams[goutil.String(buildArgs[i])] = buildArgs[i+1]
		}

		URLBuilder = NewBuildRequestURL()
	}

	return URLBuilder.Path(path).Build(withParams)
}

// BuildRequestURL alias of the method BuildRequestURL()
func (r *Router) BuildRequestURL(name string, buildArgs ...any) *url.URL {
	return r.BuildURL(name, buildArgs...)
}

// BuildURL build Request URL one arg can be set buildRequestURL or rux.M
func (r *Router) BuildURL(name string, buildArgs ...any) *url.URL {
	route := r.GetRoute(name)
	if route == nil {
		goutil.Panicf("BuildRequestURL get route is nil(name: %s)", name)
	}

	//noinspection GoNilness
	return route.ToURL(buildArgs...)
}

// check route info
func (r *Route) goodInfo() {
	if r.handler == nil {
		goutil.Panicf("the route handler cannot be empty.(path: '%s')", r.path)
	}

	if len(r.methods) == 0 {
		goutil.Panicf("the route allowed methods cannot be empty.(path: '%s')", r.path)
	}

	str := MethodsString()
	for _, method := range r.methods {
		if strings.Index(","+str, ","+method) == -1 {
			goutil.Panicf("invalid method name '%s', must in: %s", method, str)
		}
	}
}
