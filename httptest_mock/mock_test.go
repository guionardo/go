package httptestmock

import (
	"errors"
)

type badMarshaler struct{}

func (b badMarshaler) MarshalJSON() ([]byte, error) { // nocover
	return nil, errors.New("marshal error")
}
