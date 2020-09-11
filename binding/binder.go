package binding

import (
	"net/http"
)

type (
	// Binder interface
	Binder interface {
		Name() string
		Bind(r *http.Request, obj interface{}) error
	}

	// BinderFunc bind func, implement the Binder() interface
	BinderFunc func(r *http.Request, obj interface{}) error

	// DataValidator interface
	DataValidator interface {
		Validate(i interface{}) error
	}
)

// build-in data binder
var (
	Form   = FormBinder{FormTagName}
	Header = HeaderBinder{HeaderTagName}
	Query  = QueryBinder{QueryTagName}

	JSON = JSONBinder{}
	XML  = XMLBinder{}
	// TODO more driver
	// YAML = YAMLBinder{}
	// MSGPACK = MSGPACKBinder{}
	// PROTOBUF = PROTOBUFBinder{}

	// Binders mapping
	Binders = map[string]Binder{
		"xml":    XML,
		"json":   JSON,
		"query":  Query,
		"form":   Form,
		"header": Header,
		// TODO more driver
		// "yaml": YAML,
		// "msgpack": MSGPACK,
		// "protobuf": PROTOBUF,
	}
)

// BinderFunc implements the Binder interface
func (fn BinderFunc) Name() string {
	return "unknown"
}

// BinderFunc implements the Binder interface
func (fn BinderFunc) Bind(r *http.Request, obj interface{}) error {
	return fn(r, obj)
}

// GetBinder get an binder by name
func GetBinder(name string) Binder {
	if b, ok := Binders[name]; ok {
		return b
	}
	return nil
}

// Register new binder with name
func Register(name string, b Binder) {
	if name != "" && b != nil {
		Binders[name] = b
	}
}

// Remove exists binder(s)
func Remove(names ...string) {
	for _, name := range names {
		if _, ok := Binders[name]; ok {
			delete(Binders, name)
		}
	}
}
