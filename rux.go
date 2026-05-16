// Package rux is a high-performance HTTP router for Go.
//
// This package is the public API surface; all implementation lives in
// internal/v2. Public symbols are type aliases / function vars over the
// internal types so users get a single coherent import path.
//
// Source: https://github.com/gookit/rux
package rux

import (
	v2 "github.com/gookit/rux/internal/v2"
)

// HTTP method constants.
const (
	GET     = v2.GET
	HEAD    = v2.HEAD
	POST    = v2.POST
	PUT     = v2.PUT
	PATCH   = v2.PATCH
	DELETE  = v2.DELETE
	OPTIONS = v2.OPTIONS
	CONNECT = v2.CONNECT
	TRACE   = v2.TRACE
)

// Content-Type and disposition header constants.
const (
	ContentType        = v2.ContentType
	ContentBinary      = v2.ContentBinary
	ContentDisposition = v2.ContentDisposition
)

// Context keys exposed by the dispatcher.
const (
	CTXAllowedMethods = v2.CTXAllowedMethods
	CTXRecoverResult  = v2.CTXRecoverResult
)

// Public types — all aliased to the internal/v2 implementation.
type (
	Router          = v2.Router
	Context         = v2.Context
	Route           = v2.Route
	HandlerFunc     = v2.HandlerFunc
	HandlersChain   = v2.HandlersChain
	Param           = v2.Param
	Params          = v2.Params
	RouteInfo       = v2.RouteInfo
	M               = v2.M
	BuildRequestURL = v2.BuildRequestURL
	Renderer        = v2.Renderer
	Validator       = v2.Validator
	ControllerFace  = v2.ControllerFace
)

// REST action names.
var (
	IndexAction  = v2.IndexAction
	CreateAction = v2.CreateAction
	StoreAction  = v2.StoreAction
	ShowAction   = v2.ShowAction
	EditAction   = v2.EditAction
	UpdateAction = v2.UpdateAction
	DeleteAction = v2.DeleteAction

	RESTFulActions = v2.RESTFulActions
)

// Constructor and lifecycle helpers.
var (
	New                = v2.New
	NewBuildRequestURL = v2.NewBuildRequestURL
	Debug              = v2.Debug
	IsDebug            = v2.IsDebug
	AnyMethods         = v2.AnyMethods
	AllMethods         = v2.AllMethods
	MethodsString      = v2.MethodsString
)

// Router options.
var (
	StrictLastSlash        = v2.StrictLastSlash
	UseEncodedPath         = v2.UseEncodedPath
	HandleMethodNotAllowed = v2.HandleMethodNotAllowed
	HandleFallbackRoute    = v2.HandleFallbackRoute
	InterceptAll           = v2.InterceptAll
)

// Middleware adapters (wrap http.Handler / http.HandlerFunc as HandlerFunc).
var (
	WrapH               = v2.WrapH
	HTTPHandler         = v2.HTTPHandler
	WrapHTTPHandler     = v2.WrapHTTPHandler
	WrapHF              = v2.WrapHF
	HTTPHandlerFunc     = v2.HTTPHandlerFunc
	WrapHTTPHandlerFunc = v2.WrapHTTPHandlerFunc
)
