package main

import (
	"regexp"
	"fmt"
)

func main() {
	var varRegex = regexp.MustCompile(`:([a-zA-Z0-9]+)`)

	path := `/users/:uid(\d+)/blog/:id`

	ss := varRegex.FindAllString(path, -1)

	fmt.Println(ss)
}