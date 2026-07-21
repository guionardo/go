package release

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:goconst // table-driven test; repeated strings are intentional
func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
		err      bool
	}{
		{input: "v1.2.3", expected: "1.2.3"},
		{input: "1.2.3", expected: "1.2.3"},
		{input: "v1.2.3-beta1", expected: "1.2.3-beta1"},
		{input: "v1.2.3+sha.abc1234", expected: "1.2.3+sha.abc1234"},
		{input: "v0.0.0-20250101120000-abc1234", expected: "0.0.0-20250101120000-abc1234"},
		{input: "v10.20.30", expected: "10.20.30"},
		{input: "v1.2", expected: "1.2.0"},
		{input: "v1.2.3.4", expected: "1.2.3.4"},
		{input: "invalid", err: true},
		{input: "", err: true},
		{input: "abc.def.ghi", err: true},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			v, err := ParseVersion(test.input)
			if test.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expected, v.String())
		})
	}
}

func TestVersionCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a, b     string
		expected int
	}{
		{"equal", "1.0.0", "1.0.0", 0},
		{"major greater", "2.0.0", "1.0.0", 1},
		{"major less", "1.0.0", "2.0.0", -1},
		{"minor greater", "1.2.0", "1.1.0", 1},
		{"minor less", "1.1.0", "1.2.0", -1},
		{"patch greater", "1.1.3", "1.1.2", 1},
		{"patch less", "1.1.2", "1.1.3", -1},
		{"prerelease less than release", "1.0.0-alpha", "1.0.0", -1},
		{"release greater than prerelease", "1.0.0", "1.0.0-alpha", 1},
		{"prerelease alpha < beta", "1.0.0-alpha", "1.0.0-beta", -1},
		{"prerelease beta > alpha", "1.0.0-beta", "1.0.0-alpha", 1},
		{"prerelease equal", "1.0.0-rc1", "1.0.0-rc1", 0},
		{"pseudo less than release", "0.0.0-20250101", "1.0.0", -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			a, err := ParseVersion(test.a)
			require.NoError(t, err)

			b, err := ParseVersion(test.b)
			require.NoError(t, err)
			require.Equal(t, test.expected, a.Compare(b))
		})
	}
}

func TestVersionString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"v1.2.3", "1.2.3"},
		{"v1.2.3-beta1", "1.2.3-beta1"},
		{"v10.20.30", "10.20.30"},
		{"v0.0.0-20250101120000-abc1234", "0.0.0-20250101120000-abc1234"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			t.Parallel()

			v, err := ParseVersion(test.input)
			require.NoError(t, err)
			require.Equal(t, test.expected, v.String())
		})
	}
}

func TestGetCurrentVersion(t *testing.T) {
	t.Parallel()

	v, err := GetCurrentVersion()
	if err != nil {
		return
	}

	segments := v.Segments()
	sum := segments[0] + segments[1] + segments[2]
	require.NotZero(t, sum)
}
