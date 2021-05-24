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

const (
	anyMatch = `[^/]+`
)

const (
	// ContentType header key
	ContentType = "Content-Type"
	// ContentBinary represents content type application/octet-stream
	ContentBinary = "application/octet-stream"

	// ContentDisposition describes contentDisposition
	ContentDisposition = "Content-Disposition"
	// describes content disposition type
	dispositionInline = "inline"
	// describes content disposition type
	dispositionAttachment = "attachment"
)

const (
	// CTXMatchResult key name in the context
	// CTXMatchResult = "_matchResult"

	// CTXRecoverResult key name in the context
	CTXRecoverResult = "_recoverResult"
	// CTXAllowedMethods key name in the context
	CTXAllowedMethods = "_allowedMethods"
	// CTXCurrentRouteName key name in the context
	CTXCurrentRouteName = "_currentRouteName"
	// CTXCurrentRoutePath key name in the context
	CTXCurrentRoutePath = "_currentRoutePath"
)

type routes []*Route

// like "GET": [ Route, ...]
type methodRoutes map[string]routes

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

// RESTFul method names definition
var (
	IndexAction  = "Index"
	CreateAction = "Create"
	StoreAction  = "Store"
	ShowAction   = "Show"
	EditAction   = "Edit"
	UpdateAction = "Update"
	DeleteAction = "Delete"

	// RESTFulActions action methods definition
	RESTFulActions = map[string][]string{
		IndexAction:  {GET},
		CreateAction: {GET},
		StoreAction:  {POST},
		ShowAction:   {GET},
		EditAction:   {GET},
		UpdateAction: {PUT, PATCH},
		DeleteAction: {DELETE},
	}
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
