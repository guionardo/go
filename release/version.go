package release

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

const (
	versionPartsCount = 3
	maxSplitParts     = 2
)

func ParseVersion(s string) (Version, error) {
	v := strings.TrimPrefix(s, "v")

	if idx := strings.Index(v, "+"); idx >= 0 {
		v = v[:idx]
	}

	parts := strings.SplitN(v, "-", maxSplitParts)
	digits := strings.Split(parts[0], ".")

	if len(digits) != versionPartsCount {
		return Version{}, fmt.Errorf("invalid version string: %s", s)
	}

	major, err := strconv.Atoi(digits[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version in %q", s)
	}

	minor, err := strconv.Atoi(digits[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version in %q", s)
	}

	patch, err := strconv.Atoi(digits[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version in %q", s)
	}

	prerelease := ""
	if len(parts) > 1 {
		prerelease = parts[1]
	}

	return Version{Major: major, Minor: minor, Patch: patch, Prerelease: prerelease}, nil
}

func (v Version) String() string {
	s := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}

	return s
}

func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return sign(v.Major - other.Major)
	}

	if v.Minor != other.Minor {
		return sign(v.Minor - other.Minor)
	}

	if v.Patch != other.Patch {
		return sign(v.Patch - other.Patch)
	}

	return comparePrerelease(v.Prerelease, other.Prerelease)
}

func comparePrerelease(a, b string) int {
	if a == "" && b == "" {
		return 0
	}
	if a == "" {
		return 1
	}
	if b == "" {
		return -1
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}

	return 0
}

func sign(n int) int {
	if n > 0 {
		return 1
	}

	if n < 0 {
		return -1
	}

	return 0
}

func GetCurrentVersion() (Version, error) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return Version{}, errors.New("build info not found")
	}

	version := buildInfo.Main.Version
	if version == "" || version == "(devel)" {
		return Version{}, fmt.Errorf("version not set in build info: %q", version)
	}

	return ParseVersion(version)
}
