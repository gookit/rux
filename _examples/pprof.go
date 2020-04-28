package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gookit/rux"
	"github.com/gookit/rux/pprof"
)

// run serve:
// 	go run ./_examples/pprof.go
// access page:
// 	http://localhost:3000/debug/pprof
func main() {
	// debug
	rux.Debug(true)
	r := rux.New()

	pprof.UsePProf(r)

	r.GET("/", func(c *rux.Context) {
		_, _ = c.Resp.Write([]byte("Welcome!\n"))
	})

	r.GET("/user/{id}", func(c *rux.Context) {
		c.WriteString(c.Param("id"))
	})

	fmt.Println("Server started at localhost:3000")

	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}
