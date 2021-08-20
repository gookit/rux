package rux

import (
	"regexp"
	"strings"
)

/*************************************************************
 * route parse
 *************************************************************/

// Parsing routes with parameters
func (r *Router) parseParamRoute(route *Route) (first string) {
	path := route.path
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
		nvStr := str[1 : len(str)-1] // "{level:[1-9]{1,2}}" -> "level:[1-9]{1,2}"

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

		route.goodRegexString(n, v)
		route.matches = append(route.matches, n)
	}

	// `/users/{uid:\d+}/blog/{id}` -> `/users/{uid}/blog/{id}`
	if len(rawVar) > 0 {
		path = strings.NewReplacer(rawVar...).Replace(path)
		// save simple path
		route.spath = path
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

		if pos := strings.IndexByte(start[1:], '/'); pos > 0 {
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

// Match route by given request METHOD and URI path
//
// ps  - route path Params, when has path vars.
// alm - allowed request methods
func (r *Router) Match(method, path string) (route *Route, ps Params, alm []string) {
	return r.QuickMatch(strings.ToUpper(method), path)
}

// QuickMatch match route by given request METHOD and URI path
// ps  - route path Params, when has path vars.
// alm - allowed request methods
func (r *Router) QuickMatch(method, path string) (route *Route, ps Params, alm []string) {
	if r.interceptAll != "" {
		path = r.interceptAll
	} else {
		path = r.formatPath(path)
	}

	// do match route
	if route, ps = r.match(method, path); route != nil {
		return
	}

	// for HEAD requests, attempt fallback to GET
	if method == HEAD {
		route, ps = r.match(GET, path)
		if route != nil {
			return
		}
	}

	// handle fallback route. add by: router->Any("/*", handler)
	if r.handleFallbackRoute {
		key := method + "/*"
		if route, ok := r.stableRoutes[key]; ok {
			return route, nil, nil
		}
	}

	// handle method not allowed. will find allowed methods
	if r.handleMethodNotAllowed {
		alm = r.findAllowedMethods(method, path)
		if len(alm) > 0 {
			return
		}
	}

	// don't handle method not allowed, will return not found
	return
}

// func (r *Router) Search(method, path string) *Route {
// }

func (r *Router) match(method, path string) (rt *Route, ps Params) {
	// find in stable routes
	key := method + path
	if route, ok := r.stableRoutes[key]; ok {
		// return r.newMatchResult(route, nil)
		return route, nil
	}

	// find in cached routes
	if r.enableCaching {
		route, ok := r.cachedRoutes.Get(key)
		if ok {
			return route, route.params
		}
	}

	// find in regular routes
	if pos := strings.IndexByte(path[1:], '/'); pos > 0 {
		key = method + path[1:pos+1]

		if rs, ok := r.regularRoutes[key]; ok {
			for _, route := range rs {
				if strings.Index(path, route.start) != 0 {
					continue
				}

				if ps, ok := route.matchRegex(path); ok {
					// ret = r.newMatchResult(route, ps)
					r.cacheDynamicRoute(key, ps, route)
					return route, ps
				}
			}
		}
	}

	// find in irregular routes
	if rs, ok := r.irregularRoutes[method]; ok {
		for _, route := range rs {
			if ps, ok := route.matchRegex(path); ok {
				r.cacheDynamicRoute(key, ps, route)
				return route, ps
			}
		}
	}
	return
}

// cache dynamic Params route when EnableRouteCache is true
func (r *Router) cacheDynamicRoute(key string, ps Params, route *Route) {
	if !r.enableCaching {
		return
	}

	// copy new route instance. Notice: cache matched Params
	r.cachedRoutes.Set(key, route.copyWithParams(ps))
}

// find allowed methods for current request
func (r *Router) findAllowedMethods(method, path string) (allowed []string) {
	// use map for prevent duplication
	mMap := map[string]int{}
	for _, m := range anyMethods {
		if m == method { // expected current method
			continue
		}

		if rt, _ := r.match(m, path); rt != nil {
			mMap[m] = 1
		}
	}

	if len(mMap) > 0 {
		for m := range mMap {
			allowed = append(allowed, m)
		}
	}
	return
}
