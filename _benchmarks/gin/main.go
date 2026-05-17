package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// run serve:
// 	go run ./gin
// bench test:
// 	bombardier -c 125 -n 1000000 http://localhost:3000
// 	bombardier -c 125 -n 1000000 http://localhost:3000/user/42
func main() {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.GET("/", func(c *gin.Context) {
		_, _ = c.Writer.Write([]byte("Welcome!\n"))
	})

	r.GET("/user/:id", func(c *gin.Context) {
		// Force text/plain so a "<script>" id can't trip browser sniff → XSS.
		c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = c.Writer.Write([]byte(c.Param("id")))
	})

	fmt.Println("Server started at localhost:3000")

	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}
