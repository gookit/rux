package main

import (
	"log"

	// "github.com/alecthomas/kingpin"
	"github.com/valyala/fasthttp"
)

// from https://github.com/codesenberg/bombardier/blob/master/cmd/utils/simplebenchserver/main.go
// var serverPort = kingpin.Flag("port", "port to use for benchmarks").
// 	Default("8080").
// 	Short('p').
// 	String()
// var responseSize = kingpin.Flag("size", "size of response in bytes").
// 	Default("1024").
// 	Short('s').
// 	Uint()

// run serve:
// 	go run ./fasthttp
// bench test:
// 	bombardier -c 125 -n 1000000 http://localhost:3000
// 	bombardier -c 125 -n 1000000 http://localhost:3000/user/42
func main() {
	// kingpin.Parse()
	// response := strings.Repeat("a", int(*responseSize))
	// addr := "localhost:" + *serverPort
	response := "Welcome!\n"
	addr := "localhost:3000"
	log.Println("Starting HTTP server on:", addr)
	err := fasthttp.ListenAndServe(addr, func(c *fasthttp.RequestCtx) {
		_, werr := c.WriteString(response)
		if werr != nil {
			log.Println(werr)
		}
	})
	if err != nil {
		log.Println(err)
	}
}
