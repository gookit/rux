package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	r := http.NewServeMux()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		_, _ = w.Write([]byte("Welcome!\n"))
	})

	// Notice: raw mux not support dynamic path route

	fmt.Println("Server started at localhost:3000")

	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}
