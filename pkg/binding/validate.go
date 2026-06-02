package binding

// Validator validates bound data. It is nil by default; applications can
// install their own validator adapter when validation is needed.
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
