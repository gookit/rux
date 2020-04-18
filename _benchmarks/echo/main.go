package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo"
)

// run serve:
// 	go run ./echo
// bench test:
// 	bombardier -c 125 -n 1000000 http://localhost:3000
// 	bombardier -c 125 -n 1000000 http://localhost:3000/user/42
func main() {
	r := echo.New()

	r.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome!\n")
	})

	r.GET("/user/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, c.Param("id"))
	})

	fmt.Println("Server started at localhost:3000")
	err := r.Start(":3000")
	if err != nil {
		log.Fatal(err)
	}
}
