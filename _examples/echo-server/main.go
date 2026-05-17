// Echo server example — a httpbin-style HTTP debug server.
//
// Two usage patterns shown:
//  1. NewEchoServer() — standalone server, fastest path.
//  2. MountEchoRoutes(r) under a group — embed echo as a /debug subtree
//     inside an existing application.
//
// Run standalone:
//
//	go run ./_examples/echo-server
//
// Run embedded mode:
//
//	go run ./_examples/echo-server -mode=embed
//
// Then try:
//
//	curl http://127.0.0.1:18080/anything
//	curl http://127.0.0.1:18080/status/418
//	curl -OJ "http://127.0.0.1:18080/download/test.txt?type=text&size=2048"
//	curl -F "file=@./main.go" http://127.0.0.1:18080/upload
package main

import (
	"flag"
	"log"

	"github.com/gookit/rux/v2"
	"github.com/gookit/rux/v2/server"
)

func main() {
	mode := flag.String("mode", "standalone", "standalone | embed")
	addr := flag.String("addr", "127.0.0.1:18080", "listen address")
	flag.Parse()

	switch *mode {
	case "embed":
		runEmbedded(*addr)
	default:
		runStandalone(*addr)
	}
}

// runStandalone serves the full echo API at the root of the listener.
func runStandalone(addr string) {
	s := server.NewEchoServer()
	s.Addr = addr
	log.Printf("echo server (standalone) listening on http://%s", addr)
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}

// runEmbedded mounts echo routes under /debug inside a larger app router.
// Useful when you only want debug endpoints exposed alongside your real API.
func runEmbedded(addr string) {
	s := server.New(true) // debugMode → enables RequestLogger
	s.Addr = addr

	s.GET("/", func(c *rux.Context) {
		c.Text(200, "main app — see /debug/anything for echo endpoints")
	})

	// Mount the entire httpbin-style API under /debug.
	s.Group("/debug", func() {
		server.MountEchoRoutes(s.Router)
	})

	log.Printf("echo server (embedded under /debug) listening on http://%s", addr)
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
