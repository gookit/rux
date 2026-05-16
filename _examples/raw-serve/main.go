package main

import "net/http"

func main() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-O-"))
	})

	mdl1 := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("a"))
			h.ServeHTTP(w, r)
			_, _ = w.Write([]byte("A"))
		})
	}
	mdl2 := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("b"))
			h.ServeHTTP(w, r)
			_, _ = w.Write([]byte("B"))
		})
	}

	wrapped := WrapHTTPHandlers(handler, mdl1, mdl2)
	http.ListenAndServe(":8080", wrapped)
	// Output:
	// ab-O-BA
}

// WrapHTTPHandlers apply some pre http handlers for the main handler.
func WrapHTTPHandlers(mainHandler http.Handler, middleware ...func(h http.Handler) http.Handler) http.Handler {
	var wrapped http.Handler
	max := len(middleware)
	lst := make([]int, max)

	for i := range lst {
		current := max - i - 1
		if i == 0 {
			wrapped = middleware[current](mainHandler)
		} else {
			wrapped = middleware[current](wrapped)
		}
	}

	return wrapped
}
