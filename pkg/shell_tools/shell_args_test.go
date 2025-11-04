package shelltools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuotedShellArgs(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		s          string
		want       QuotedShellArgs
		wantJoined string
	}{
		{
			name:       "empty",
			s:          " ",
			want:       QuotedShellArgs{},
			wantJoined: "",
		},
		{
			name:       "only spaces",
			s:          "     ",
			want:       QuotedShellArgs{},
			wantJoined: "",
		},
		{
			name:       "single word",
			s:          "hello",
			want:       QuotedShellArgs{"hello"},
			wantJoined: "hello",
		},
		{
			name:       "multiple words",
			s:          "hello world",
			want:       QuotedShellArgs{"hello", "world"},
			wantJoined: "hello world",
		},
		{
			name:       "quoted substring",
			s:          "hello \"new world\"",
			want:       QuotedShellArgs{"hello", "new world"},
			wantJoined: "hello \"new world\"",
		},
		{
			name:       "mixed quotes",
			s:          "\"say hello\" to the \"new world\"",
			want:       QuotedShellArgs{"say hello", "to", "the", "new world"},
			wantJoined: "\"say hello\" to the \"new world\"",
		},
		{
			name:       "trailing space",
			s:          "hello world ",
			want:       QuotedShellArgs{"hello", "world"},
			wantJoined: "hello world",
		},
		{
			name:       "leading space",
			s:          "  hello world",
			want:       QuotedShellArgs{"hello", "world"},
			wantJoined: "hello world",
		},
		{
			name:       "complex case",
			s:          "  \"hello world\"  this is a \"test case\"  ",
			want:       QuotedShellArgs{"hello world", "this", "is", "a", "test case"},
			wantJoined: "\"hello world\" this is a \"test case\"",
		},
		{
			name:       "single quotes",
			s:          "it's a 'test case'",
			want:       QuotedShellArgs{"it's", "a", "test case"},
			wantJoined: "it's a \"test case\"",
		},
		{
			name:       "nested quotes",
			s:          "'she said \"hello\"'",
			want:       QuotedShellArgs{"she said \"hello\""},
			wantJoined: "she said \"hello\"",
		},
		{
			name:       "adjacent quotes",
			s:          "\"hello\"'world'",
			want:       QuotedShellArgs{"hello", "world"},
			wantJoined: "hello world",
		}, {
			name:       "escaped quotes",
			s:          `one "two three" 'four five' six\ seven`,
			want:       QuotedShellArgs{"one", "two three", "four five", "six seven"},
			wantJoined: `one "two three" "four five" "six seven"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewQuotedShellArgs(tt.s)
			if assert.Equal(t, tt.want, got, "NewQuotedShellArgs(%q)", tt.s) {
				gotJoined := got.String()
				assert.Equal(t, tt.wantJoined, gotJoined, "QuotedShellArgs.String() for input %q", tt.s)
			}
		})
	}
}
