package brdocs_test

import (
	"testing"

	brdocs "github.com/guionardo/go/pkg/br_docs"
	"github.com/stretchr/testify/assert"
)

func TestIsCPF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		doc  string
		want bool
	}{
		{"valid_cpf_with_points", "355.085.990-29", true},
		{"valid_cpf_without_points", "18272187035", true},
		{"invalid_cpf", "182721870359", false},
		{"all_equal", "00000000000", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := brdocs.IsCPF(tt.doc)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsCNPJ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		doc  string
		want bool
	}{
		{"valid_cnpj_with_points", "23.192.973/0001-75", true},
		{"valid_cnpj_without_points", "09892025000111", true},
		{"valid_cnpj_with_letters", "DR.50S.66W/KSIE-50", true},
		{"invalid_cnpj", "182721870359", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := brdocs.IsCNPJ(tt.doc)
			assert.Equal(t, tt.want, got)
		})
	}
}
