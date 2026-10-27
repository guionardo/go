# br_docs

Validation for the Brazilian documents CPF (individual taxpayer) and CNPJ (corporate
taxpayer), including the alphanumeric CNPJ format. Reach for it when you need to accept
or reject a document number, with punctuation removed and check digits verified.

The import path is `br_docs` but the package name is `brdocs`.

Part of [`github.com/guionardo/go`](../README.md) · API reference: [pkg.go.dev](https://pkg.go.dev/github.com/guionardo/go/br_docs)

## Install

```bash
go get github.com/guionardo/go/br_docs
```

## Usage

```go
import "github.com/guionardo/go/br_docs"

brdocs.IsCPF("355.085.990-29")      // true
brdocs.IsCPF("00000000000")         // false — all digits equal
brdocs.IsCNPJ("23.192.973/0001-75") // true
brdocs.IsCNPJ("DR.50S.66W/KSIE-50") // true — alphanumeric CNPJ

value := "355.085.990-29"
brdocs.RemoveNonDigitAndLetters(&value)
// value == "35508599029"
```

## API

```go
func IsCPF(doc string) bool
func IsCNPJ(doc string) bool
func RemoveNonDigitAndLetters(value *string)
```

## Behaviour

- `IsCPF` and `IsCNPJ` sanitise the input first — punctuation and letters are stripped —
  then verify the check digits.
- Documents where every character is the same (`00000000000`) are rejected.
- `IsCPF` requires exactly 11 digits after sanitising.
- `IsCNPJ` accepts the alphanumeric format: 12 characters in `0-9A-Z` followed by 2
  numeric check digits.
- `RemoveNonDigitAndLetters` keeps only `0-9` and `A-Z`, mutating the string through its
  pointer. Lowercase letters are **discarded, not uppercased** — uppercase the input
  before calling it if you accept lowercase alphanumeric CNPJs.
