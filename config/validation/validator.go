// Package validation provides struct validation using go-playground/validator
// and a custom Validator interface for per-type validation logic.
package validation

import "github.com/go-playground/validator/v10"

// Validator is the interface implemented by types that can self-validate.
// Types implementing this interface receive custom validation logic
// before the standard struct validation runs.
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
