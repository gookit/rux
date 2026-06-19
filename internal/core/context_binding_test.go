package core

import (
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/rux/v2/pkg/binding"
)

// mockValidator + withMockValidator mirror the helper from the binding
// package's external tests; redeclared here because Go test packages
// don't share helpers across modules even when they're internal_test.
type mockValidator struct {
	called bool
	err    error
}

func (m *mockValidator) Validate(any) error {
	m.called = true
	return m.err
}

func withMockValidator(t *testing.T, m binding.DataValidator) {
	t.Helper()
	prev := binding.Validator
	binding.Validator = m
	t.Cleanup(func() { binding.Validator = prev })
}

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

// ----- direct Context.* binding methods --------------------------

type bindUser struct {
	Age  int    `query:"age" form:"age" json:"age" xml:"age" header:"age"`
	Name string `query:"name" form:"name" json:"name" xml:"name" header:"name"`
}

func bindCtx(t *testing.T, method, target, body string, ctype string) *Context {
	t.Helper()
	c := &Context{}
	// Pass a typed nil io.Reader (NOT a nil *strings.Reader) when there's
	// no body; httptest.NewRequest would NPE trying to call .Len() on a
	// nil-pointer-wrapped-interface.
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, rdr)
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	c.Init(httptest.NewRecorder(), req)
	return c
}

func TestContext_ShouldBind_JSON(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	c := bindCtx(t, "POST", "/x", `{"age":12,"name":"alice"}`, "application/json")
	var u bindUser
	assert.NoErr(t, c.ShouldBind(&u, binding.JSON))
	assert.Eq(t, 12, u.Age)
	assert.Eq(t, "alice", u.Name)
}

func TestContext_MustBind_Success(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	c := bindCtx(t, "POST", "/x", `{"age":7,"name":"bob"}`, "application/json")
	var u bindUser
	c.MustBind(&u, binding.JSON) // must not panic
	assert.Eq(t, 7, u.Age)
}

func TestContext_MustBind_PanicsOnErr(t *testing.T) {
	c := bindCtx(t, "POST", "/x", `{not-json`, "application/json")
	defer func() {
		r := recover()
		assert.NotNil(t, r, "MustBind should panic on bind error")
	}()
	c.MustBind(&bindUser{}, binding.JSON)
}

func TestContext_AutoBind_QueryStringGET(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	c := bindCtx(t, "GET", "/x?age=21&name=carol", "", "")
	var u bindUser
	assert.NoErr(t, c.AutoBind(&u))
	assert.Eq(t, 21, u.Age)
}

func TestContext_BindAlias_AutoRoutes(t *testing.T) {
	// Context.Bind is an alias for AutoBind — same behavior.
	withMockValidator(t, &mockValidator{})
	c := bindCtx(t, "POST", "/x", `{"age":1,"name":"x"}`, "application/json")
	var u bindUser
	assert.NoErr(t, c.Bind(&u))
	assert.Eq(t, 1, u.Age)
}

func TestContext_Validate_RoutesThroughValidator(t *testing.T) {
	m := &mockValidator{}
	withMockValidator(t, m)
	c := bindCtx(t, "GET", "/x", "", "")
	assert.NoErr(t, c.Validate(&bindUser{}))
	assert.True(t, m.called)
}

func TestContext_Validate_BubblesError(t *testing.T) {
	withMockValidator(t, &mockValidator{err: errors.New("invalid")})
	c := bindCtx(t, "GET", "/x", "", "")
	assert.Err(t, c.Validate(&bindUser{}))
}

func TestContext_BindForm_Direct(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	c := bindCtx(t, "POST", "/x", "age=4&name=dan", "application/x-www-form-urlencoded")
	var u bindUser
	assert.NoErr(t, c.BindForm(&u))
	assert.Eq(t, 4, u.Age)
	assert.Eq(t, "dan", u.Name)
}

func TestContext_BindJSON(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	c := bindCtx(t, "POST", "/x", `{"age":9,"name":"eve"}`, "application/json")
	var u bindUser
	assert.NoErr(t, c.BindJSON(&u))
	assert.Eq(t, 9, u.Age)
}

func TestContext_BindXML(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	xml := `<?xml version="1.0"?><doc><age>3</age><name>frank</name></doc>`
	c := bindCtx(t, "POST", "/x", xml, "text/xml")
	type doc struct {
		Age  int    `xml:"age"`
		Name string `xml:"name"`
	}
	var d doc
	assert.NoErr(t, c.BindXML(&d))
	assert.Eq(t, 3, d.Age)
	assert.Eq(t, "frank", d.Name)
}
