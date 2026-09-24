package binding

import (
	"net/http"
)

type (
	// Binder interface
	Binder interface {
		Name() string
		Bind(r *http.Request, obj any) error
	}

	// BinderFunc bind func, implement the Binder() interface
	BinderFunc func(r *http.Request, obj any) error

	// DataValidator interface
	DataValidator interface {
		Validate(i any) error
	}

	// RequestValidator is an optional extension of DataValidator. When the
	// installed Validator implements it, every binder that holds the request
	// calls ValidateRequest instead of Validate, so a rule can look at the raw
	// request (an uploaded file, for example) and not only at the bound struct.
	RequestValidator interface {
		DataValidator
		ValidateRequest(r *http.Request, obj any) error
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
		"file":   File,
		// TODO more driver
		// "yaml": YAML,
		// "msgpack": MSGPACK,
		// "protobuf": PROTOBUF,
	}
)

// Name implements the Binder interface
func (fn BinderFunc) Name() string {
	return "unknown"
}

// Bind implements the Binder interface
func (fn BinderFunc) Bind(r *http.Request, obj any) error {
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
		delete(Binders, name)
	}
}
