package profile

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/guionardo/go/config/merger"

	"gopkg.in/yaml.v3"
)

// GetScopedProfileContent tries to find the scope files, unmarshal and merge the content into a new YAML representation
func GetScopedProfileContent(basePath, defaultScope, scope string) ([]byte, error) {
	merged, err := getProfileMap(basePath, defaultScope, scope)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(merged)
}

func getProfileMap(basePath, defaultScope, scope string) (map[string]any, error) {
	defaultProfile, scopeProfile, err := getProfileFiles(basePath, defaultScope, scope)
	if err != nil {
		return nil, err
	}

	defaultMap, err := readProfileMap(defaultProfile)
	if err != nil {
		return nil, err
	}

	scopeMap, err := readProfileMap(scopeProfile)
	if err != nil {
		return nil, err
	}

	merged := merger.MergeMaps(defaultMap, scopeMap)

	return merged, nil
}

func readProfileMap(profile string) (map[string]any, error) {
	file, err := os.Open(path.Clean(profile))
	if err != nil {
		return nil, fmt.Errorf("error reading profile %s - %w", profile, err)
	}
	defer file.Close() //nolint: errcheck

	pm := make(map[string]any)

	err = yaml.NewDecoder(file).Decode(&pm)
	if err != nil {
		return nil, fmt.Errorf("error decoding profile %s - %w", profile, err)
	}

	return pm, nil
}

func getProfileFiles(
	basePath string,
	defaultScope string,
	scope string,
) (defaultProfile, scopeProfile string, err error) {
	for _, s := range []string{defaultScope, scope} {
		if err := validateScope(basePath, s); err != nil {
			return "", "", err
		}
	}

	defaultProfile, err = findYAMLFile(path.Join(basePath, defaultScope))
	if err != nil {
		return "", "", err
	}

	scopeProfile, err = findYAMLFile(path.Join(basePath, scope))
	if err != nil {
		return "", "", err
	}

	slog.Debug("getProfileFiles", slog.String("default", defaultProfile), slog.String("scope", scopeProfile))

	return defaultProfile, scopeProfile, nil
}

// validateScope reports whether scope resolves to a path inside basePath.
// Containment is decided on the cleaned relative path, so a base path of "."
// is valid and a sibling sharing the base name as a prefix (e.g. "../guarded-evil"
// against base "guarded") is still rejected.
func validateScope(basePath, scope string) error {
	base := filepath.Clean(basePath)
	target := filepath.Clean(filepath.Join(basePath, scope))

	rel, err := filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("scope %q escapes base path %q", scope, basePath)
	}

	return nil
}

func findYAMLFile(fileName string) (string, error) {
	for _, ext := range []string{"", ".yml", ".yaml", ".YML", ".YAML"} {
		if stat, err := os.Stat(fileName + ext); err == nil && !stat.IsDir() {
			slog.Debug("findYAMLFile", slog.String("filename", fileName+ext))
			return fileName + ext, nil
		}
	}

	return "", fmt.Errorf("file not found: %s", fileName)
}
