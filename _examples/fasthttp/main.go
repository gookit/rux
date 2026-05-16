package main

// TODO(v2): fasthttp adapter not yet ported to v2 — see https://github.com/gookit/rux/issues
// Original v1 code is preserved below for reference; will be re-enabled once
// a v2 fasthttp adapter (ServeFastHTTP / request wrapping) is implemented.

/*
import (
	"fmt"
	"log"
	"net/http"

	"github.com/gookit/rux/v2"
	"github.com/valyala/fasthttp"
)

// examples for run rux on fasthttp
// run demo:
// 	go run ./_examples/fasthttp
// access page:
// 	http://localhost:3000/debug/pprof
// fasthttp github: https://github.com/valyala/fasthttp
func main() {
	// debug
	rux.Debug(true)
	r := rux.New()

	r.GET("/", func(c *rux.Context) {
		_, _ = c.Resp.Write([]byte("Welcome!\n"))
	})

	r.GET("/user/{id}", func(c *rux.Context) {
		c.WriteString(c.Param("id"))
	})

	fmt.Println("Server started at localhost:3000")

	addr := "localhost:3000"
	log.Println("Starting HTTP server on:", addr)
	err := fasthttp.ListenAndServe(addr, func(c *fasthttp.RequestCtx) {
		_, werr := c.WriteString("hello")
		if werr != nil {
			log.Println(werr)
		}

		// TODO ...
		r.ServeFastHTTP(c.Request, c.Response)
	})

	if err != nil {
		log.Println(err)
	}
}

func wrapFastHTTPContext(fc *fasthttp.RequestCtx) (w http.ResponseWriter, r *http.Request) {
	return
}

func wrapFastHTTPContext1(fc *fasthttp.RequestCtx) (c *rux.Context) {
	return
}

func wrapFastHTTPRequest() {

}
*/

func main() {
	// placeholder until v2 fasthttp adapter is ported
}
