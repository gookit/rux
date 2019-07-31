package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gookit/rux"
)

// install bombardier:
// 	go get -u github.com/codesenberg/bombardier
// run serve:
// 	go run ./_benchmarks/rux
// bench test:
// 	bombardier -c 125 -n 1000000 http://localhost:3000
func main() {
	// close debug
	rux.Debug(false)
	r := rux.New()

	r.GET("/", func(c *rux.Context) {
		c.WriteString("welcome")
	})

	r.GET("/user/{id}", func(c *rux.Context) {
		c.WriteString(c.Param("id"))
	})

	fmt.Println("Server started at localhost:3000")

	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}
