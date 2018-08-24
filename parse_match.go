package sux

import (
	"regexp"
	"strings"
)

/*************************************************************
 * route parse
 *************************************************************/

const (
	anyMatch = `[^/]+`
)

// "/users/{id}" "/users/{id:\d+}" `/users/{uid:\d+}/blog/{id}`
var varRegex = regexp.MustCompile(`{([^/]+)}`)

// Parsing routes with parameters
func (r *Router) parseParamRoute(path string, route *Route) (first string) {
	// collect route Params
	ss := varRegex.FindAllString(path, -1)

	// no vars, but contains optional char
	if len(ss) == 0 {
		regexStr := checkAndParseOptional(quotePointChar(path))
		route.regex = regexp.MustCompile("^" + regexStr + "$")
		return
	}

	var n, v string
	var rawVar, varRegex []string
	for _, str := range ss {
		nvStr := str[1:len(str)-1] // "{level:[1-9]{1,2}}" -> "level:[1-9]{1,2}"

		// eg "{uid:\d+}" -> "uid", "\d+"
		if strings.IndexByte(nvStr, ':') > 0 {
			nv := strings.SplitN(nvStr, ":", 2)
			n, v = strings.TrimSpace(nv[0]), strings.TrimSpace(nv[1])
			rawVar = append(rawVar, str, "{"+n+"}")
			varRegex = append(varRegex, "{"+n+"}", "("+v+")")
		} else {
			n = nvStr // "{name}" -> "name"
			v = getGlobalVar(n, anyMatch)
			varRegex = append(varRegex, str, "("+v+")")
		}

		route.matches = append(route.matches, n)
	}

	// `/users/{uid:\d+}/blog/{id}` -> `/users/{uid}/blog/{id}`
	if len(rawVar) > 0 {
		path = strings.NewReplacer(rawVar...).Replace(path)
	}

	// "." -> "\."
	path = quotePointChar(path)
	argPos := strings.IndexByte(path, '{')
	optPos := strings.IndexByte(path, '[')
	minPos := argPos

	// has optional char. /blog[/{id}]
	if optPos > 0 && argPos > optPos {
		minPos = optPos
	}

	start := path[0:minPos]
	if len(start) > 1 {
		route.start = start

		if pos := strings.IndexByte(start[1:], '/'); pos > 1 {
			first = start[1 : pos+1]
			// start string only one node. "/users/"
			if len(start)-len(first) == 2 {
				route.start = ""
			}
		}
	}

	// has optional char. /blog[/{id}]  -> /blog(?:/{id})
	if optPos > 0 {
		path = checkAndParseOptional(path)
	}

	// replace {var} -> regex str
	regexStr := strings.NewReplacer(varRegex...).Replace(path)
	route.regex = regexp.MustCompile("^" + regexStr + "$")
	return
}

/*************************************************************
 * route match
 *************************************************************/

// MatchResult for the route match
type MatchResult struct {
	// Status match status: 1 found 2 not found 3 method not allowed
	Status uint8
	// Params route path Params, when Status = 1 and has path vars.
	Params Params
	// Handler the main handler for the route(Status = 1)
	Handler HandlerFunc
	// Handlers middleware handlers for the route(Status = 1)
	Handlers HandlersChain
	// AllowedMethods allowed request methods(Status = 3)
	AllowedMethods []string
}

var notFoundResult = &MatchResult{Status: NotFound}

func newMatchResult(status uint8, h HandlerFunc, hs HandlersChain) *MatchResult {
	return &MatchResult{Status: status, Handler: h, Handlers: hs}
}

// Match route by given request METHOD and URI path
func (r *Router) Match(method, path string) (result *MatchResult) {
	if r.interceptAll != "" {
		path = r.interceptAll
	}

	path = r.formatPath(path)
	method = strings.ToUpper(method)

	// do match
	result = r.match(method, path)
	if result.Status == Found {
		return
	}

	// for HEAD requests, attempt fallback to GET
	if method == HEAD {
		result = r.match(GET, path)
		if result.Status == Found {
			return
		}
	}

	// don't handle method not allowed, will return not found
	if !r.handleMethodNotAllowed {
		return
	}

	// find allowed methods
	allowed := r.findAllowedMethods(method, path)
	if len(allowed) > 0 {
		result = &MatchResult{Status: NotAllowed, AllowedMethods: allowed}
	}

	return
}

func (r *Router) match(method, path string) (ret *MatchResult) {
	// find in stable routes
	key := method + " " + path
	if route, ok := r.stableRoutes[key]; ok {
		return newMatchResult(Found, route.handler, route.handlers)
	}

	// find in cached routes
	if route, ok := r.cachedRoutes[key]; ok {
		ret = newMatchResult(Found, route.handler, route.handlers)
		ret.Params = route.params
		return
	}

	// find in regular routes
	if pos := strings.IndexByte(path[1:], '/'); pos > 1 {
		first := path[1 : pos+1]
		key = method + " " + first

		if rs, ok := r.regularRoutes[key]; ok {
			for _, route := range rs {
				if ps, ok := route.match(path); ok {
					ret = newMatchResult(Found, route.handler, route.handlers)
					ret.Params = ps
					r.cacheDynamicRoute(path, ps, route)
					return
				}
			}
		}
	}

	// find in irregular routes
	if rs, ok := r.irregularRoutes[method]; ok {
		for _, route := range rs {
			if ps, ok := route.match(path); ok {
				ret = newMatchResult(Found, route.handler, route.handlers)
				ret.Params = ps
				r.cacheDynamicRoute(path, ps, route)
				return
			}
		}
	}

	return notFoundResult
}

// cache dynamic Params route when EnableRouteCache is true
func (r *Router) cacheDynamicRoute(path string, ps Params, route *Route) {
	if !r.enableCaching {
		return
	}

	if r.cachedRoutes == nil {
		r.cachedRoutes = make(map[string]*Route, r.maxNumCaches)
	} else if len(r.cachedRoutes) >= int(r.maxNumCaches) {
		num := 0
		maxClean := int(r.maxNumCaches / 10)

		// clean up 1/10 each time
		for k := range r.cachedRoutes {
			if num == maxClean {
				break
			}

			num++
			r.cachedRoutes[k] = nil
			delete(r.cachedRoutes, k)
		}
	}

	key := route.method + " " + path
	// copy new route instance. Notice: cache matched Params
	r.cachedRoutes[key] = route.copyWithParams(ps)
}

// find allowed methods for current request
func (r *Router) findAllowedMethods(method, path string) (allowed []string) {
	// use map for prevent duplication
	mMap := map[string]int{}

	// in stable routes
	for _, m := range anyMethods {
		if m == method {
			continue
		}

		key := m + " " + path
		if _, ok := r.stableRoutes[key]; ok {
			mMap[m] = 1
		}
	}

	// in regular routes
	if pos := strings.IndexByte(path[1:], '/'); pos > 1 {
		first := path[1 : pos+1]
		for _, m := range anyMethods {
			if m == method {
				continue
			}

			key := m + " " + first
			if rs, ok := r.regularRoutes[key]; ok {
				for _, route := range rs {
					if _, ok := route.match(path); ok {
						mMap[m] = 1
					}
				}
			}
		}
	}

	// in irregular routes
	for _, m := range anyMethods {
		if m == method {
			continue
		}

		if rs, ok := r.irregularRoutes[m]; ok {
			for _, route := range rs {
				if _, ok := route.match(path); ok {
					mMap[m] = 1
				}
			}
		}
	}

	if len(mMap) > 0 {
		for m := range mMap {
			allowed = append(allowed, m)
		}
	}

	return
}
