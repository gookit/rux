# TODO

## new server

```go
srv := rux.NewServer()
```

## context

```go
type ServiceProvider interface {
	Service(name string) any
}

type ServiceProviderFunc(name string) any

// ServiceProviderFunc implements ServiceProvider
func (f ServiceProviderFunc) Service(name string) any {
    return f(name)
}

type Context struct {
    // add service provider
	ProviderFunc ServiceProvider

	// app
	srv *Server
	app *Server
	app *Router
}

// Service returns a service by name. eg: c.Service("db")
func (c *Context) Service(name string) any {
    return c.ProviderFunc(name)
}
```

