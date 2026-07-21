// Package brdocs provides validation for Brazilian documents CPF and CNPJ.
//
// Supported operations:
//   - IsCPF: validates CPF (individual taxpayer) numbers, including check-digit verification
//   - IsCNPJ: validates CNPJ (corporate taxpayer) numbers, including the new alphanumeric format
//   - RemoveNonDigitAndLetters: sanitizes input by stripping non-alphanumeric characters
//
// Both IsCPF and IsCNPJ automatically strip punctuation and letters before validation.
package brdocs
