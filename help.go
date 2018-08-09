// Package sux is a simple and fast request router for golang HTTP applications.
//
// Source code and other details for the project are available at GitHub:
// 		https://github.com/gookit/sux
//
// usage please ref examples and README
package sux

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
)

/*************************************************************
 * global path params
 *************************************************************/

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
 * help methods
 *************************************************************/

// String all routes to string
func (r *Router) String() string {
	buf := new(bytes.Buffer)

	fmt.Fprintf(buf, "Routes Count: %d\n", r.counter)

	fmt.Fprint(buf, "Stable(fixed):\n")
	for _, route := range r.stableRoutes {
		fmt.Fprintf(buf, " %s\n", route)
	}

	fmt.Fprint(buf, "Regular(dynamic):\n")
	for pfx, routes := range r.regularRoutes {
		fmt.Fprintf(buf, " %s:\n", pfx)
		for _, route := range routes {
			fmt.Fprintf(buf, "   %s\n", route.String())
		}
	}

	fmt.Fprint(buf, "Irregular(dynamic):\n")
	for m, routes := range r.irregularRoutes {
		fmt.Fprintf(buf, " %s:\n", m)
		for _, route := range routes {
			fmt.Fprintf(buf, "   %s\n", route.String())
		}
	}

	return buf.String()
}

func (r *Router) formatPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "/"
	}

	if path[0] != '/' {
		path = "/" + path
	}

	if !r.strictLastSlash {
		path = strings.TrimRight(path, "/")
	}

	return path
}

func (r *Router) isFixedPath(s string) bool {
	return strings.IndexByte(s, '{') < 0 && strings.IndexByte(s, '[') < 0
}

/*************************************************************
 * help functions
 *************************************************************/

func resolveAddress(addr []string) (fullAddr string) {
	ip := "0.0.0.0"
	switch len(addr) {
	case 0:
		if port := os.Getenv("PORT"); len(port) > 0 {
			debugPrint("Environment variable PORT=\"%s\"", port)
			return ip + ":" + port
		}
		debugPrint("Environment variable PORT is undefined. Using port :8080 by default")
		return ip + ":8080"
	case 1:
		var port string
		if strings.IndexByte(addr[0], ':') != -1 {
			ss := strings.SplitN(addr[0], ":", 2)
			if ss[0] != "" {
				return addr[0]
			}
			port = ss[1]
		} else {
			port = addr[0]
		}

		return ip + ":" + port
	default:
		panic("too much parameters")
	}
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

func quotePointChar(path string) string {
	if strings.IndexByte(path, '.') > 0 {
		// "about.html" -> "about\.html"
		return strings.Replace(path, ".", `\.`, -1)
	}

	return path
}

func nameOfFunction(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func debugPrintRoute(route *Route) {
	// if debug {
	// 	fmt.Println("[SUX-DEBUG]", route.String())
	// }
	debugPrint(route.String())
}

func debugPrintError(err error) {
	if err != nil {
		debugPrint("[ERROR] %v\n", err)
	}
}

func debugPrint(f string, v ...interface{}) {
	if debug {
		msg := fmt.Sprintf(f, v...)
		// fmt.Printf("[SUX-DEBUG] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
		fmt.Printf("[SUX-DEBUG] %s\n", msg)
	}
}
