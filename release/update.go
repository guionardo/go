package release

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Config struct {
	Owner string
	Repo  string
	Token string
}

type Option interface {
	apply(*Config)
}

type optionFunc func(*Config)

func (f optionFunc) apply(c *Config) {
	f(c)
}

var githubAPIBase = "https://api.github.com"

func WithOwner(owner string) Option {
	return optionFunc(func(c *Config) {
		c.Owner = owner
	})
}

func WithRepo(repo string) Option {
	return optionFunc(func(c *Config) {
		c.Repo = repo
	})
}

func WithGitHubToken(token string) Option {
	return optionFunc(func(c *Config) {
		c.Token = token
	})
}

func CheckForUpdate(ctx context.Context, currentVersion string, opts ...Option) (*Release, bool, error) {
	cfg := &Config{}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	owner, repo := cfg.Owner, cfg.Repo

	if owner == "" || repo == "" {
		moduleName, err := getCurrentModule()
		if err != nil {
			return nil, false, err
		}

		u, err := url.Parse(moduleName)
		if err != nil {
			return nil, false, err
		}

		words := strings.Split(u.Path, "/")
		if len(words) < 3 {
			return nil, false, fmt.Errorf("invalid module path: %s", moduleName)
		}

		if owner == "" {
			owner = words[1]
		}
		if repo == "" {
			repo = words[2]
		}
	}

	currVer, err := ParseVersion(currentVersion)
	if err != nil {
		return nil, false, fmt.Errorf("parsing current version: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repos/%s/%s/releases/latest", githubAPIBase, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, false, err
	}

	req.Header.Add("X-Github-Api-Version", "2026-03-10")
	req.Header.Add("Accept", "application/vnd.github+json")
	if cfg.Token != "" {
		req.Header.Add("Authorization", "Bearer "+cfg.Token)
	}

	resp, err := githubClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, false, fmt.Errorf("failed deserialization of release: %w", err)
	}

	relVer, err := ParseVersion(release.TagName)
	if err != nil {
		return nil, false, fmt.Errorf("parsing release version %q: %w", release.TagName, err)
	}

	return &release, relVer.Compare(currVer) > 0, nil
}

func DownloadUpdate(ctx context.Context, rel *Release, targetDir string) (string, error) {
	asset := findAsset(rel, runtime.GOOS, runtime.GOARCH)
	if asset == nil {
		return "", fmt.Errorf("no asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}

	filePath := filepath.Join(targetDir, asset.Name)
	f, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	if err := asset.Download(f); err != nil {
		os.Remove(filePath)

		return "", fmt.Errorf("download failed: %w", err)
	}

	return filePath, nil
}

func findAsset(rel *Release, goos, goarch string) *Asset {
	for i := range rel.Assets {
		name := strings.ToLower(rel.Assets[i].Name)
		if strings.Contains(name, goos) && strings.Contains(name, goarch) {
			return &rel.Assets[i]
		}
	}

	return nil
}
