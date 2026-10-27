package brdocs_test

import (
	"fmt"

	brdocs "github.com/guionardo/go/br_docs"
)

// ExampleIsCPF shows that punctuation is stripped before the check digits are
// verified, and that a document with every character equal is rejected.
func ExampleIsCPF() {
	fmt.Println(brdocs.IsCPF("355.085.990-29"))
	fmt.Println(brdocs.IsCPF("18272187035"))
	fmt.Println(brdocs.IsCPF("00000000000"))
	fmt.Println(brdocs.IsCPF("182721870359"))

	// Output:
	// true
	// true
	// false
	// false
}

// ExampleIsCNPJ shows both accepted CNPJ formats: the classic numeric one and
// the alphanumeric one.
func ExampleIsCNPJ() {
	fmt.Println(brdocs.IsCNPJ("23.192.973/0001-75"))
	fmt.Println(brdocs.IsCNPJ("09892025000111"))
	fmt.Println(brdocs.IsCNPJ("DR.50S.66W/KSIE-50"))
	fmt.Println(brdocs.IsCNPJ("23.192.973/0001-74"))

	// Output:
	// true
	// true
	// true
	// false
}
