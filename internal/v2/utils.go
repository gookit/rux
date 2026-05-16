package v2

import "strings"

// normalizePath enforces:
//   - leading '/'
//   - no trailing '/' (unless root)
//   - no doubled '//'
//
// Idempotent. Allocates only when changes are needed.
func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	// Fast path: already canonical.
	if path[0] == '/' && !strings.Contains(path, "//") &&
		(len(path) == 1 || path[len(path)-1] != '/') {
		return path
	}

	var b strings.Builder
	b.Grow(len(path) + 1)
	if path[0] != '/' {
		b.WriteByte('/')
	}
	prevSlash := false
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '/' {
			if prevSlash {
				continue
			}
			prevSlash = true
		} else {
			prevSlash = false
		}
		b.WriteByte(c)
	}
	out := b.String()
	if len(out) > 1 && out[len(out)-1] == '/' {
		out = out[:len(out)-1]
	}
	return out
}

// isStaticPath reports whether path contains no dynamic segments.
func isStaticPath(path string) bool {
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '{', '[', ':', '*':
			return false
		}
	}
	return true
}

// longestCommonPrefix returns the length of the longest common byte prefix.
func longestCommonPrefix(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}
