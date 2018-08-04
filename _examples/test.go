package main

import (
	"regexp"
	"fmt"
)

func main() {
	var varRegex = regexp.MustCompile(`{([^/]+)}`)

	path := `/users/{uid:\d+}/blog/{id}`

	ss := varRegex.FindAllString(path, -1)
	sss := varRegex.FindAllStringSubmatch(path, -1)

	fmt.Println(ss, sss)
}