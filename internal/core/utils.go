package core

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

// hasOptionalSegment reports whether path contains an optional segment
// like "[/{id}]" or "[.html]" outside of brace-quoted parameter regex.
func hasOptionalSegment(path string) bool {
	inBraces := false
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '{':
			inBraces = true
		case '}':
			inBraces = false
		case '[':
			if !inBraces {
				return true
			}
		}
	}
	return false
}

// parseOptionalSegments expands "/posts[/{id}]" into
// {"/posts", "/posts/:id"}. Always returns at least one element.
// Caller must have validated the path with util.ValidateOptionalSegments first.
func parseOptionalSegments(path string) []string {
	start := strings.IndexByte(path, '[')
	end := strings.IndexByte(path, ']')
	if start < 0 || end < 0 {
		return []string{convertParamSyntax(path)}
	}

	before := convertParamSyntax(path[:start])
	inner := convertParamSyntax(path[start+1 : end])
	after := ""
	if end+1 < len(path) {
		after = convertParamSyntax(path[end+1:])
	}
	return []string{before + after, before + inner + after}
}

// convertParamSyntax rewrites Rux's brace param syntax to colon syntax.
//
//	{id}        -> :id
//	{id:\d+}    -> :id          (regex stripped — see P-14)
//	{file:.+}   -> *file        (catch-all)
//	{file:.*}   -> *file
func convertParamSyntax(path string) string {
	for {
		start := strings.IndexByte(path, '{')
		if start == -1 {
			return path
		}
		// Find matching '}' with brace counting (regex may contain '{1,2}').
		depth := 0
		end := -1
		for i := start; i < len(path); i++ {
			switch path[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = i
				}
			}
			if end != -1 {
				break
			}
		}
		if end == -1 {
			return path
		}

		content := path[start+1 : end]
		name := content
		if colon := strings.IndexByte(content, ':'); colon > 0 {
			name = strings.TrimSpace(content[:colon])
			regex := strings.TrimSpace(content[colon+1:])
			if regex == ".+" || regex == ".*" {
				path = path[:start] + "*" + name + path[end+1:]
				continue
			}
		}
		path = path[:start] + ":" + name + path[end+1:]
	}
}
