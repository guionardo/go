package release

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/hashicorp/go-version"
)

func ParseVersion(s string) (*version.Version, error) {
	v, err := version.NewVersion(s)
	if err != nil {
		return nil, fmt.Errorf("invalid version string %q: %w", s, err)
	}

	return v, nil
}

func GetCurrentVersion() (*version.Version, error) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, errors.New("build info not found")
	}

	verStr := buildInfo.Main.Version
	if verStr == "" || verStr == "(devel)" {
		return nil, fmt.Errorf("version not set in build info: %q", verStr)
	}

	return ParseVersion(verStr)
}
