package render

import (
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	r := New()

	fmt.Printf("%#v\n", r)
}

func TestNewHTTPRenderer(t *testing.T) {
	r := NewHTTPRenderer()
	fmt.Printf("%+v\n", r)
}
