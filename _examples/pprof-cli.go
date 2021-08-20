package main

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	"github.com/gookit/rux"
)

// run serve:
// 	go run ./_examples/pprof-cli.go
// see prof on cli:
// 	go tool pprof rux_cpu.prof
// see prof on web:
// 	go tool pprof -http=:8080 rux_cpu.prof
func main() {
	// rux.Debug(true)
	r := rux.New()

	r.GET("/", func(c *rux.Context) {
		_, _ = c.Resp.Write([]byte("Welcome!\n"))
	})

	r.GET("/user/{id}", func(c *rux.Context) {
		c.WriteString(c.Param("id"))
	})

	times := 1000000
	fmt.Println("start profile, run times:", times)

	ruxProfile := "rux_cpu.prof"
	f, err := os.Create(ruxProfile)
	if err != nil {
		log.Fatal(err)
	}

	err = pprof.StartCPUProfile(f)
	if err != nil {
		log.Fatal(err)
	}

	defer pprof.StopCPUProfile()

	for i := 0; i < times; i++ {
		// r.QuickMatch("GET", "/")
		r.QuickMatch("GET", "/user/23")
		// r.Match("get", "/")
		// r.Match("get", "/users/23")
		// fmt.Println(ret)
	}

	fmt.Println("see prof on cli:\n  go tool pprof", ruxProfile)
	fmt.Println("see prof on web:\n  go tool pprof -http=:8080", ruxProfile)
}
