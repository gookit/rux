package binding_test

import (
	"net/http"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/rux/binding"
)

func TestBinder_Name(t *testing.T) {
	is := assert.New(t)
	for name, binder := range binding.Binders {
		is.Eq(name, binder.Name())
	}
}

func TestGetBinder(t *testing.T) {
	is := assert.New(t)
	b := binding.GetBinder("query")

	req, err := http.NewRequest("GET", "/?"+userQuery, nil)
	is.NoErr(err)

	u := &User{}
	err = b.Bind(req, u)
	testBoundedUserIsOK(is, err, u)
}
