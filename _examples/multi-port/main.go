package main

import (
	"fmt"
	"log"
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
		err := http.ListenAndServe(PORT, nil)
		if err != nil && err != http.ErrServerClosed {
			// Error starting or closing listener:
			log.Printf("HTTP server ListenAndServe: %v", err)
		}
	})

	Launch(func() {
		err := http.ListenAndServeTLS(":443", "cert.pem", "key.pem", nil)
		if err != nil && err != http.ErrServerClosed {
			// Error starting or closing listener:
			log.Printf("HTTP server ListenAndServeTLS: %v", err)
		}
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
	_,_= fmt.Fprintf(w, "hello, gopher")
}
