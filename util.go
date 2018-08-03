package souter

import (
	"strings"
	"regexp"
)

/*************************************************************
 * route parse
 *************************************************************/

var allPattern = regexp.MustCompile(`.+`)
var anyPattern = regexp.MustCompile(`[^/]+`)
var varPattern = regexp.MustCompile(`:([a-zA-Z0-9]+)`)

// Parsing routes with parameters
func (r *Router) parseParamRoute(path string, route *Route) {
	argPos := strings.Index(path, "{")
	optPos := strings.Index(path, "[")

	minPos := argPos
	if argPos > optPos {
		minPos = optPos
	}

	start := path[0:minPos-1]
	if len(start) > 1 {
		route.start = start

		if pos := strings.Index(start[1:], "/");pos > 1 {
			route.first = start[1:pos-1]
		}
	}

	// has optional char. /blog[/:id]
	if optPos > 0 {
		noClosedOptional := strings.TrimRight(path, "]")
		optionalNum := len(path) - len(noClosedOptional)

		if optionalNum != strings.Count(noClosedOptional, "["){
			panic("Optional segments can only occur at the end of a route")
		}

		// '/hello[/:name]' -> '/hello(?:/:name)?'
		path = strings.NewReplacer("[", "(?:", "]", ")").Replace(path)
	}
}

/*************************************************************
 * helper methods
 *************************************************************/

func (r *Router) formatPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}

	if path[0] != '/' {
		path = "/" + path
	}

	if r.ignoreLastSlash {
		path = strings.TrimRight(path, "/")
	}

	return path
}

func (r *Router) buildRealPath(path string) string {
	if r.currentGroupPrefix != "" {
		return r.currentGroupPrefix + path
	}

	return path
}

func isFixedPath(path string) bool {
	return strings.Index(path, ":") < 0 && strings.Index(path, "[") < 0
}