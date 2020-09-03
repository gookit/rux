package rux

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
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
 * help functions
 *************************************************************/

// no route params
func isFixedPath(s string) bool {
	return strings.IndexByte(s, '{') < 0 && strings.IndexByte(s, '[') < 0
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

func panicf(f string, v ...interface{}) {
	panic(fmt.Sprintf(f, v...))
}

func debugPrintError(err error) {
	if err != nil {
		debugPrint("[ERROR] %v\n", err)
	}
}

func debugPrint(f string, v ...interface{}) {
	if debug {
		msg := fmt.Sprintf(f, v...)
		// fmt.Printf("[RUX-DEBUG] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
		fmt.Printf("[RUX-DEBUG] %s\n", msg)
	}
}

// from gin framework
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

func formatMethods(methods []string) (formatted []string) {
	for _, method := range methods {
		method = strings.TrimSpace(method)

		if method != "" {
			formatted = append(formatted, strings.ToUpper(method))
		}
	}
	return
}

func formatMethodsWithDefault(methods []string, defMethod string) []string {
	if len(methods) == 0 {
		methods = []string{defMethod}
	} else {
		methods = formatMethods(methods)
	}

	return methods
}

func toString(i interface{}) string {
	if i == nil {
		return ""
	}

	switch value := i.(type) {
	case int:
		return strconv.Itoa(value)
	case int8:
		return strconv.Itoa(int(value))
	case int16:
		return strconv.Itoa(int(value))
	case int32:
		return strconv.Itoa(int(value))
	case int64:
		return strconv.Itoa(int(value))
	case uint:
		return strconv.FormatUint(uint64(value), 10)
	case uint8:
		return strconv.FormatUint(uint64(value), 10)
	case uint16:
		return strconv.FormatUint(uint64(value), 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value)
	case string:
		return value
	case []byte:
		return string(value)
	default:
		// String conversion using JSON by default
		jsonContent, _ := json.Marshal(value)
		return string(jsonContent)
	}
}
