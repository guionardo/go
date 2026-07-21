// Package validation provides struct validation.
//
// Defines a Validator interface for self-validating types and wraps
// go-playground/validator/v10 for struct tag validation.
//
// Usage:
//
//	err := validation.Validate(myStruct)
//
// Types implementing the Validator interface can provide custom
// validation logic beyond struct tags.
package validation
