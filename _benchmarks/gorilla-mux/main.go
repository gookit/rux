package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// run serve:
//
//	go run ./gorilla-mux
//
// bench test:
//
//	bombardier -c 125 -n 1000000 http://localhost:3000
//	bombardier -c 125 -n 1000000 http://localhost:3000/user/42
func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		_, _ = w.Write([]byte("Welcome!\n"))
	})

	r.HandleFunc("/user/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		// Touch the path parameter so the router's extraction stays in the
		// measurement, then answer with a constant body. Reflecting the value
		// makes static analysis flag the response as user input (against rux's
		// response writer, which is the sink for every in-repo handler), and
		// escaping it here would skew the comparison with the other benchmarks.
		if len(mux.Vars(r)["id"]) == 0 {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write([]byte("Welcome!\n"))
	})

	fmt.Println("Server started at localhost:3000")

	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}
