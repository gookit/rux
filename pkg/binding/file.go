package binding

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
)

// FileBinder binds uploaded files (multipart/form-data) into struct fields.
//
// Field names are resolved like form values: the tag value (first comma part)
// when present, else the field name. Nested structs use a dotted path
// ("avatar.path"), embedded structs are flattened and "-" skips a field.
//
// Supported field types are multipart.FileHeader, *multipart.FileHeader and
// slices or arrays of either. A single-value field receives the first upload
// for its name, a slice receives all of them.
type FileBinder struct {
	TagName string
}

// File is the built-in binder for uploaded files. It shares the "form" tag
// with Form, so `form:"avatar"` names the field for both.
var File = FileBinder{FormTagName}

// Name get name
func (FileBinder) Name() string {
	return "file"
}

// Bind uploaded files from http.Request into ptr.
//
// It parses the multipart body itself, so it works on a request that no other
// binder has touched yet.
func (b FileBinder) Bind(r *http.Request, ptr any) error {
	if err := r.ParseMultipartForm(DefaultMaxMemory); err != nil {
		return err
	}

	return DecodeFiles(multipartFiles(r), ptr, b.TagName)
}

// BindValues binds an already parsed multipart file map into ptr.
func (b FileBinder) BindValues(files map[string][]*multipart.FileHeader, ptr any) error {
	return DecodeFiles(files, ptr, b.TagName)
}

// multipartFiles returns the parsed uploads of r, tolerating a request whose
// multipart body was never parsed.
func multipartFiles(r *http.Request) map[string][]*multipart.FileHeader {
	if r.MultipartForm == nil {
		return nil
	}
	return r.MultipartForm.File
}

// DecodeFiles binds uploaded files into the struct pointer ptr.
//
// It is the file counterpart of DecodeUrlValues: fields without an upload keep
// their zero value and an upload without a matching field is ignored. A target
// that does not point to a struct is a no-op, so DecodeMultipart can hand over
// any target the value decoder accepts.
func DecodeFiles(files map[string][]*multipart.FileHeader, ptr any, tagName string) error {
	rv := reflect.ValueOf(ptr)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("binding: file target must be a non-nil pointer")
	}
	if rv.Elem().Kind() != reflect.Struct || len(files) == 0 {
		return nil
	}

	return decodeFiles(files, rv.Elem(), tagName, "")
}

var (
	fileHeaderType  = reflect.TypeOf(multipart.FileHeader{})
	fileHeaderPtrTy = reflect.TypeOf((*multipart.FileHeader)(nil))
)

// decodeFiles walks the struct and fills every file field it can find. prefix
// carries the dotted path of the enclosing structs.
func decodeFiles(files map[string][]*multipart.FileHeader, rv reflect.Value, tagName, prefix string) error {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		fd, fv := rt.Field(i), rv.Field(i)

		if fd.Anonymous {
			// embedded struct: flatten it, its own fields carry the names.
			// An embedded unexported type is not settable itself, but its
			// exported fields are.
			if sub, ok := nestedStructValue(fv); ok {
				if err := decodeFiles(files, sub, tagName, prefix); err != nil {
					return err
				}
				continue
			}
			// embedded file header: bind it by its field name
			if !isFileField(fd.Type) || !fv.CanSet() {
				continue
			}
		} else if !fv.CanSet() {
			// unexported, or only reachable through an unexported field
			continue
		}

		tagVal := tagNameOf(fd.Tag, tagName)
		if tagVal == "-" {
			continue
		}
		name := tagVal
		if name == "" {
			name = fd.Name
		}
		if prefix != "" {
			name = prefix + "." + name
		}

		// named nested struct: descend with a dotted path
		if isNestedStruct(fd.Type) {
			// only allocate and descend when an upload uses this path
			if hasFilePrefix(files, name) {
				if sub, ok := nestedStructValue(fv); ok {
					if err := decodeFiles(files, sub, tagName, name); err != nil {
						return err
					}
				}
			}
			continue
		}

		fhs := files[name]
		if len(fhs) == 0 {
			continue
		}
		if err := setFileValue(fv, fhs); err != nil {
			return fmt.Errorf("binding: field %s: %w", fd.Name, err)
		}
	}

	return nil
}

// isNestedStruct reports whether t is a struct, or a pointer to one, that the
// walk should descend into. File fields are structs too, but they are leaves.
func isNestedStruct(t reflect.Type) bool {
	if isFileField(t) {
		return false
	}

	switch t.Kind() {
	case reflect.Struct:
		return true
	case reflect.Ptr:
		return t.Elem().Kind() == reflect.Struct
	}

	return false
}

// hasFilePrefix reports whether any upload sits under path, so "meta.pic"
// counts for path "meta".
func hasFilePrefix(files map[string][]*multipart.FileHeader, path string) bool {
	prefix := path + "."
	for name := range files {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// nestedStructValue returns the struct value to descend into for fv. It
// allocates a nil pointer on the way, so nested file fields bind even when the
// parent struct was left unset.
func nestedStructValue(fv reflect.Value) (reflect.Value, bool) {
	t := fv.Type()
	if isFileField(t) {
		return fv, false
	}

	switch t.Kind() {
	case reflect.Struct:
		return fv, true
	case reflect.Ptr:
		if t.Elem().Kind() != reflect.Struct {
			return fv, false
		}
		if fv.IsNil() {
			if !fv.CanSet() {
				return fv, false
			}
			fv.Set(reflect.New(t.Elem()))
		}
		return fv.Elem(), true
	}

	return fv, false
}

// isFileField reports whether t is a type DecodeFiles can fill.
func isFileField(t reflect.Type) bool {
	return t == fileHeaderType || t == fileHeaderPtrTy
}

// tagNameOf returns the field name part of a struct tag, formam style.
func tagNameOf(tag reflect.StructTag, tagName string) string {
	v := tag.Get(tagName)
	if p := strings.Index(v, ","); p != -1 {
		return v[:p]
	}
	return v
}

// setFileValue writes the uploads of one field name into fv.
func setFileValue(fv reflect.Value, fhs []*multipart.FileHeader) error {
	if !fv.CanSet() {
		return fmt.Errorf("cannot set %s", fv.Type())
	}

	switch fv.Kind() {
	case reflect.Ptr: // *multipart.FileHeader, first upload wins
		if fv.Type() != fileHeaderPtrTy {
			return unsupportedFileType(fv.Type())
		}
		fv.Set(reflect.ValueOf(fhs[0]))
	case reflect.Struct: // multipart.FileHeader, first upload wins
		if fv.Type() != fileHeaderType {
			return unsupportedFileType(fv.Type())
		}
		fv.Set(reflect.ValueOf(*fhs[0]))
	case reflect.Slice:
		out := reflect.MakeSlice(fv.Type(), len(fhs), len(fhs))
		for i, fh := range fhs {
			switch fv.Type().Elem() {
			case fileHeaderPtrTy:
				out.Index(i).Set(reflect.ValueOf(fh))
			case fileHeaderType:
				out.Index(i).Set(reflect.ValueOf(*fh))
			default:
				return unsupportedFileType(fv.Type())
			}
		}
		fv.Set(out)
	case reflect.Array:
		if !isFileField(fv.Type().Elem()) {
			return unsupportedFileType(fv.Type())
		}
		n := fv.Len()
		if len(fhs) < n {
			n = len(fhs)
		}
		for i := 0; i < n; i++ {
			if fv.Type().Elem() == fileHeaderPtrTy {
				fv.Index(i).Set(reflect.ValueOf(fhs[i]))
			} else {
				fv.Index(i).Set(reflect.ValueOf(*fhs[i]))
			}
		}
	default:
		return unsupportedFileType(fv.Type())
	}

	return nil
}

func unsupportedFileType(t reflect.Type) error {
	return fmt.Errorf("unsupported file field type %s, want multipart.FileHeader or a slice of it", t)
}
