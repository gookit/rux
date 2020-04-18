package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// run serve:
// 	go run ./gorilla-mux
// bench test:
// 	bombardier -c 125 -n 1000000 http://localhost:3000
// 	bombardier -c 125 -n 1000000 http://localhost:3000/user/42
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

		_, _ = w.Write([]byte(mux.Vars(r)["id"]))
	})

	fmt.Println("Server started at localhost:3000")

	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}
