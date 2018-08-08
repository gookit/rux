package main

import (
	"fmt"
)

func main()  {
	var s1 []int

	s2 := append(s1, 2,3)
	s3 := append(s1, 4,5)

	fmt.Printf("%v %+v\n", s2, s3)
	// Output: [2 3] [4 5]

	s11 := make([]int, 5)

	s12 := append(s11, 2,3)
	s13 := append(s11, 4,5)

	fmt.Printf("%v %+v\n", s12, s13)
	// Output: [0 0 0 0 0 2 3] [0 0 0 0 0 4 5]
}
