// Package rux is a high-performance HTTP router for Go.
//
// This package is the public API surface; all implementation lives in
// internal/core. Public symbols are type aliases / function vars over
// the internal types so users get a single coherent import path.
//
// Source: https://github.com/gookit/rux
package rux

import (
	"github.com/gookit/rux/internal/core"
)

// HTTP method constants.
const (
	GET     = core.GET
	HEAD    = core.HEAD
	POST    = core.POST
	PUT     = core.PUT
	PATCH   = core.PATCH
	DELETE  = core.DELETE
	OPTIONS = core.OPTIONS
	CONNECT = core.CONNECT
	TRACE   = core.TRACE
)

// Content-Type and disposition header constants.
const (
	ContentType        = core.ContentType
	ContentBinary      = core.ContentBinary
	ContentDisposition = core.ContentDisposition
)

// Context keys exposed by the dispatcher.
const (
	CTXAllowedMethods = core.CTXAllowedMethods
	CTXRecoverResult  = core.CTXRecoverResult
)

// Public types — all aliased to the internal/core implementation.
type (
	Router          = core.Router
	Context         = core.Context
	Route           = core.Route
	HandlerFunc     = core.HandlerFunc
	HandlersChain   = core.HandlersChain
	Param           = core.Param
	Params          = core.Params
	RouteInfo       = core.RouteInfo
	M               = core.M
	BuildRequestURL = core.BuildRequestURL
	Renderer        = core.Renderer
	Validator       = core.Validator
	ControllerFace  = core.ControllerFace
)

// REST action names.
var (
	IndexAction  = core.IndexAction
	CreateAction = core.CreateAction
	StoreAction  = core.StoreAction
	ShowAction   = core.ShowAction
	EditAction   = core.EditAction
	UpdateAction = core.UpdateAction
	DeleteAction = core.DeleteAction

	RESTFulActions = core.RESTFulActions
)

// Constructor and lifecycle helpers.
var (
	New                = core.New
	NewBuildRequestURL = core.NewBuildRequestURL
	Debug              = core.Debug
	IsDebug            = core.IsDebug
	AnyMethods         = core.AnyMethods
	AllMethods         = core.AllMethods
	MethodsString      = core.MethodsString
)

// Router options.
var (
	StrictLastSlash        = core.StrictLastSlash
	UseEncodedPath         = core.UseEncodedPath
	HandleMethodNotAllowed = core.HandleMethodNotAllowed
	HandleFallbackRoute    = core.HandleFallbackRoute
	InterceptAll           = core.InterceptAll
)

// Middleware adapters (wrap http.Handler / http.HandlerFunc as HandlerFunc).
var (
	WrapH               = core.WrapH
	HTTPHandler         = core.HTTPHandler
	WrapHTTPHandler     = core.WrapHTTPHandler
	WrapHF              = core.WrapHF
	HTTPHandlerFunc     = core.HTTPHandlerFunc
	WrapHTTPHandlerFunc = core.WrapHTTPHandlerFunc
)
