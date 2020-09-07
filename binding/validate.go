package binding

import "github.com/gookit/validate"

// Validator for validate bounded data
var Validator DataValidator = &stdValidator{}

type stdValidator struct{}

// Validate the struct data, if fail return error
func (sv *stdValidator) Validate(obj interface{}) error {
	v := validate.New(obj)
	v.Validate()

	return v.Errors
}

func validating(obj interface{}) error {
	if Validator == nil {
		return nil
	}

	return Validator.Validate(obj)
}