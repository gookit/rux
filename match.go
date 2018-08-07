package sux

import (
	"strings"
)

// MatchResult for the route match
type MatchResult struct {
	// Status match status: 1 found 2 not found 3 method not allowed
	Status uint8
	// Params route path params, when Status = 1 and has path vars.
	Params Params
	// Handler the main handler for the route(Status = 1)
	Handler HandlerFunc
	// Handlers middleware handlers for the route(Status = 1)
	Handlers HandlersChain
	// AllowedMethods allowed request methods(Status = 3)
	AllowedMethods []string
}

var notFoundResult = &MatchResult{Status: NotFound}

func newMatchResult(status uint8, handler HandlerFunc, handlers HandlersChain) *MatchResult {
	return &MatchResult{Status: status, Handler: handler, Handlers: handlers}
}

// JoinAllowedMethods join allowed methods to string
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
		result = r.match(GET, path)
		if result.Status == Found {
			return
		}
	}

	// don't handle method not allowed, will return not found
	if !r.HandleMethodNotAllowed {
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
	if pos := strings.Index(path[1:], "/"); pos > 1 {
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

// cache dynamic params route when EnableRouteCache is true
func (r *Router) cacheDynamicRoute(path string, ps Params, route *Route) {
	if !r.EnableRouteCache {
		return
	}

	if r.cachedRoutes == nil {
		r.cachedRoutes = make(map[string]*Route, r.MaxCachedRoute)
	} else if len(r.cachedRoutes) >= int(r.MaxCachedRoute) {
		num := 0
		maxClean := int(r.MaxCachedRoute / 10)

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

	// copy new route instance. Notice: cache matched params
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
	if pos := strings.Index(path[1:], "/"); pos > 1 {
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
