package v2

import "strconv"

// MaxParams is the maximum number of path parameters per route.
// Inlined storage in Context avoids any heap allocation for params.
// Registering a route that exceeds this limit panics.
const MaxParams = 16

// Param is a single path parameter.
type Param struct {
	Key   string
	Value string
}

// Params is a fixed-capacity inline parameter container.
// It lives directly inside Context, providing zero-allocation parameter
// passing when the Context itself is reused via sync.Pool.
type Params struct {
	data [MaxParams]Param
	n    uint8
}

// Len returns the number of parameters.
func (p *Params) Len() int { return int(p.n) }

// Get returns the value for name, or "" if not found.
func (p *Params) Get(name string) string {
	for i := uint8(0); i < p.n; i++ {
		if p.data[i].Key == name {
			return p.data[i].Value
		}
	}
	return ""
}

// Has reports whether a parameter with the given name exists.
func (p *Params) Has(name string) bool {
	for i := uint8(0); i < p.n; i++ {
		if p.data[i].Key == name {
			return true
		}
	}
	return false
}

// Int parses the named parameter as int, returning 0 on miss or parse error.
func (p *Params) Int(name string) int {
	if v := p.Get(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}

// Snapshot returns a heap-allocated copy of the params slice.
// Use this when you need to retain params beyond the handler scope
// (e.g., goroutines, async logging).
func (p *Params) Snapshot() []Param {
	out := make([]Param, p.n)
	for i := uint8(0); i < p.n; i++ {
		out[i] = p.data[i]
	}
	return out
}

// Reset clears the params (called by Context.Reset on pool return).
func (p *Params) Reset() { p.n = 0 }

// append adds a parameter. Panics if MaxParams is exceeded.
// Callers in tree.lookup should ensure tree.maxParams <= MaxParams at registration.
func (p *Params) append(key, value string) {
	if p.n >= MaxParams {
		panic("rux: params overflow (MaxParams=" + strconv.Itoa(MaxParams) + ")")
	}
	p.data[p.n].Key = key
	p.data[p.n].Value = value
	p.n++
}
