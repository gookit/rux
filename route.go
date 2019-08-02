package rux

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

/*************************************************************
 * Route Params
 *************************************************************/

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

	// start string in the route path. "/users/{id}" -> "/user/"
	start string
	// path but no regex
	// "/users/{uid:\d+}/blog/{id}" -> "/users/{uid}/blog/{id}"
	spath string
	// hosts []string
	// regexp for the route path
	regex *regexp.Regexp
	// dynamic route param values, only use for route cache
	params Params
	// matched var names in the route path. eg "/api/{var1}/{var2}" -> [var1, var2]
	matches []string

	// the main handler for the route
	handler HandlerFunc
	// middleware handlers list for the route
	handlers HandlersChain

	// Opts some options data for the route
	Opts map[string]interface{}

	// defaults
}

// RouteInfo simple route info struct
type RouteInfo struct {
	Name, Path, HandlerName string
	// supported method of the route
	Methods []string
}

// NewRoute create a new route
func NewRoute(path string, handler HandlerFunc, methods ...string) *Route {
	return &Route{
		path: strings.TrimSpace(path),
		// handler
		handler: handler,
		methods: formatMethodsWithDefault(methods, GET),
		// handlers: middleware,
	}
}

// NewNamedRoute create a new route with name
func NewNamedRoute(name, path string, handler HandlerFunc, methods ...string) *Route {
	return &Route{
		name: strings.TrimSpace(name),
		path: strings.TrimSpace(path),
		// handler
		handler: handler,
		methods: formatMethodsWithDefault(methods, GET),
	}
}

// Use add middleware handlers to the route
func (r *Route) Use(middleware ...HandlerFunc) *Route {
	finalSize := len(r.handlers) + len(middleware)
	if finalSize >= int(abortIndex) {
		panicf("too many handlers(number: %d)", finalSize)
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
	r.SetName(name)
	if r.name != "" {
		router.namedRoutes[r.name] = r
	}
}

// SetName set a name for the route
func (r *Route) SetName(name string) *Route {
	r.name = strings.TrimSpace(name)
	return r
}

// setMethods set a name for the route
func (r *Route) setMethods(methods ...string) *Route {
	r.methods = formatMethods(methods)
	return r
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

// HandlerName get the main handler name
func (r *Route) HandlerName() string {
	return nameOfFunction(r.handler)
}

// String route info to string
func (r *Route) String() string {
	return fmt.Sprintf(
		"%-20s %-32s --> %s (%d middleware)",
		r.MethodString(","), r.path, r.HandlerName(), len(r.handlers),
	)
}

// Info get basic info of the route
func (r *Route) Info() RouteInfo {
	return RouteInfo{r.name, r.path, r.HandlerName(), r.methods}
}

// BuildRequestURL build RequestURL
func (r *Router) BuildRequestURL(name string, buildRequestURLs ...*BuildRequestURL) *url.URL {
	var buildRequestURL *BuildRequestURL

	path := r.GetRoute(name).path

	if len(buildRequestURLs) == 0 {
		return NewBuildRequestURL().Path(path).Build()
	}

	buildRequestURL = buildRequestURLs[0]
	ss := varRegex.FindAllString(path, -1)

	if len(ss) == 0 {
		return nil
	}

	var n string
	var varParams = make(map[string]string)

	for _, str := range ss {
		nvStr := str[1 : len(str)-1]

		if strings.IndexByte(nvStr, ':') > 0 {
			nv := strings.SplitN(nvStr, ":", 2)
			n, _ = strings.TrimSpace(nv[0]), strings.TrimSpace(nv[1])
			varParams[str] = "{" + n + "}"
		} else {
			varParams[str] = str
		}
	}

	for paramRegex, name := range varParams {
		path = strings.NewReplacer(paramRegex, name).Replace(path)
	}

	return buildRequestURL.Path(path).Build()
}

// check route info
func (r *Route) goodInfo() {
	if r.handler == nil {
		panicf("the route handler cannot be empty.(path: '%s')", r.path)
	}

	if len(r.methods) == 0 {
		panicf("the route allowed methods cannot be empty.(path: '%s')", r.path)
	}

	for _, method := range r.methods {
		if strings.Index(","+StringMethods, ","+method) == -1 {
			panicf("invalid method name '%s', must in: %s", method, StringMethods)
		}
	}
}

// check custom var regex string.
// ERROR:
// 	"{id:(\d+)}" -> "(\d+)"
//
// RIGHT:
// 	"{id:\d+}"
// 	"{id:(?:\d+)}"
func (r *Route) goodRegexString(n, v string) {
	pos := strings.IndexByte(v, '(')

	if pos != -1 && pos < len(v) && v[pos+1] != '?' {
		panicf("invalid path var regex string, dont allow char '('. var: %s, regex: %s", n, v)
	}
}

// check start string and match a regex route
func (r *Route) match(path string) (ps Params, ok bool) {
	// check start string
	if r.start != "" && strings.Index(path, r.start) != 0 {
		return
	}
	return r.matchRegex(path)
}

// match a regex route
func (r *Route) matchRegex(path string) (ps Params, ok bool) {
	// regex match
	ss := r.regex.FindAllStringSubmatch(path, -1)
	if len(ss) == 0 {
		return
	}

	ok = true
	vs := ss[0]
	ps = make(Params, len(vs))

	// Notice: vs[0] is full path.
	for i, val := range vs[1:] {
		n := r.matches[i]
		ps[n] = val
	}
	return
}

// copy a new instance
func (r *Route) copyWithParams(ps Params) *Route {
	var nr = *r
	nr.regex = nil
	nr.matches = nil
	nr.params = ps

	return &nr
}
