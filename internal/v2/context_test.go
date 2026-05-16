package v2

import (
	"net/http/httptest"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestContext_Param_ReadsFromInlineParams(t *testing.T) {
	c := &Context{}
	c.params.append("id", "42")
	assert.Eq(t, "42", c.Param("id"))
	assert.Eq(t, "", c.Param("missing"))
}

func TestContext_SetGet_LazyMap(t *testing.T) {
	c := &Context{}
	_, ok := c.Get("missing")
	assert.False(t, ok)
	assert.Nil(t, c.data, "data map should not be allocated until Set is called")

	c.Set("k", 42)
	v, ok := c.Get("k")
	assert.True(t, ok)
	assert.Eq(t, 42, v)
}

func TestContext_Init_AssignsRequestAndResponse(t *testing.T) {
	c := &Context{}
	req := httptest.NewRequest("GET", "/x", nil)
	w := httptest.NewRecorder()
	c.Init(w, req)
	assert.Same(t, req, c.Req)
	assert.NotNil(t, c.Resp)
}
