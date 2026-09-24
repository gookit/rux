package binding

import "net/http"

// Validator validates bound data. It is nil by default; applications can
// install their own validator adapter when validation is needed.
//
// Implement RequestValidator instead when a rule needs the request itself.
var Validator DataValidator

// DisableValidator for data binding
func DisableValidator() {
	Validator = nil
}

// ResetValidator for the package
func ResetValidator() {
	Validator = nil
}

// Validate bounded data
func Validate(obj any) error {
	// if Validator is nil, dont validate.
	if Validator == nil {
		return nil
	}
	return Validator.Validate(obj)
}

// ValidateRequest validates obj with the request at hand.
//
// A validator that implements RequestValidator receives r, so it can look at
// the raw request as well as the bound struct. Any other validator is called
// through Validate, and a nil request always takes that path.
func ValidateRequest(r *http.Request, obj any) error {
	if Validator == nil {
		return nil
	}
	if r == nil {
		return Validator.Validate(obj)
	}
	if rv, ok := Validator.(RequestValidator); ok {
		return rv.ValidateRequest(r, obj)
	}
	return Validator.Validate(obj)
}
