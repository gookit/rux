package dispatcher

type Params map[string]string

// Context for http server
type Context struct {
	Req Request
	Res ResponseWriter

	params Params

	// context data
	Data map[string]interface{}
}

func(c *Context) Next() {

}

type HandlerFunc func(*Context)
type HandlersChain []HandlerFunc

// Last returns the last handler in the chain. ie. the last handler is the main own.
func (c HandlersChain) Last() HandlerFunc {
	length := len(c)
	if length > 0 {
		return c[length-1]
	}
	return nil
}

