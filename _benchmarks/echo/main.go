package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
)

func main() {
	r := echo.New()

	r.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome!\n")
	})

	r.GET("/user/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, c.Param("id"))
	})

	fmt.Println("Server started at localhost:3000")
	r.Start(":3000")
}