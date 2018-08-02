package main

import "github.com/gookit/souter"

func main() {
	r := souter.New()
	r.Use()

	r.Match()
}
