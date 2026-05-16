package v2

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestBuildRequestURL_BasicSubstitution(t *testing.T) {
	u := NewBuildRequestURL().Path("/users/{id}").Build(M{"{id}": 42})
	assert.Eq(t, "/users/42", u.Path)
}

func TestBuildRequestURL_QueryString(t *testing.T) {
	u := NewBuildRequestURL().Path("/x").Build(M{"q": "hello"})
	assert.Eq(t, "/x", u.Path)
	assert.Eq(t, "hello", u.Query().Get("q"))
}
