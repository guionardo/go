package validation

import "github.com/go-playground/validator/v10"

type Validator interface {
	Validate() error
}

var validate = validator.New(validator.WithRequiredStructEnabled())

// Validate tries to validate a struct using validator_v10 or the inner Validate method, if declared
// If the struct implements a method Validate() error, it will be used. Otherwise, the validator/v10
// validate.Struct(v) will be used
func Validate(v any) error {
	if validator, ok := v.(Validator); ok {
		return validator.Validate()
	}

	return validate.Struct(v)
}
