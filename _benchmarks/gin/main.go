package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.GET("/", func(c *gin.Context) {
		_, _ = c.Writer.Write([]byte("Welcome!\n"))
	})

	r.GET("/user/:id", func(c *gin.Context) {
		_, _ = c.Writer.Write([]byte(c.Param("id")))
	})

	fmt.Println("Server started at localhost:3000")

	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}