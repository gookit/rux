package binding_test

import (
	"errors"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/rux/v2/pkg/binding"
)

// mockValidator is a lightweight stand-in for an application validator.
// Tests swap it in via withMockValidator so we can exercise the hook.
type mockValidator struct {
	called bool
	last   any
	err    error
}

func (m *mockValidator) Validate(obj any) error {
	m.called = true
	m.last = obj
	return m.err
}

// withMockValidator installs m as the package-level Validator for the
// duration of t and restores the previous value on cleanup. Keeps tests
// isolated from each other and from the default stdValidator.
func withMockValidator(t *testing.T, m binding.DataValidator) {
	t.Helper()
	prev := binding.Validator
	binding.Validator = m
	t.Cleanup(func() { binding.Validator = prev })
}

// ----- BinderFunc adapter -----------------------------------------

func TestBinderFunc_AdaptsToInterface(t *testing.T) {
	called := false
	fn := binding.BinderFunc(func(r *http.Request, obj any) error {
		called = true
		return nil
	})

	// Name is fixed to "unknown" for ad-hoc functions.
	assert.Eq(t, "unknown", fn.Name())

	req, _ := http.NewRequest("GET", "/x", nil)
	assert.NoErr(t, fn.Bind(req, nil))
	assert.True(t, called)
}

// ----- Register / Remove ------------------------------------------

func TestRegister_AddsBinder(t *testing.T) {
	// Restore on cleanup so other tests don't see the custom binder.
	t.Cleanup(func() { binding.Remove("custom") })

	stub := binding.BinderFunc(func(r *http.Request, obj any) error { return nil })
	binding.Register("custom", stub)

	got := binding.GetBinder("custom")
	assert.NotNil(t, got)
	assert.Eq(t, "unknown", got.Name()) // BinderFunc always reports "unknown"
}

func TestRegister_IgnoresEmptyOrNil(t *testing.T) {
	before := len(binding.Binders)
	binding.Register("", binding.JSON) // empty name → ignored
	binding.Register("x", nil)         // nil binder → ignored
	assert.Eq(t, before, len(binding.Binders))
}

func TestRemove_MissingNameIsNoOp(t *testing.T) {
	before := len(binding.Binders)
	binding.Remove("definitely-missing")
	assert.Eq(t, before, len(binding.Binders))
}

func TestGetBinder_UnknownReturnsNil(t *testing.T) {
	assert.Nil(t, binding.GetBinder("no-such-binder"))
}

// ----- MustBind ---------------------------------------------------

func TestMustBind_Success(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	req, _ := http.NewRequest("GET", "/?age=12&name=inhere", nil)
	u := &User{}
	binding.MustBind(req, u) // must not panic
	assert.Eq(t, 12, u.Age)
	assert.Eq(t, "inhere", u.Name)
}

func TestMustBind_PanicsOnError(t *testing.T) {
	// Trigger an error path: unknown content type on a POST.
	req, _ := http.NewRequest("POST", "/", strings.NewReader("xxx"))
	req.Header.Set("Content-Type", "application/unknown")
	defer func() {
		r := recover()
		assert.NotNil(t, r, "MustBind should panic on bind error")
	}()
	binding.MustBind(req, &User{})
}

// ----- Auto content-type dispatch --------------------------------

func TestAuto_FormURLEncoded(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	req, _ := http.NewRequest("POST", "/", strings.NewReader("age=21&name=alice"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	u := &User{}
	assert.NoErr(t, binding.Auto(req, u))
	assert.Eq(t, 21, u.Age)
	assert.Eq(t, "alice", u.Name)
}

func TestAuto_MultipartForm(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	body, ctype := buildMultipart(t, map[string]string{
		"age":  "33",
		"name": "bob",
	})
	req, _ := http.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", ctype)

	u := &User{}
	assert.NoErr(t, binding.Auto(req, u))
	assert.Eq(t, 33, u.Age)
	assert.Eq(t, "bob", u.Name)
}

func TestAuto_XML(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	req, _ := http.NewRequest("POST", "/", strings.NewReader(userXML))
	req.Header.Set("Content-Type", "text/xml")

	u := &User{}
	assert.NoErr(t, binding.Auto(req, u))
	assert.Eq(t, 12, u.Age)
	assert.Eq(t, "inhere", u.Name)
}

func TestAuto_UnknownContentType_Errors(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader("x"))
	req.Header.Set("Content-Type", "application/octet-stream")
	err := binding.Auto(req, &User{})
	assert.Err(t, err)
	assert.True(t, strings.Contains(err.Error(), "cannot auto binding"))
}

// ----- Auto content-type matching ---------------------------------

// The subtype drives the dispatch, so parameters and the "+json" / "+xml"
// structured syntax suffixes are handled too.
func TestAuto_JSONSubtypeVariants(t *testing.T) {
	withMockValidator(t, &mockValidator{})

	for _, cType := range []string{
		"application/json",
		"application/json; charset=utf-8",
		"APPLICATION/JSON",
		"text/json",
		"application/json-patch",
		"application/vnd.api+json",
		"application/problem+json",
	} {
		req, _ := http.NewRequest("POST", "/", strings.NewReader(userJSON))
		req.Header.Set("Content-Type", cType)

		u := &User{}
		assert.NoErr(t, binding.Auto(req, u), cType)
		assert.Eq(t, 12, u.Age, cType)
		assert.Eq(t, "inhere", u.Name, cType)
	}
}

func TestAuto_XMLSubtypeVariants(t *testing.T) {
	withMockValidator(t, &mockValidator{})

	for _, cType := range []string{
		"text/xml",
		"application/xml",
		"application/atom+xml",
	} {
		req, _ := http.NewRequest("POST", "/", strings.NewReader(userXML))
		req.Header.Set("Content-Type", cType)

		u := &User{}
		assert.NoErr(t, binding.Auto(req, u), cType)
		assert.Eq(t, 12, u.Age, cType)
	}
}

func TestAuto_FormSubtypeVariants(t *testing.T) {
	withMockValidator(t, &mockValidator{})

	for _, cType := range []string{
		"application/x-www-form-urlencoded",
		"application/x-www-form-urlencoded; charset=UTF-8",
	} {
		req, _ := http.NewRequest("POST", "/", strings.NewReader(userQuery))
		req.Header.Set("Content-Type", cType)

		u := &User{}
		assert.NoErr(t, binding.Auto(req, u), cType)
		assert.Eq(t, 12, u.Age, cType)
	}
}

// A header that mime.ParseMediaType rejects still dispatches on the subtype, so
// the branch is reached and the real problem surfaces instead of "cannot auto
// binding".
func TestAuto_MalformedContentType(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader(userQuery))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset")

	err := binding.Auto(req, &User{})
	assert.Err(t, err)
	assert.True(t, !strings.Contains(err.Error(), "cannot auto binding"), err.Error())
}

func TestAuto_MissingContentType_Errors(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader(userQuery))

	err := binding.Auto(req, &User{})
	assert.Err(t, err)
	assert.True(t, strings.Contains(err.Error(), "cannot auto binding"))
}

// The match is on the subtype token, not on any occurrence of "json", so an
// unrelated media type never reaches the JSON binder.
func TestAuto_UnrelatedSubtypeStaysUnbound(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader(userJSON))
	req.Header.Set("Content-Type", "application/x-json")

	err := binding.Auto(req, &User{})
	assert.Err(t, err)
	assert.True(t, strings.Contains(err.Error(), "cannot auto binding"))
}

// ----- FormBinder.Bind (direct, not via Auto) ---------------------

func TestFormBinder_Bind(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	req, _ := http.NewRequest("POST", "/", strings.NewReader("age=7&name=carol"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	u := &User{}
	assert.NoErr(t, binding.Form.Bind(req, u))
	assert.Eq(t, 7, u.Age)
	assert.Eq(t, "carol", u.Name)
}

// ----- JSON / XML BindBytes + decode errors -----------------------

func TestJSONBinder_BindBytes(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	u := &User{}
	assert.NoErr(t, binding.JSON.BindBytes([]byte(userJSON), u))
	assert.Eq(t, 12, u.Age)
}

func TestJSONBinder_BindBytes_Malformed(t *testing.T) {
	err := binding.JSON.BindBytes([]byte(`{not-json`), &User{})
	assert.Err(t, err)
}

func TestXMLBinder_BindBytes(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	u := &User{}
	assert.NoErr(t, binding.XML.BindBytes([]byte(userXML), u))
	assert.Eq(t, 12, u.Age)
}

func TestXMLBinder_BindBytes_Malformed(t *testing.T) {
	err := binding.XML.BindBytes([]byte(`<not><closed>`), &User{})
	assert.Err(t, err)
}

// ----- Validator hook --------------------------------------------

func TestValidate_NilValidator_NoOp(t *testing.T) {
	withMockValidator(t, nil)
	// Validator is nil → Validate must short-circuit to nil.
	assert.NoErr(t, binding.Validate(&User{}))
}

func TestValidate_RoutesThroughInstalled(t *testing.T) {
	m := &mockValidator{}
	withMockValidator(t, m)

	in := &User{Age: 1, Name: "x"}
	assert.NoErr(t, binding.Validate(in))
	assert.True(t, m.called)
	assert.Same(t, in, m.last)
}

func TestValidate_BubblesErrorFromValidator(t *testing.T) {
	withMockValidator(t, &mockValidator{err: errors.New("invalid age")})
	err := binding.Validate(&User{})
	assert.Err(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid age"))
}

func TestDisableValidator_AndReset(t *testing.T) {
	// Save the package-level Validator manually: this test exercises the
	// package default instead of the mock helper.
	prev := binding.Validator
	t.Cleanup(func() { binding.Validator = prev })

	binding.Validator = &mockValidator{}
	assert.NotNil(t, binding.Validator)

	binding.ResetValidator()
	assert.Nil(t, binding.Validator)
	assert.NoErr(t, binding.Validate(&User{}))
}

func TestResetValidator_DefaultsToNoValidation(t *testing.T) {
	prev := binding.Validator
	t.Cleanup(func() { binding.Validator = prev })

	binding.ResetValidator()
	assert.Nil(t, binding.Validator)
	assert.NoErr(t, binding.Validate(&User{}))
}

// Binder integration: every decode path ends in Validate(), so a mock
// validator that returns an error should bubble up through the binder.
func TestBinder_DecodeChain_HitsValidator(t *testing.T) {
	m := &mockValidator{err: errors.New("nope")}
	withMockValidator(t, m)

	err := binding.JSON.BindBytes([]byte(userJSON), &User{})
	assert.Err(t, err)
	assert.True(t, m.called, "JSON binder should call Validator after decode")
	assert.True(t, strings.Contains(err.Error(), "nope"))
}

// ----- Request-aware validator hook -------------------------------

// reqValidator implements RequestValidator and records which entry point ran.
type reqValidator struct {
	plainCalls int
	reqCalls   int
	lastReq    *http.Request
	lastObj    any
	fileName   string
	err        error
}

func (v *reqValidator) Validate(obj any) error {
	v.plainCalls++
	v.lastObj = obj
	return v.err
}

func (v *reqValidator) ValidateRequest(r *http.Request, obj any) error {
	v.reqCalls++
	v.lastReq = r
	v.lastObj = obj
	if f, ok := obj.(*uploadForm); ok && f.File != nil {
		v.fileName = f.File.Filename
	}
	return v.err
}

// compile-time proof that one adapter can serve both interfaces
var _ binding.RequestValidator = (*reqValidator)(nil)

func TestValidateRequest_PrefersRequestValidator(t *testing.T) {
	v := &reqValidator{}
	withMockValidator(t, v)

	req, _ := http.NewRequest("GET", "/?age=1", nil)
	u := &User{Age: 1}

	assert.NoErr(t, binding.ValidateRequest(req, u))
	assert.Eq(t, 1, v.reqCalls)
	assert.Eq(t, 0, v.plainCalls)
	assert.Eq(t, req, v.lastReq)
	assert.Eq(t, u, v.lastObj)
}

func TestValidateRequest_NilRequestUsesValidate(t *testing.T) {
	v := &reqValidator{}
	withMockValidator(t, v)

	u := &User{}
	assert.NoErr(t, binding.ValidateRequest(nil, u))
	assert.Eq(t, 1, v.plainCalls)
	assert.Eq(t, 0, v.reqCalls)
}

// A validator that only implements DataValidator keeps working.
func TestValidateRequest_PlainValidatorStaysSupported(t *testing.T) {
	m := &mockValidator{}
	withMockValidator(t, m)

	req, _ := http.NewRequest("GET", "/", nil)
	u := &User{}
	assert.NoErr(t, binding.ValidateRequest(req, u))
	assert.True(t, m.called)
	assert.Eq(t, u, m.last)
}

func TestValidateRequest_NilValidatorIsNoOp(t *testing.T) {
	withMockValidator(t, nil)

	req, _ := http.NewRequest("GET", "/", nil)
	assert.NoErr(t, binding.ValidateRequest(req, &User{}))
}

func TestValidateRequest_BubblesError(t *testing.T) {
	v := &reqValidator{err: errors.New("bad upload")}
	withMockValidator(t, v)

	req, _ := http.NewRequest("GET", "/", nil)
	err := binding.ValidateRequest(req, &User{})
	assert.ErrMsgContains(t, err, "bad upload")
	assert.Eq(t, 1, v.reqCalls)
}

// Every binder entry point that holds the request hands it over.
func TestBinders_HandRequestToValidator(t *testing.T) {
	newQueryReq := func() *http.Request {
		req, _ := http.NewRequest("GET", "/?"+userQuery, nil)
		return req
	}
	newFormReq := func() *http.Request {
		req, _ := http.NewRequest("POST", "/", strings.NewReader(userQuery))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}
	newMultipartReq := func() *http.Request {
		return newUploadReq(t, uploadPart{name: "name", value: "inhere"})
	}
	newJSONReq := func() *http.Request {
		req, _ := http.NewRequest("POST", "/", strings.NewReader(userJSON))
		req.Header.Set("Content-Type", "application/json")
		return req
	}
	newXMLReq := func() *http.Request {
		req, _ := http.NewRequest("POST", "/", strings.NewReader(userXML))
		req.Header.Set("Content-Type", "text/xml")
		return req
	}
	newHeaderReq := func() *http.Request {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("age", "12")
		req.Header.Set("name", "inhere")
		return req
	}

	cases := []struct {
		name string
		req  func() *http.Request
		bind func(*http.Request, any) error
	}{
		{"auto/query", newQueryReq, binding.Auto},
		{"auto/form", newFormReq, binding.Auto},
		{"auto/multipart", newMultipartReq, binding.Auto},
		{"auto/json", newJSONReq, binding.Auto},
		{"auto/xml", newXMLReq, binding.Auto},
		{"form", newFormReq, binding.Form.Bind},
		{"query", newQueryReq, binding.Query.Bind},
		{"header", newHeaderReq, binding.Header.Bind},
		{"json", newJSONReq, binding.JSON.Bind},
		{"xml", newXMLReq, binding.XML.Bind},
	}

	for _, c := range cases {
		v := &reqValidator{}
		withMockValidator(t, v)

		req := c.req()
		assert.NoErr(t, c.bind(req, &User{}), c.name)
		assert.Eq(t, 1, v.reqCalls, c.name)
		assert.Eq(t, 0, v.plainCalls, c.name)
		assert.Eq(t, req, v.lastReq, c.name)
	}
}

// The binder helpers without a request keep using the plain path.
func TestBindValues_UsesPlainValidate(t *testing.T) {
	v := &reqValidator{}
	withMockValidator(t, v)

	assert.NoErr(t, binding.Form.BindValues(url.Values{"age": {"12"}}, &User{}))
	assert.NoErr(t, binding.Query.BindValues(url.Values{"age": {"12"}}, &User{}))
	assert.NoErr(t, binding.JSON.BindBytes([]byte(userJSON), &User{}))
	assert.NoErr(t, binding.XML.BindBytes([]byte(userXML), &User{}))
	assert.NoErr(t, binding.DecodeMultipart(map[string][]string{"age": {"12"}}, nil, &User{}, "form"))

	assert.Eq(t, 0, v.reqCalls)
	assert.Eq(t, 5, v.plainCalls)
}

// The point of the hook: on the multipart path the request and the bound file
// are both available when validation runs.
func TestAuto_Multipart_RequestValidatorSeesFile(t *testing.T) {
	v := &reqValidator{}
	withMockValidator(t, v)

	req := newUploadReq(t, uploadPart{name: "name", value: "alice"}, upload("file", "a.png"))
	f := &uploadForm{}
	assert.NoErr(t, binding.Auto(req, f))

	assert.Eq(t, 1, v.reqCalls)
	assert.Eq(t, req, v.lastReq)
	assert.Eq(t, "a.png", v.fileName)
	assert.NotNil(t, req.MultipartForm)
}

// ----- helpers ----------------------------------------------------

// buildMultipart serializes the given fields as multipart/form-data and
// returns the body reader + matching Content-Type.
func buildMultipart(t *testing.T, fields map[string]string) (*strings.Reader, string) {
	t.Helper()
	var sb strings.Builder
	mw := multipart.NewWriter(&sb)
	for k, v := range fields {
		assert.NoErr(t, mw.WriteField(k, v))
	}
	assert.NoErr(t, mw.Close())
	return strings.NewReader(sb.String()), mw.FormDataContentType()
}
