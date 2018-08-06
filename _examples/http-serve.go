package main

import (
	"fmt"
	"net/http"
	"log"
	"html"
)

func main() {
	fooHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	}

	// http.Handle("/foo", fooHandler)
	http.HandleFunc("/foo", fooHandler)

	http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func fooHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello")
}

func mux() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"hello\": \"world\"}"))
	})

	// cors.Default() setup the middleware with default options being
	// all origins accepted with simple methods (GET, POST). See
	// documentation below for more options.
	handler := cors.Default().Handler(mux)
	http.ListenAndServe(":8080", handler)
}