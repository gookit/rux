package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func main() {
	r := httprouter.New()

	r.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		_, _ = w.Write([]byte("Welcome!\n"))
	})

	r.GET("/user/:id", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		_, _ = w.Write([]byte(params.ByName("id")))
	})

	fmt.Println("Server started at localhost:3000")

	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}
