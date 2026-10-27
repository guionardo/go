// Package validation provides struct validation.
//
// Defines a Validator interface for self-validating types and wraps
// go-playground/validator/v10 for struct tag validation.
//
// Types implementing the Validator interface replace struct-tag validation
// rather than extend it: Validate returns v.Validate() immediately, so the
// validator/v10 tags on such a type are never evaluated. Values that do not
// implement Validator go through validator/v10's Struct.
//
// Usage:
//
//	err := validation.Validate(myStruct)
package validation
