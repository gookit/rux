package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
)

// run serve:
// 	go run ./chi
// bench test:
// 	bombardier -c 125 -n 1000000 http://localhost:3000
// 	bombardier -c 125 -n 1000000 http://localhost:3000/user/42
func main() {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome!\n"))
	})

	r.Get("/user/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(chi.URLParam(r, "id")))
	})

	fmt.Println("Server started at localhost:3000")
	http.ListenAndServe(":3000", r)
}
