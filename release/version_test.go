package release

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected Version
		err      bool
	}{
		{"v1.2.3", Version{Major: 1, Minor: 2, Patch: 3}, false},
		{"1.2.3", Version{Major: 1, Minor: 2, Patch: 3}, false},
		{"v1.2.3-beta1", Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "beta1"}, false},
		{"v1.2.3+sha.abc1234", Version{Major: 1, Minor: 2, Patch: 3}, false},
		{"v0.0.0-20250101120000-abc1234", Version{Prerelease: "20250101120000-abc1234"}, false},
		{"v10.20.30", Version{Major: 10, Minor: 20, Patch: 30}, false},
		{"invalid", Version{}, true},
		{"v1.2", Version{}, true},
		{"", Version{}, true},
		{"v1.2.3.4", Version{}, true},
		{"abc.def.ghi", Version{}, true},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			v, err := ParseVersion(test.input)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, v)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a, b     Version
		expected int
	}{
		{"equal", Version{1, 0, 0, ""}, Version{1, 0, 0, ""}, 0},
		{"major greater", Version{2, 0, 0, ""}, Version{1, 0, 0, ""}, 1},
		{"major less", Version{1, 0, 0, ""}, Version{2, 0, 0, ""}, -1},
		{"minor greater", Version{1, 2, 0, ""}, Version{1, 1, 0, ""}, 1},
		{"minor less", Version{1, 1, 0, ""}, Version{1, 2, 0, ""}, -1},
		{"patch greater", Version{1, 1, 3, ""}, Version{1, 1, 2, ""}, 1},
		{"patch less", Version{1, 1, 2, ""}, Version{1, 1, 3, ""}, -1},
		{"prerelease less than release", Version{1, 0, 0, "alpha"}, Version{1, 0, 0, ""}, -1},
		{"release greater than prerelease", Version{1, 0, 0, ""}, Version{1, 0, 0, "alpha"}, 1},
		{"prerelease alpha < beta", Version{1, 0, 0, "alpha"}, Version{1, 0, 0, "beta"}, -1},
		{"prerelease beta > alpha", Version{1, 0, 0, "beta"}, Version{1, 0, 0, "alpha"}, 1},
		{"prerelease equal", Version{1, 0, 0, "rc1"}, Version{1, 0, 0, "rc1"}, 0},
		{"pseudo less than release", Version{0, 0, 0, "20250101"}, Version{1, 0, 0, ""}, -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.expected, test.a.Compare(test.b))
		})
	}
}

func TestVersionString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v        Version
		expected string
	}{
		{Version{1, 2, 3, ""}, "v1.2.3"},
		{Version{1, 2, 3, "beta1"}, "v1.2.3-beta1"},
		{Version{10, 20, 30, ""}, "v10.20.30"},
		{Version{0, 0, 0, "20250101120000-abc1234"}, "v0.0.0-20250101120000-abc1234"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.expected, test.v.String())
		})
	}
}

func TestGetCurrentVersion(t *testing.T) {
	t.Parallel()

	v, err := GetCurrentVersion()
	if err != nil {
		return
	}

	require.NotZero(t, v.Major+v.Minor+v.Patch)
}
