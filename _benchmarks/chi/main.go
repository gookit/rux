package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// run serve:
//
//	go run ./chi
//
// bench test:
//
//	bombardier -c 125 -n 1000000 http://localhost:3000
//	bombardier -c 125 -n 1000000 http://localhost:3000/user/42
func main() {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome!\n"))
	})

	r.Get("/user/{id}", func(w http.ResponseWriter, r *http.Request) {
		// Touch the path parameter so the router's extraction stays in the
		// measurement, then answer with a constant body. Reflecting the value
		// makes static analysis flag the response as user input (against rux's
		// response writer, which is the sink for every in-repo handler), and
		// escaping it here would skew the comparison with the other benchmarks.
		if len(chi.URLParam(r, "id")) == 0 {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write([]byte("Welcome!\n"))
	})

	fmt.Println("Server started at localhost:3000")
	http.ListenAndServe(":3000", r)
}
