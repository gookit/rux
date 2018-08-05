package sux

import (
	"fmt"
)

func Example() {
	r := New()
	r.GET("/", func(ctx *Context) {
		ctx.WriteString("hello")
	})

	ret := r.Match("GET", "/")
	fmt.Print(ret.Status)

	// run http server
	// r.RunServe(":8080")

	// Output:
	// 1
}
