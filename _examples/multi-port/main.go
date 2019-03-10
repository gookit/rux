package main

import (
	"fmt"
	"net/http"
	"sync"
)

const (
	PORT = ":11024"
)

var wg sync.WaitGroup

func main() {
	http.HandleFunc("/", Hello)

	Launch(func() {
		http.ListenAndServe(PORT, nil)
	})

	Launch(func() {
		http.ListenAndServeTLS(":443", "cert.pem", "key.pem", nil)
	})

	wg.Wait()
}

func Launch(fn func()) {
	wg.Add(1)

	go func() {
		defer wg.Done()
		fn()
	}()
}

func Hello(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "hello, gopher")
}
