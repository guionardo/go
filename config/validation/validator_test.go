package validation_test

import (
	"errors"
	"testing"

	"github.com/guionardo/go/config/validation"
	"github.com/stretchr/testify/require"
)

type (
	ValidatorStruct struct {
		Int int `validate:"required"`
	}

	SubStruct struct {
		Name string `validate:"required"`
	}
	ValidatorStructWithRequiredStructEnabled struct {
		Int       int `validate:"required"`
		SubStruct SubStruct
	}
)

func (v ValidatorStruct) Validate() error {
	if v.Int == 0 {
		return errors.New("int is required")
	}

	return nil
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("Validate should return an error if the struct is not valid", func(t *testing.T) {
		t.Parallel()

		v := struct {
			Int int `validate:"required"`
		}{}
		err := validation.Validate(v)
		require.Error(t, err)
	})

	t.Run("Validate should return nil if the struct is valid", func(t *testing.T) {
		t.Parallel()

		v := struct {
			Int int `validate:"required"`
		}{Int: 1}
		require.NoError(t, validation.Validate(v))
	})

	t.Run("Validate should validates the structs that implement the Validator interface", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, validation.Validate(ValidatorStruct{Int: 1}))
	})

	t.Run("Validate should validates the structs with required struct enabled", func(t *testing.T) {
		t.Parallel()

		s := ValidatorStructWithRequiredStructEnabled{
			Int:       1,
			SubStruct: SubStruct{},
		}

		require.Error(t, validation.Validate(s))
	})
}
