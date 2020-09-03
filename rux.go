// Package rux is a simple and fast request router for golang HTTP applications.
//
// Source code and other details for the project are available at GitHub:
// 		https://github.com/gookit/rux
//
// Usage please ref examples and README
package rux

import (
	"strings"

	"github.com/gookit/color"
)

// All supported HTTP verb methods name
const (
	GET     = "GET"
	PUT     = "PUT"
	HEAD    = "HEAD"
	POST    = "POST"
	PATCH   = "PATCH"
	TRACE   = "TRACE"
	DELETE  = "DELETE"
	CONNECT = "CONNECT"
	OPTIONS = "OPTIONS"
)

// Match status:
// - 1: found
// - 2: not found
// - 3: method not allowed
const (
	Found uint8 = iota + 1
	NotFound
	NotAllowed
)

// ControllerFace a simple controller interface
type ControllerFace interface {
	// AddRoutes for support register routes in the controller.
	AddRoutes(g *Router)
}

var (
	debug bool
	// current supported HTTP method
	// all supported methods string, use for method check
	// more: ,COPY,PURGE,LINK,UNLINK,LOCK,UNLOCK,VIEW,SEARCH
	anyMethods = []string{GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD, CONNECT, TRACE}
)

// Debug switch debug mode
func Debug(val bool) {
	debug = val
	if debug {
		color.Info.Println(" NOTICE, rux DEBUG mode is opened by rux.Debug(true)")
		color.Info.Println("======================================================")
	}
}

// IsDebug return rux is debug mode.
func IsDebug() bool {
	return debug
}

// AnyMethods get all methods
func AnyMethods() []string {
	return anyMethods
}

// AllMethods get all methods
func AllMethods() []string {
	return anyMethods
}

// MethodsString of all supported methods
func MethodsString() string {
	return strings.Join(anyMethods, ",")
}
