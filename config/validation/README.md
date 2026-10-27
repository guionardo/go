# validation

Struct validation through `go-playground/validator` tags, or through a type's own
`Validate()` method.

Part of [`config`](../README.md) in [`github.com/guionardo/go`](../../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/config/validation)

## Install

```bash
go get github.com/guionardo/go/config/validation
```

## Usage

```go
type AppConfig struct {
	Host string `validate:"required"`
	Port int    `validate:"required,gt=0"`
}

if err := validation.Validate(AppConfig{Host: "localhost", Port: 8080}); err != nil {
	log.Fatal(err)
}
```

The error from a tag failure is `validator.ValidationErrors`, so
`errors.As(err, &fieldErrors)` gets you the per-field detail. See
[`example_test.go`](example_test.go) for both branches.

## Behaviour

**A type with a `Validate() error` method is validated by that method alone.** The
interface check comes first and returns immediately:

```go
func Validate(v any) error {
	if validator, ok := v.(Validator); ok {
		return validator.Validate()
	}
	return getValidator().Struct(v)
}
```

`validator.Struct` never runs for such a type, so its `validate` tags are not checked
at all — a field tagged `validate:"required"` passes if your `Validate()` returns `nil`.
Put every rule you need inside `Validate()`, and return `validator.ValidationErrors`
yourself if callers rely on that type.

**Receiver kind decides which branch you get.** With `func (c *Config) Validate() error`,
a `Config` value does not implement the interface and falls through to tag validation;
pass `&cfg` to reach the method.

**Tag validation** goes through one shared `*validator.Validate`, built once with
`validator.WithRequiredStructEnabled()`, so `validate:"required"` on a nested struct
field is enforced — a zero-valued nested struct fails even though that field carries no
tag of its own.

`Validate` accepts `any` and returns a validator error (not a panic) for non-struct
input. A `nil` pointer that implements `Validator` is passed straight to the method, so
guard the receiver if the method dereferences.
