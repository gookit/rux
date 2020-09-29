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

// Debug switch debug mode
func Debug(val bool) {
	debug = val
	if debug {
		color.Info.Println("    NOTICE, rux DEBUG mode is opened by rux.Debug(true)")
		color.Info.Println("===========================================================")
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

/*************************************************************
 * Router options
 *************************************************************/

// InterceptAll setting for the router
func InterceptAll(path string) func(*Router) {
	return func(r *Router) {
		r.interceptAll = strings.TrimSpace(path)
	}
}

// MaxNumCaches setting for the router
func MaxNumCaches(num uint16) func(*Router) {
	return func(r *Router) {
		r.maxNumCaches = num
	}
}

// CachingWithNum for the router
func CachingWithNum(num uint16) func(*Router) {
	return func(r *Router) {
		r.maxNumCaches = num
		r.enableCaching = true
	}
}

// UseEncodedPath enable for the router
func UseEncodedPath(r *Router) {
	r.useEncodedPath = true
}

// EnableCaching for the router
func EnableCaching(r *Router) {
	r.enableCaching = true
}

// StrictLastSlash enable for the router
func StrictLastSlash(r *Router) {
	r.strictLastSlash = true
}

// MaxMultipartMemory set max memory limit for post forms
// func MaxMultipartMemory(max int64) func(*Router) {
// 	return func(r *Router) {
// 		r.maxMultipartMemory = max
// 	}
// }

// HandleFallbackRoute enable for the router
func HandleFallbackRoute(r *Router) {
	r.handleFallbackRoute = true
}

// HandleMethodNotAllowed enable for the router
func HandleMethodNotAllowed(r *Router) {
	r.handleMethodNotAllowed = true
}
