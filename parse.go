package sux

import (
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

var globalVars = map[string]string{
	"all": `.*`,
	"any": `[^/]+`,
	"num": `[1-9][0-9]*`,
}

// SetGlobalVar set an global path var
func SetGlobalVar(name, regex string) {
	globalVars[name] = regex
}

// GetGlobalVars get all global path vars
func GetGlobalVars() map[string]string {
	return globalVars
}

func getGlobalVar(name, def string) string {
	if val, ok := globalVars[name]; ok {
		return val
	}

	return def
}

/*************************************************************
 * route parse
 *************************************************************/

const (
	allMatch = `.+`
	anyMatch = `[^/]+`
)

// "/users/{id}" "/users/{id:\d+}" `/users/{uid:\d+}/blog/{id}`
var varRegex = regexp.MustCompile(`{([^/]+)}`)

// Parsing routes with parameters
func (r *Router) parseParamRoute(path string, route *Route) (first string) {
	// collect route params
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
		nvStr := strings.Trim(str, "{}: ")

		// eg "{uid:\d+}" -> "uid", "\d+"
		if strings.Index(nvStr, ":") > 0 {
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
	argPos := strings.Index(path, "{")
	optPos := strings.Index(path, "[")
	minPos := argPos

	// has optional char. /blog[/{id}]
	if optPos > 0 && argPos > optPos {
		minPos = optPos
	}

	start := path[0:minPos]
	if len(start) > 1 {
		route.start = start

		if pos := strings.Index(start[1:], "/"); pos > 1 {
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

func checkAndParseOptional(path string) string {
	noClosedOptional := strings.TrimRight(path, "]")
	optionalNum := len(path) - len(noClosedOptional)

	if optionalNum != strings.Count(noClosedOptional, "[") {
		panic("Optional segments can only occur at the end of a route")
	}

	// '/hello[/{name}]' -> '/hello(?:/{name})?'
	return strings.NewReplacer("[", "(?:", "]", ")?").Replace(path)
}

/*************************************************************
 * helper methods
 *************************************************************/

func (r *Router) formatPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "/"
	}

	if path[0] != '/' {
		path = "/" + path
	}

	if r.IgnoreLastSlash {
		path = strings.TrimRight(path, "/")
	}

	return path
}

func (r *Router) isFixedPath(path string) bool {
	return strings.Index(path, "{") < 0 && strings.Index(path, "[") < 0
}

func quotePointChar(path string) string {
	if strings.Index(path, ".") > 0 {
		// "about.html" -> "about\.html"
		return strings.Replace(path, ".", `\.`, -1)
	}

	return path
}

func nameOfFunction(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}
