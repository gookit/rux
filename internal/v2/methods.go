package v2

// HTTP method names — duplicated from root rux package for internal use
// without an import cycle. The root package re-exports these via constants.
const (
	GET     = "GET"
	HEAD    = "HEAD"
	POST    = "POST"
	PUT     = "PUT"
	PATCH   = "PATCH"
	DELETE  = "DELETE"
	OPTIONS = "OPTIONS"
	CONNECT = "CONNECT"
	TRACE   = "TRACE"
)

// methodCount is the total number of HTTP methods rux understands.
const methodCount = 9

// methodIndex maps an HTTP method string to a 0..8 array index.
// Returns -1 for unknown methods.
func methodIndex(m string) int {
	if len(m) == 0 {
		return -1
	}
	switch m[0] {
	case 'G':
		if m == GET {
			return 0
		}
	case 'H':
		if m == HEAD {
			return 1
		}
	case 'P':
		if m == POST {
			return 2
		}
		if m == PUT {
			return 3
		}
		if m == PATCH {
			return 4
		}
	case 'D':
		if m == DELETE {
			return 5
		}
	case 'O':
		if m == OPTIONS {
			return 6
		}
	case 'C':
		if m == CONNECT {
			return 7
		}
	case 'T':
		if m == TRACE {
			return 8
		}
	}
	return -1
}
