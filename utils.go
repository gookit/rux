package rux

import (
	"fmt"
	"os"
	"strings"

	"github.com/gookit/color"
)

// normalizePath 标准化路径
func normalizePath(path string) string {
	if len(path) == 0 {
		return "/"
	}

	path = strings.TrimSpace(path)

	// 确保以 '/' 开头
	if path[0] != '/' {
		path = "/" + path
	}

	// 去除尾随 '/'（除非是根路径）
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	// 处理连续的斜杠，压缩为单个斜杠
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	return path
}

// normalizePathStrict 标准化路径但保留末尾斜杠（用于 strictLastSlash 模式）
func normalizePathStrict(path string) string {
	if len(path) == 0 {
		return "/"
	}

	path = strings.TrimSpace(path)

	// 确保以 '/' 开头
	if path[0] != '/' {
		path = "/" + path
	}

	// 注意：在 strict 模式下，保留末尾斜杠
	// 只处理连续的斜杠，压缩为单个斜杠
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	return path
}

// longestCommonPrefix 查找最长公共前缀
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

// parseOptionalSegments 解析可选段并展开为多条路由
// 输入：/posts[/{id}]
// 输出：["/posts", "/posts/:id"]
//
// 输入：/api/users[/{name}]/profile
// 输出：["/api/users/profile", "/api/users/:name/profile"]
func parseOptionalSegments(path string) []string {
	var results []string

	// 找到可选段的开始和结束位置
	startIdx := strings.IndexByte(path, '[')
	endIdx := strings.IndexByte(path, ']')

	// 没有可选段，返回原路径（转换参数语法）
	if startIdx == -1 || endIdx == -1 {
		return []string{convertParamSyntax(path)}
	}

	// 可选段之前的部分（不包括 '['）
	beforeOptional := path[:startIdx]
	// 可选段内部的内容（不包括括号）
	optionalContent := path[startIdx+1 : endIdx]
	// 可选段之后的部分（不包括 ']'）
	afterOptional := ""
	if endIdx+1 < len(path) {
		afterOptional = path[endIdx+1:]
	}

	// 转换参数语法 {param} -> :param
	beforeOptional = convertParamSyntax(beforeOptional)
	afterOptional = convertParamSyntax(afterOptional)
	optionalContent = convertParamSyntax(optionalContent)

	// 版本 1：跳过可选段
	withoutOptional := beforeOptional + afterOptional
	results = append(results, withoutOptional)

	// 版本 2：包含可选段
	withOptional := beforeOptional + optionalContent + afterOptional
	results = append(results, withOptional)

	return results
}

// convertParamSyntax converts parameter syntax {param} -> :param
// Also strips regex constraints: {id:\d+} -> :id
// And converts {file:.+} or {file:.*} to *file (wildcard)
// Handles nested braces in regex patterns: {level:[1-9]{1,2}} -> :level
func convertParamSyntax(path string) string {
	for {
		start := strings.IndexByte(path, '{')
		if start == -1 {
			break
		}
		// Find matching closing brace using balanced brace counting
		depth := 0
		end := -1
		for i := start; i < len(path); i++ {
			if path[i] == '{' {
				depth++
			} else if path[i] == '}' {
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
		}
		if end == -1 {
			break
		}

		paramContent := path[start+1 : end]

		// strip regex constraint: {id:\d+} -> id
		paramName := paramContent
		if colonIdx := strings.IndexByte(paramContent, ':'); colonIdx > 0 {
			paramName = strings.TrimSpace(paramContent[:colonIdx])
			regexPart := strings.TrimSpace(paramContent[colonIdx+1:])

			// check if this is a wildcard-like pattern (.+ or .*)
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
	// if debug {
	// 	fmt.Println("[SUX-DEBUG]", route.String())
	// }
	debugPrint(route.String())
}

func debugPrintError(err error) {
	if err != nil {
		debugPrint("<red>[ERROR]</> %v\n", err)
	}
}

func debugPrint(f string, v ...any) {
	if debug {
		// fmt.Printf("[RUX-DEBUG] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
		color.Printf("<cyan>[RUX-DEBUG]</> %s\n", fmt.Sprintf(f, v...))
	}
}

// from gin framework. TODO use httpreq.ParseAccept() instead.
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
