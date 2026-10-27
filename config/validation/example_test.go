package validation_test

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/guionardo/go/config/validation"
)

func ExampleValidate() {
	type serverConfig struct {
		Host string `validate:"required"`
		Port int    `validate:"required,gt=0"`
	}

	fmt.Println(validation.Validate(serverConfig{Host: "localhost", Port: 8080}))

	fieldErrors, ok := errors.AsType[validator.ValidationErrors](validation.Validate(serverConfig{}))
	if ok {
		for _, fieldError := range fieldErrors {
			fmt.Printf("%s failed on %s\n", fieldError.Field(), fieldError.Tag())
		}
	}

	// Output:
	// <nil>
	// Host failed on required
	// Port failed on required
}
