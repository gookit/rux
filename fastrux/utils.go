package fastrux

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gookit/color"
	"github.com/gookit/goutil"
)

// normalizePath standardizes a path
func normalizePath(path string) string {
	if len(path) == 0 {
		return "/"
	}

	path = strings.TrimSpace(path)

	// ensure starts with '/'
	if path[0] != '/' {
		path = "/" + path
	}

	// remove trailing '/' (except root path)
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	// compress consecutive slashes
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	return path
}

// normalizePathStrict standardizes path but preserves trailing slash (for strictLastSlash mode)
func normalizePathStrict(path string) string {
	if len(path) == 0 {
		return "/"
	}

	path = strings.TrimSpace(path)

	// ensure starts with '/'
	if path[0] != '/' {
		path = "/" + path
	}

	// compress consecutive slashes
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	return path
}

// longestCommonPrefix finds the longest common prefix length
func longestCommonPrefix(a, b string) int {
	maxLen := len(a)
	if len(b) < maxLen {
		maxLen = len(b)
	}

	i := 0
	for i < maxLen && a[i] == b[i] {
		i++
	}

	return i
}

// validateOptionalSegments validates optional parameter rules:
// 1. Optional segments can only appear at the end of the path
// 2. Only one optional segment is supported
func validateOptionalSegments(path string) {
	firstOptionalPos := strings.IndexByte(path, '[')
	lastOptionalPos := strings.LastIndexByte(path, '[')

	// no optional segments, return directly
	if firstOptionalPos == -1 {
		return
	}

	// rule 1: only one optional segment allowed
	if firstOptionalPos != lastOptionalPos {
		panic(fmt.Sprintf("route %s: only one optional segment is allowed", path))
	}

	// rule 2: nothing after optional segment
	closingBracketPos := strings.IndexByte(path, ']')
	afterOptionalPos := closingBracketPos + 1
	if afterOptionalPos < len(path) {
		panic(fmt.Sprintf("route %s: optional segment must be at the end of the path, found '%s' after ']'",
			path, path[afterOptionalPos:]))
	}
}

// parseOptionalSegments parses optional segments and expands into multiple routes
// Input: /posts[/{id}]
// Output: ["/posts", "/posts/:id"]
func parseOptionalSegments(path string) []string {
	// find optional segment start and end positions
	startIdx := strings.IndexByte(path, '[')
	endIdx := strings.IndexByte(path, ']')

	// no optional segment, return original path (with param syntax conversion)
	if startIdx == -1 || endIdx == -1 {
		return []string{convertParamSyntax(path)}
	}

	// part before optional segment (not including '[')
	beforeOptional := path[:startIdx]
	// content inside optional segment (not including brackets)
	optionalContent := path[startIdx+1 : endIdx]
	// part after optional segment (not including ']')
	afterOptional := ""
	if endIdx+1 < len(path) {
		afterOptional = path[endIdx+1:]
	}

	// convert parameter syntax {param} -> :param
	beforeOptional = convertParamSyntax(beforeOptional)
	afterOptional = convertParamSyntax(afterOptional)
	optionalContent = convertParamSyntax(optionalContent)

	// version 1: skip optional segment
	withoutOptional := beforeOptional + afterOptional
	// version 2: include optional segment
	withOptional := beforeOptional + optionalContent + afterOptional

	return []string{withoutOptional, withOptional}
}

// convertParamSyntax converts parameter syntax {param} -> :param
// Also strips regex constraints: {id:\d+} -> :id
// And converts {file:.+} to *file (wildcard)
func convertParamSyntax(path string) string {
	for {
		start := strings.IndexByte(path, '{')
		if start == -1 {
			break
		}
		end := strings.IndexByte(path[start:], '}')
		if end == -1 {
			break
		}
		end += start // adjust to absolute position

		paramContent := path[start+1 : end]

		// strip regex constraint: {id:\d+} -> id
		paramName := paramContent
		if colonIdx := strings.IndexByte(paramContent, ':'); colonIdx > 0 {
			paramName = strings.TrimSpace(paramContent[:colonIdx])
			regexPart := strings.TrimSpace(paramContent[colonIdx+1:])

			// check if this is a wildcard-like pattern (.+)
			if regexPart == ".+" || regexPart == ".*" {
				// convert to wildcard: {file:.+} -> *file
				path = path[:start] + "*" + paramName + path[end+1:]
				continue
			}
			// otherwise strip the regex constraint
		}

		path = path[:start] + ":" + paramName + path[end+1:]
	}
	return path
}

/*************************************************************
 * global path params
 *************************************************************/

var globalVars = map[string]string{
	"all": `.*`,
	"any": `[^/]+`,
	"num": `[1-9][0-9]*`,
}

// SetGlobalVar set a global path var
func SetGlobalVar(name, regex string) {
	globalVars[name] = regex
}

// GetGlobalVars get all global path vars
func GetGlobalVars() map[string]string {
	return globalVars
}

/*************************************************************
 * help functions
 *************************************************************/

// isFixedPath returns true if path has no dynamic params or optional segments
func isFixedPath(s string) bool {
	return strings.IndexByte(s, '{') < 0 &&
		strings.IndexByte(s, '[') < 0 &&
		strings.IndexByte(s, ':') < 0 &&
		strings.IndexByte(s, '*') < 0
}

func simpleFmtPath(path string) string {
	path = strings.TrimSpace(path)

	if path == "" {
		return "/"
	}
	return "/" + strings.TrimLeft(path, "/")
}

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

		// "IP:PORT" OR ":PORT"
		if strings.IndexByte(addr[0], ':') != -1 {
			ss := strings.SplitN(addr[0], ":", 2)
			if ss[0] != "" {
				return addr[0]
			}
			port = ss[1]
		} else { // Only port
			port = addr[0]
		}

		return ip + ":" + port
	case 2: // "IP" + "PORT"
		return addr[0] + ":" + addr[1]
	default:
		panic("too many addr parameters")
	}
}

func debugPrintRoute(route *Route) {
	debugPrint(route.String())
}

func debugPrintError(err error) {
	if err != nil {
		debugPrint("<red>[ERROR]</> %v\n", err)
	}
}

func debugPrint(f string, v ...any) {
	if debug {
		color.Printf("<cyan>[FASTRUX-DEBUG]</> %s\n", fmt.Sprintf(f, v...))
	}
}

func parseAccept(acceptHeader string) []string {
	if acceptHeader == "" {
		return []string{}
	}

	parts := strings.Split(acceptHeader, ",")
	outs := make([]string, 0, len(parts))

	for _, part := range parts {
		if part = strings.TrimSpace(strings.Split(part, ";")[0]); part != "" {
			outs = append(outs, part)
		}
	}
	return outs
}

func formatMethodsWithDefault(methods []string, defMethod string) []string {
	if len(methods) == 0 {
		methods = []string{defMethod}
	} else {
		methods = formatMethods(methods)
	}
	return methods
}

func formatMethods(methods []string) (formatted []string) {
	for _, method := range methods {
		method = strings.TrimSpace(method)

		if method != "" {
			formatted = append(formatted, strings.ToUpper(method))
		}
	}
	return
}

/*************************************************************
 * Quick build uri by route name
 *************************************************************/

// M a short name for `map[string]any`
type M map[string]any

// BuildRequestURL struct
type BuildRequestURL struct {
	queries url.Values
	params  M
	path    string
	scheme  string
	host    string
	user    *url.Userinfo
}

// NewBuildRequestURL get new obj
func NewBuildRequestURL() *BuildRequestURL {
	return &BuildRequestURL{
		queries: make(url.Values),
		params:  make(M),
	}
}

// Queries set Queries
func (b *BuildRequestURL) Queries(queries url.Values) *BuildRequestURL {
	b.queries = queries
	return b
}

// Params set Params
func (b *BuildRequestURL) Params(params M) *BuildRequestURL {
	b.params = params
	return b
}

// Scheme set Scheme
func (b *BuildRequestURL) Scheme(scheme string) *BuildRequestURL {
	b.scheme = scheme
	return b
}

// User set User
func (b *BuildRequestURL) User(username, password string) *BuildRequestURL {
	b.user = url.UserPassword(username, password)
	return b
}

// Host set Host
func (b *BuildRequestURL) Host(host string) *BuildRequestURL {
	b.host = host
	return b
}

// Path set Path
func (b *BuildRequestURL) Path(path string) *BuildRequestURL {
	b.path = path
	return b
}

// Build url
func (b *BuildRequestURL) Build(withParams ...M) *url.URL {
	var path = b.path

	if len(withParams) > 0 {
		for k, d := range withParams[0] {
			if strings.IndexByte(k, '{') == -1 && strings.IndexByte(k, '}') == -1 {
				b.queries.Add(k, goutil.String(d))
			} else {
				b.params[k] = goutil.String(d)
			}
		}
	}

	var u = new(url.URL)

	u.Scheme = b.scheme
	u.User = b.user
	u.Host = b.host
	u.Path = path
	u.RawQuery = b.queries.Encode()

	return u
}
