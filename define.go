package rux

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
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

	// RESTFul action methods definition
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

/*************************************************************
 * Route Params/Info
 *************************************************************/

// RouteInfo simple route info struct
type RouteInfo struct {
	Name, Path, HandlerName string
	// supported method of the route
	Methods    []string
	HandlerNum int
}

// Params for current route
type Params map[string]string

// Has param key in the Params
func (p Params) Has(key string) bool {
	_, ok := p[key]
	return ok
}

// String get string value by key
func (p Params) String(key string) (val string) {
	if val, ok := p[key]; ok {
		return val
	}
	return
}

// Int get int value by key
func (p Params) Int(key string) (val int) {
	if str, ok := p[key]; ok {
		val, err := strconv.Atoi(str)
		if err == nil {
			return val
		}
	}
	return
}

/*************************************************************
 * internal vars
 *************************************************************/

// "/users/{id}" "/users/{id:\d+}" `/users/{uid:\d+}/blog/{id}`
var varRegex = regexp.MustCompile(`{[^/]+}`)

var internal404Handler HandlerFunc = func(c *Context) {
	http.NotFound(c.Resp, c.Req)
}

var internal405Handler HandlerFunc = func(c *Context) {
	allowed := c.MustGet(CTXAllowedMethods).([]string)
	sort.Strings(allowed)
	c.SetHeader("Allow", strings.Join(allowed, ", "))

	if c.Req.Method == OPTIONS {
		c.SetStatus(200)
	} else {
		http.Error(c.Resp, "Method not allowed", 405)
	}
}
