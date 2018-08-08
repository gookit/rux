package main

import (
	"os"
	"fmt"
)

var oldStdout *os.File

func main()  {
	discardStdout()
	fmt.Println("Hello, playground")
	restoreStdout()
	fmt.Println("Hello, playground 2")
	// Output:
	// Hello, playground 2
}

// usage:
// discardStdout()
// fmt.Println("Hello, playground")
// restoreStdout()
func discardStdout() error {
	// save old os.Stdout
	oldStdout = os.Stdout

	stdout, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err == nil {
		os.Stdout = stdout
	}

	return err
}

func restoreStdout()  {
	if oldStdout != nil {
		// close now
		os.Stdout.Close()
		// restore
		os.Stdout = oldStdout
		oldStdout = nil
	}
}