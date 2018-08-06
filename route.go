package sux

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

/*************************************************************
 * Route params
 *************************************************************/

// Params for current route
type Params map[string]string

// Has param key in the params
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
	// name   string
	// path for the route. eg "/users" "/users/{id}"
	path   string
	method string

	// start string in the route path. "/users/{id}" -> "/user/"
	start string
	hosts []string
	// regexp for the route path
	regex *regexp.Regexp
	// matched var names in the route path. eg "/api/{var1}/{var2}" -> [var1, var2]
	matches []string
	// handlers list for the route
	handlers HandlersChain
	// dynamic route param values, only use for route cache
	params Params
	// var define for the route. eg ["name": `\w+`]
	vars map[string]string

	// some options data for the route
	Opts map[string]interface{}

	// defaults
}

func newRoute(method, path string, handlers HandlersChain) *Route {
	return &Route{
		method:   method,
		path:     path,
		handlers: handlers,
	}
}

// Use some middleware handlers
func (r *Route) Use(handlers ...HandlerFunc) *Route {
	r.handlers = append(r.handlers, handlers...)
	return r
}

// SetVar add var regex for the route path
func (r *Route) SetVar(name, regex string) *Route {
	r.vars[name] = regex
	return r
}

// SetVars add vars path for the route path
func (r *Route) SetVars(vars map[string]string) *Route {
	for name, regex := range vars {
		r.vars[name] = regex
	}

	return r
}

// String route info to string
func (r *Route) String() string {
	nuHandlers := len(r.handlers)
	handlerName := nameOfFunction(r.handlers.Last())

	return fmt.Sprintf(
		"%-6s %-25s --> %s (%d handlers)",
		r.method, r.path, handlerName, nuHandlers,
	)
}

func (r *Route) getVar(name, def string) string {
	if val, ok := r.vars[name]; ok {
		return val
	}

	if val, ok := globalVars[name]; ok {
		return val
	}

	return def
}

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
	nr.vars = nil
	nr.regex = nil
	nr.matches = nil
	nr.params = ps

	return &nr
}
