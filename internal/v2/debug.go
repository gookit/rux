package v2

import (
	"strings"

	"github.com/gookit/color"
)

// debug is the package-level debug flag toggled by Debug.
var debug bool

// Debug toggles verbose logging for rux. When enabled, the router prints
// startup banners and additional diagnostic output.
func Debug(val bool) {
	debug = val
	if debug {
		color.Info.Println("    NOTICE, rux DEBUG mode is opened by rux.Debug(true)")
		color.Info.Println("===========================================================")
	}
}

// IsDebug reports whether debug mode is active.
func IsDebug() bool {
	return debug
}

// AnyMethods returns the canonical list of HTTP methods supported by the
// router. The returned slice is shared — callers must not mutate it.
func AnyMethods() []string {
	return anyMethods
}

// AllMethods is a synonym for AnyMethods, kept for backward compatibility.
func AllMethods() []string {
	return anyMethods
}

// MethodsString returns the supported HTTP methods joined by commas.
func MethodsString() string {
	return strings.Join(anyMethods, ",")
}
