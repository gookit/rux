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

// params for current route
type Params map[string]string

func (p Params) String(key string) (val string) {
	if val, ok := p[key]; ok {
		return val
	}

	return
}

func (p Params) Int(key string) (val int) {
	if str, ok := p[key]; ok {
		val, err := strconv.Atoi(str)
		if err != nil {
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
	method string

	// route params, only use for route cache
	Params Params

	// path/pattern definition for the route. eg "/users" "/users/{id}"
	pattern string

	// start string in the route pattern. "/users/{id}" -> "/user/"
	start string

	// regexp for the route pattern
	regex *regexp.Regexp

	// handlers list for the route
	handlers HandlersChain

	// some options data for the route
	Opts map[string]interface{}

	vars map[string]string

	hosts []string
	// var names in the route path. /api/{var1}/{var2} -> [var1, var2]
	matches []string

	// domains
	// defaults
}

func newRoute(method, path string, handlers HandlersChain) *Route {
	return &Route{
		method:   method,
		pattern:  path,
		handlers: handlers,
	}
}

// Use some middleware handlers
func (r *Route) Use(handlers ...HandlerFunc) *Route {
	r.handlers = append(r.handlers, handlers...)
	return r
}

// Vars add vars pattern for the route path
func (r *Route) SetVars(vars map[string]string) *Route {
	for name, pattern := range vars {
		r.vars[name] = pattern
	}

	return r
}

func (r *Route) String() string {
	nuHandlers := len(r.handlers)
	handlerName := nameOfFunction(r.handlers.Last())

	return fmt.Sprintf(
		"%-6s %-25s --> %s (%d handlers)",
		r.method, r.pattern, handlerName, nuHandlers,
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
		n := r.matches[i]
		ps[n] = item[1]
	}

	return
}

func (r *Route) withParams(ps Params) *Route {
	r.Params = ps
	return r
}

func (r *Route) Copy() *Route {
	var route = *r

	return &route
}
