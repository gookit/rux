package rux

import (
	"fmt"
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
	path   string
	method string

	// start string in the route path. "/users/{id}" -> "/user/"
	start string
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

// RouteInfo struct
type RouteInfo struct {
	Path, Method, HandlerName string
}

// NewRoute create a new route
func NewRoute(method, path string, handler HandlerFunc, handlers HandlersChain) *Route {
	return &Route{
		path:   strings.TrimSpace(path),
		method: strings.ToUpper(method),
		// handler
		handler:  handler,
		handlers: handlers,
	}
}

// Use add middleware handlers to the route
func (r *Route) Use(middleware ...HandlerFunc) *Route {
	r.handlers = append(r.handlers, middleware...)
	return r
}

// AttachTo register the route to router.
func (r *Route) AttachTo(router *Router) {
	router.AddRoute(r)
}

// NamedTo register the route to router.
func (r *Route) NamedTo(name string, router *Router) *Route {
	r.SetName(name)

	if r.name != "" {
		router.namedRoutes[r.name] = r
	}

	return r
}

// SetName set a name for the route
func (r *Route) SetName(name string) *Route {
	r.name = strings.TrimSpace(name)
	return r
}

// Name get
func (r *Route) Name() string {
	return r.name
}

// Path get route path string.
func (r *Route) Path() string {
	return r.path
}

// Method get route request method string.
func (r *Route) Method() string {
	return r.method
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
		"%-7s %-25s --> %s (%d middleware)",
		r.method, r.path, r.HandlerName(), len(r.handlers),
	)
}

// Info get basic info of the route
func (r *Route) Info() RouteInfo {
	return RouteInfo{r.path, r.method, r.HandlerName()}
}

// check route info
func (r *Route) goodInfo() {
	if r.handler == nil {
		panicf("the route handler cannot be empty.(path: '%s')", r.path)
	}

	if strings.Index(","+StringMethods, ","+r.method) == -1 {
		panicf("invalid method name '%s', must in: %s", r.method, StringMethods)
	}
}

// match a regex route
func (r *Route) match(path string) (ps Params, ok bool) {
	// check start string
	if r.start != "" && strings.Index(path, r.start) != 0 {
		return
	}

	// regex match
	ss := r.regex.FindAllStringSubmatch(path, -1)
	if len(ss) == 0 {
		return
	}

	ok = true
	ps = make(Params, len(ss))
	for i, item := range ss {
		if len(item) > 1 {
			n := r.matches[i]
			ps[n] = item[1]
		}
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
