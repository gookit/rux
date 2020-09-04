package binding

import (
	"net/http"
)

type (
	// Binder interface
	Binder interface {
		Bind(r *http.Request, obj interface{}) error
	}

	// DataValidator interface
	DataValidator interface {
		Validate(i interface{}) error
	}
)

// build-in data binder
var (
	Query = QueryBinder{}
	JSON  = JSONBinder{}
	XML   = XMLBinder{}
)

var binders = map[string]Binder{
	"xml":   XML,
	"json":  JSON,
	"query": Query,
	// TODO more driver
	// "form": ,
	// "yml": ,
	// "header": ,
	// "msgpack": ,
	// "protobuf": ,
}

// BinderFunc bind func
type BinderFunc func(interface{}, *http.Request) error

// BinderFunc implements the Binder interface
func (fn BinderFunc) Name() string {
	return "unknown"
}

// BinderFunc implements the Binder interface
func (fn BinderFunc) Bind(r *http.Request, obj interface{}) error {
	return fn(ptr, r)
}

// Register new binder with name
func Register(name string, b Binder) {
	if name != "" && b != nil {
		binders[name] = b
	}
}

// Remove a exist binder
func Remove(name string) {
	if _, ok := binders[name]; ok {
		delete(binders, name)
	}
}
