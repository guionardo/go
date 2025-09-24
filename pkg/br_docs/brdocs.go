package brdocs

import (
	"bytes"
	"regexp"
	"strconv"
)

var (
	cpfRegexp  = regexp.MustCompile(`^\d{11}$`)
	cnpjRegexp = regexp.MustCompile(`^[0-9A-Z]{12}\d{2}$`)
)

// IsCPF verifies if the given string is a valid CPF document.
// Punctuation will be automatically removed
func IsCPF(doc string) bool {
	const (
		size = 9
		pos  = 10
	)

	return isCadastro(doc, cpfRegexp, size, pos)
}

// IsCNPJ verifies if the given string is a valid CNPJ document.
// Punctuation will be automatically removed. Rules for new alfanumeric format.
func IsCNPJ(doc string) bool {
	const (
		size = 12
		pos  = 5
	)

	return isCadastro(doc, cnpjRegexp, size, pos)
}

// isCadastro generates the digits for a given CPF or CNPJ and compares it with
// the original digits.
func isCadastro(
	doc string,
	pattern *regexp.Regexp,
	size int,
	position int,
) bool {
	RemoveNonDigitAndLetters(&doc)

	if !pattern.MatchString(doc) {
		return false
	}

	// Invalidates documents with all digits equal.
	if allEq(doc) {
		return false
	}

	d := doc[:size]
	digit := calcCadastroDigit(d, position)

	d = d + digit
	digit = calcCadastroDigit(d, position+1)

	return doc == d+digit
}

// calcCadastroDigit calculates the next digit for the given document.
func calcCadastroDigit(doc string, position int) string {
	var sum int
	for _, r := range doc {

		sum += int(r-'0') * position
		position--

		if position < 2 {
			position = 9
		}
	}

	sum %= 11
	if sum < 2 {
		return "0"
	}

	return strconv.Itoa(11 - sum)
}

// RemoveNonDigitAndLetters updates the value, keeping only 0-9, A-Z characters
func RemoveNonDigitAndLetters(value *string) {
	buf := bytes.NewBufferString("")
	for _, r := range *value {
		if ('0' <= r && r <= '9') || ('A' <= r && r <= 'Z') {
			buf.WriteRune(r)
		}
	}
	*value = buf.String()
}

// allEq checks if every rune in a given string is equal.
func allEq(doc string) bool {
	base := doc[0]
	for i := 1; i < len(doc); i++ {
		if base != doc[i] {
			return false
		}
	}

	return true
}
