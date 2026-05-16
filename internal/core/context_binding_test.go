package core

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestContext_BindForm(t *testing.T) {
	type form struct {
		Name string `form:"name"`
	}
	r := New()
	r.POST("/x", func(c *Context) {
		var f form
		if err := c.Bind(&f); err != nil {
			c.AbortWithStatus(400)
			return
		}
		_, _ = c.Resp.Write([]byte("name=" + f.Name))
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/x", strings.NewReader("name=alice"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	body, _ := io.ReadAll(w.Body)
	assert.Eq(t, "name=alice", string(body))
}
