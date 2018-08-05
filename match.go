package sux

import (
	"strings"
)

// MatchResult for the route match
type MatchResult struct {
	// match status: 1 found 2 not found 3 method not allowed
	Status uint8
	// route params, when Status = 1
	Params Params
	// route handlers, when Status = 1
	Handlers HandlersChain
	// allowed methods, when Status = 3
	AllowedMethods []string
}

var notFoundResult = &MatchResult{Status: NotFound}

func newMatchResult(status uint8, handlers HandlersChain) *MatchResult {
	return &MatchResult{Status: status, Handlers: handlers}
}

func (mr *MatchResult) JoinAllowedMethods(sep string) string {
	return strings.Join(mr.AllowedMethods, sep)
}

/*************************************************************
 * route match
 *************************************************************/

// Match route by given request METHOD and URI path
func (r *Router) Match(method, path string) (result *MatchResult) {
	path = r.formatPath(path)
	method = strings.ToUpper(method)

	// do match
	result = r.match(method, path)
	if result.Status == Found {
		return
	}

	// for HEAD requests, attempt fallback to GET
	if method == HEAD {
		result = r.match(method, path)
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
		result = &MatchResult{Status:NotAllowed, AllowedMethods: allowed}
	}

	return
}

func (r *Router) match(method, path string) (ret *MatchResult) {
	// find in stable routes
	key := method + " " + path
	if route, ok := r.stableRoutes[key]; ok {
		return newMatchResult(Found, route.handlers)
	}

	// find in cached routes
	if route, ok := r.cachedRoutes[key]; ok {
		ret = newMatchResult(Found, route.handlers)
		ret.Params = route.Params
		return
	}

	// find in regular routes
	if pos := strings.Index(path[1:], "/"); pos > 1 {
		first := path[1 : pos-1]
		key = method + " " + first

		if rs, ok := r.regularRoutes[key]; ok {
			for _, route := range rs {
				if ps, ok := route.match(path); ok {
					ret = newMatchResult(Found, route.handlers)
					ret.Params = ps
					return
				}
			}
		}
	}

	// find in irregular routes
	if rs, ok := r.irregularRoutes[method]; ok {
		for _, route := range rs {
			if ps, ok := route.match(path); ok {
				ret = newMatchResult(Found, route.handlers)
				ret.Params = ps
				r.cacheDynamicRoute(path, ps, route)
				return
			}
		}
	}

	return notFoundResult
}

func (r *Router) cacheDynamicRoute(method string, ps Params, route *Route) {
	if !r.enableRouteCache  {
		return
	}

	if r.cachedRoutes == nil {
		r.cachedRoutes = make(map[string]*Route, r.maxCachedRoute)
	} else if len(r.cachedRoutes) >= int(r.maxCachedRoute) {
		num := 0
		maxClean := int(r.maxCachedRoute/10)

		// clean up 1/10 each time
		for k, _ := range r.cachedRoutes {
			if num == maxClean {
				break
			}

			num++
			r.cachedRoutes[k] = nil
			delete(r.cachedRoutes, k)
		}
	}

	key := method + " " + route.pattern

	// copy new route instance
	nr := route.Copy()
	nr.vars = nil
	nr.regex = nil
	nr.matches = nil
	nr.Params = ps // Notice: cache matched params

	r.cachedRoutes[key] = nr
}

func (r *Router) findAllowedMethods(method, path string) (allowed []string) {
	// in stable routes
	for _, m := range anyMethods {
		if m == method {
			continue
		}

		key := m + " " + path
		if _, ok := r.stableRoutes[key]; ok {
			allowed = append(allowed, m)
		}
	}

	// in regular routes
	if pos := strings.Index(path[1:], "/"); pos > 1 {
		for _, m := range anyMethods {
			if m == method {
				continue
			}

			first := path[1 : pos-1]
			key := m + " " + first

			if rs, ok := r.regularRoutes[key]; ok {
				for _, route := range rs {
					if _, ok := route.match(path); ok {
						allowed = append(allowed, m)
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
					allowed = append(allowed, m)
				}
			}
		}
	}

	return
}
