package binding_test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/rux/v2/pkg/binding"
)

// ----- fixtures ---------------------------------------------------

// uploadForm exercises every file field shape DecodeFiles supports.
type uploadForm struct {
	Name   string                   `form:"name"`
	File   *multipart.FileHeader    `form:"file"`
	Docs   []*multipart.FileHeader  `form:"docs"`
	Cover  multipart.FileHeader     `form:"cover"`
	Copies []multipart.FileHeader   `form:"copies"`
	Pair   [2]*multipart.FileHeader `form:"pair"`
	Avatar avatarInfo               `form:"avatar"`
	Skip   *multipart.FileHeader    `form:"-"`
	NoTag  *multipart.FileHeader    // binds by field name
	hidden *multipart.FileHeader    // never bound
}

type avatarInfo struct {
	Path string                `form:"path"`
	Pic  *multipart.FileHeader `form:"pic"`
}

// logoBase is embedded as an unexported type: the field itself is not settable,
// but its exported file field is.
type logoBase struct {
	Logo *multipart.FileHeader `form:"logo"`
}

type embeddedForm struct {
	logoBase
	Banner *multipart.FileHeader `form:"banner"`
}

type namingForm struct {
	Tagged  *multipart.FileHeader `form:"tagged"`
	Field   *multipart.FileHeader
	Skipped *multipart.FileHeader `form:"-"`
	Comma   *multipart.FileHeader `form:"comma,omitempty"`
	private *multipart.FileHeader `form:"private"`
}

type nestedForm struct {
	Avatar avatarInfo  `form:"avatar"`
	Extra  *avatarInfo `form:"extra"`
}

type badForm struct {
	Size int `form:"size"`
}

// countValidator records how often it ran and whether the upload was already
// bound at that point.
type countValidator struct {
	calls   int
	sawFile bool
}

func (v *countValidator) Validate(obj any) error {
	v.calls++
	if f, ok := obj.(*uploadForm); ok && f.File != nil {
		v.sawFile = true
	}
	return nil
}

// ----- helpers ----------------------------------------------------

// uploadPart is one multipart part: a plain field when filename is empty,
// otherwise an uploaded file.
type uploadPart struct {
	name     string
	value    string
	filename string
}

func buildUploadBody(t *testing.T, parts ...uploadPart) (*bytes.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	for _, p := range parts {
		if p.filename == "" {
			assert.NoErr(t, mw.WriteField(p.name, p.value))
			continue
		}

		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name=%q; filename=%q`, p.name, p.filename))
		header.Set("Content-Type", "image/png")

		part, err := mw.CreatePart(header)
		assert.NoErr(t, err)
		_, err = part.Write([]byte(p.value))
		assert.NoErr(t, err)
	}

	assert.NoErr(t, mw.Close())
	return bytes.NewReader(buf.Bytes()), mw.FormDataContentType()
}

func newUploadReq(t *testing.T, parts ...uploadPart) *http.Request {
	t.Helper()
	body, ctype := buildUploadBody(t, parts...)

	req, err := http.NewRequest("POST", "/upload", body)
	assert.NoErr(t, err)
	req.Header.Set("Content-Type", ctype)
	return req
}

// parsedFiles returns the uploads of a request after parsing them.
func parsedFiles(t *testing.T, req *http.Request) map[string][]*multipart.FileHeader {
	t.Helper()
	assert.NoErr(t, req.ParseMultipartForm(binding.DefaultMaxMemory))
	return req.MultipartForm.File
}

func upload(name, filename string) uploadPart {
	return uploadPart{name: name, filename: filename, value: "png:" + filename}
}

// ----- DecodeFiles: field shapes ----------------------------------

func TestDecodeFiles_FieldShapes(t *testing.T) {
	req := newUploadReq(t,
		uploadPart{name: "name", value: "alice"},
		upload("file", "a.png"),
		upload("docs", "d1.png"),
		upload("docs", "d2.png"),
		upload("cover", "c.png"),
		upload("copies", "x1.png"),
		upload("copies", "x2.png"),
		upload("pair", "p1.png"),
		upload("pair", "p2.png"),
		upload("NoTag", "n.png"),
	)

	f := &uploadForm{}
	assert.NoErr(t, binding.DecodeFiles(parsedFiles(t, req), f, "form"))

	assert.NotNil(t, f.File)
	assert.Eq(t, "a.png", f.File.Filename)
	assert.Len(t, f.Docs, 2)
	assert.Eq(t, "d1.png", f.Docs[0].Filename)
	assert.Eq(t, "d2.png", f.Docs[1].Filename)
	assert.Eq(t, "c.png", f.Cover.Filename)
	assert.Len(t, f.Copies, 2)
	assert.Eq(t, "x1.png", f.Copies[0].Filename)
	assert.Len(t, f.Pair, 2)
	assert.Eq(t, "p2.png", f.Pair[1].Filename)
	assert.Eq(t, "n.png", f.NoTag.Filename) // field name fallback
	assert.Nil(t, f.Skip)
	assert.Nil(t, f.hidden)
}

func TestDecodeFiles_FirstUploadWinsForSingleField(t *testing.T) {
	req := newUploadReq(t,
		upload("file", "first.png"),
		upload("file", "second.png"),
	)

	f := &uploadForm{}
	assert.NoErr(t, binding.DecodeFiles(parsedFiles(t, req), f, "form"))
	assert.Eq(t, "first.png", f.File.Filename)
}

func TestDecodeFiles_FileContentIsReadable(t *testing.T) {
	req := newUploadReq(t, upload("file", "a.png"))

	f := &uploadForm{}
	assert.NoErr(t, binding.DecodeFiles(parsedFiles(t, req), f, "form"))

	fd, err := f.File.Open()
	assert.NoErr(t, err)
	defer func() { _ = fd.Close() }()

	body, err := io.ReadAll(fd)
	assert.NoErr(t, err)
	assert.Eq(t, "png:a.png", string(body))
	assert.Eq(t, int64(len(body)), f.File.Size)
}

// ----- DecodeFiles: naming and nesting ----------------------------

func TestDecodeFiles_NameResolution(t *testing.T) {
	req := newUploadReq(t,
		upload("tagged", "t.png"),
		upload("Field", "f.png"),
		upload("Skipped", "s.png"),
		upload("comma", "c.png"),
		upload("private", "p.png"),
	)

	f := &namingForm{}
	assert.NoErr(t, binding.DecodeFiles(parsedFiles(t, req), f, "form"))

	assert.Eq(t, "t.png", f.Tagged.Filename)
	assert.Eq(t, "f.png", f.Field.Filename)
	assert.Eq(t, "c.png", f.Comma.Filename) // only the first tag part is the name
	assert.Nil(t, f.Skipped)                // "-" opts out
	assert.Nil(t, f.private)                // unexported fields are never bound
}

func TestDecodeFiles_NestedStruct(t *testing.T) {
	req := newUploadReq(t,
		uploadPart{name: "avatar.path", value: "local.png"},
		upload("avatar.pic", "a.png"),
		upload("extra.pic", "e.png"),
	)

	f := &nestedForm{}
	assert.NoErr(t, binding.DecodeFiles(parsedFiles(t, req), f, "form"))

	assert.Eq(t, "a.png", f.Avatar.Pic.Filename)
	assert.Eq(t, "", f.Avatar.Path, "file decoding never touches form values")

	// a nil parent pointer is allocated on the way
	assert.NotNil(t, f.Extra)
	assert.Eq(t, "e.png", f.Extra.Pic.Filename)
}

func TestDecodeFiles_NestedStructWithoutUploadStaysNil(t *testing.T) {
	req := newUploadReq(t, upload("avatar.pic", "a.png"))

	f := &nestedForm{}
	assert.NoErr(t, binding.DecodeFiles(parsedFiles(t, req), f, "form"))

	assert.Eq(t, "a.png", f.Avatar.Pic.Filename)
	assert.Nil(t, f.Extra, "unused nested pointers must not be allocated")
}

func TestDecodeFiles_EmbeddedStruct(t *testing.T) {
	req := newUploadReq(t, upload("logo", "l.png"), upload("banner", "b.png"))

	f := &embeddedForm{}
	assert.NoErr(t, binding.DecodeFiles(parsedFiles(t, req), f, "form"))

	assert.Eq(t, "l.png", f.Logo.Filename)
	assert.Eq(t, "b.png", f.Banner.Filename)
}

// ----- DecodeFiles: edge cases ------------------------------------

func TestDecodeFiles_NoUploads(t *testing.T) {
	f := &uploadForm{}
	assert.NoErr(t, binding.DecodeFiles(nil, f, "form"))
	assert.NoErr(t, binding.DecodeFiles(map[string][]*multipart.FileHeader{}, f, "form"))
	assert.Nil(t, f.File)
}

func TestDecodeFiles_UnknownUploadIgnored(t *testing.T) {
	req := newUploadReq(t, upload("nothing-matches", "x.png"))

	f := &uploadForm{}
	assert.NoErr(t, binding.DecodeFiles(parsedFiles(t, req), f, "form"))
	assert.Nil(t, f.File)
}

func TestDecodeFiles_InvalidTarget(t *testing.T) {
	req := newUploadReq(t, upload("file", "a.png"))
	files := parsedFiles(t, req)

	assert.Err(t, binding.DecodeFiles(files, nil, "form"))
	assert.Err(t, binding.DecodeFiles(files, uploadForm{}, "form"))
	assert.ErrMsgContains(t, binding.DecodeFiles(files, nil, "form"), "non-nil pointer")
}

func TestDecodeFiles_NonStructTargetIsNoOp(t *testing.T) {
	req := newUploadReq(t, upload("file", "a.png"))

	var list []string
	assert.NoErr(t, binding.DecodeFiles(parsedFiles(t, req), &list, "form"))
	assert.Nil(t, list)
}

func TestDecodeFiles_UnsupportedFieldType(t *testing.T) {
	req := newUploadReq(t, upload("size", "a.png"))

	err := binding.DecodeFiles(parsedFiles(t, req), &badForm{}, "form")
	assert.Err(t, err)
	assert.ErrMsgContains(t, err, "unsupported file field type")
	assert.ErrMsgContains(t, err, "field Size")
}

func TestDecodeFiles_CustomTagName(t *testing.T) {
	req := newUploadReq(t, upload("file", "a.png"))

	f := &uploadForm{}
	assert.NoErr(t, binding.DecodeFiles(parsedFiles(t, req), f, "json"))
	assert.Nil(t, f.File, "no json tag matches the upload")
}

// ----- FileBinder -------------------------------------------------

func TestFileBinder_NameAndRegistry(t *testing.T) {
	assert.Eq(t, "file", binding.File.Name())
	assert.Eq(t, "file", binding.FileBinder{TagName: "form"}.Name())

	b := binding.GetBinder("file")
	assert.NotNil(t, b)
	assert.Eq(t, "file", b.Name())
}

func TestFileBinder_BindParsesRequest(t *testing.T) {
	req := newUploadReq(t, uploadPart{name: "name", value: "alice"}, upload("file", "a.png"))

	f := &uploadForm{}
	assert.NoErr(t, binding.File.Bind(req, f))
	assert.Eq(t, "a.png", f.File.Filename)
	assert.Eq(t, "", f.Name, "the file binder only fills uploads")
}

func TestFileBinder_BindValues(t *testing.T) {
	req := newUploadReq(t, upload("docs", "d1.png"), upload("docs", "d2.png"))

	f := &uploadForm{}
	assert.NoErr(t, binding.File.BindValues(parsedFiles(t, req), f))
	assert.Len(t, f.Docs, 2)
}

func TestFileBinder_BindRejectsNonMultipart(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader(`{"name":"a"}`))
	req.Header.Set("Content-Type", "application/json")

	assert.Err(t, binding.File.Bind(req, &uploadForm{}))
}

// ----- Auto / Form.Bind multipart integration ---------------------

// TestAuto_MultipartForm_BindsFiles is the case from gookit/rux#186: the file
// field must be filled by AutoBind, not just the plain form values.
func TestAuto_MultipartForm_BindsFiles(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	req := newUploadReq(t,
		uploadPart{name: "name", value: "alice"},
		upload("file", "a.png"),
		upload("docs", "d1.png"),
		upload("docs", "d2.png"),
	)

	f := &uploadForm{}
	assert.NoErr(t, binding.Auto(req, f))

	assert.Eq(t, "alice", f.Name)
	assert.NotNil(t, f.File)
	assert.Eq(t, "a.png", f.File.Filename)
	assert.Len(t, f.Docs, 2)
}

func TestAuto_MultipartForm_ValidatorSeesFileOnce(t *testing.T) {
	v := &countValidator{}
	withMockValidator(t, v)

	req := newUploadReq(t, uploadPart{name: "name", value: "alice"}, upload("file", "a.png"))

	f := &uploadForm{}
	assert.NoErr(t, binding.Auto(req, f))

	assert.Eq(t, 1, v.calls, "validation must run exactly once")
	assert.True(t, v.sawFile, "the validator must see the bound upload")
}

func TestAuto_MultipartContentTypeVariants(t *testing.T) {
	withMockValidator(t, &mockValidator{})

	// a charset parameter on top of the boundary still routes to multipart
	body, ctype := buildUploadBody(t, uploadPart{name: "name", value: "alice"})
	req, _ := http.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", ctype+"; charset=utf-8")

	f := &uploadForm{}
	assert.NoErr(t, binding.Auto(req, f))
	assert.Eq(t, "alice", f.Name)

	// a lookalike media type is not a multipart form
	req2, _ := http.NewRequest("POST", "/", strings.NewReader("x"))
	req2.Header.Set("Content-Type", "application/form-data")
	err := binding.Auto(req2, &uploadForm{})
	assert.ErrMsgContains(t, err, "cannot auto binding")

	// a malformed header keeps the historical lenient match, so the failure
	// comes from parsing the body instead of the dispatcher
	req3, _ := http.NewRequest("POST", "/", strings.NewReader("x"))
	req3.Header.Set("Content-Type", "multipart/form-data; boundary")
	err = binding.Auto(req3, &uploadForm{})
	assert.Err(t, err)
	assert.True(t, !strings.Contains(err.Error(), "cannot auto binding"), err.Error())
}

func TestFormBinder_Bind_Multipart(t *testing.T) {
	withMockValidator(t, &mockValidator{})
	req := newUploadReq(t, uploadPart{name: "name", value: "bob"}, upload("file", "a.png"))

	f := &uploadForm{}
	assert.NoErr(t, binding.Form.Bind(req, f))

	assert.Eq(t, "bob", f.Name)
	assert.Eq(t, "a.png", f.File.Filename)
}

func TestDecodeMultipart_NilFiles(t *testing.T) {
	v := &countValidator{}
	withMockValidator(t, v)

	f := &uploadForm{}
	assert.NoErr(t, binding.DecodeMultipart(map[string][]string{"name": {"carol"}}, nil, f, "form"))

	assert.Eq(t, "carol", f.Name)
	assert.Nil(t, f.File)
	assert.Eq(t, 1, v.calls)
}
