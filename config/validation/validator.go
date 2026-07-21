package validation

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

// Validator is the interface implemented by types that can self-validate.
// Types implementing this interface receive custom validation logic
// before the standard struct validation runs.
type Validator interface {
	Validate() error
}

var (
	validateOnce sync.Once
	validate     *validator.Validate
)

func getValidator() *validator.Validate {
	validateOnce.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
	})

	return validate
}

// Validate tries to validate a struct using validator_v10 or the inner Validate method, if declared
// If the struct implements a method Validate() error, it will be used. Otherwise, the validator/v10
// validate.Struct(v) will be used
func Validate(v any) error {
	if validator, ok := v.(Validator); ok {
		return validator.Validate()
	}

	return getValidator().Struct(v)
}
