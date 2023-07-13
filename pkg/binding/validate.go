package binding

import "github.com/gookit/validate"

// Validator for validate bounded data
var Validator DataValidator = &stdValidator{}

type stdValidator struct{}

// Validate the struct data, if fail return error
func (sv *stdValidator) Validate(obj any) error {
	v := validate.New(obj)

	if v.Validate() {
		return nil
	}
	return v.Errors.OneError()
}

// DisableValidator for data binding
func DisableValidator() {
	Validator = nil
}

// ResetValidator for the package
func ResetValidator() {
	Validator = &stdValidator{}
}

func validating(obj any) error {
	// if Validator is nil, dont validate.
	if Validator == nil {
		return nil
	}
	return Validator.Validate(obj)
}
