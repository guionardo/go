package release

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

var mu sync.Mutex

func TestFindAsset(t *testing.T) {
	t.Parallel()

	rel := &Release{
		Assets: []Asset{
			{Name: "myapp_darwin_amd64.tar.gz"},
			{Name: "myapp_linux_amd64.tar.gz"},
			{Name: "myapp_windows_amd64.zip"},
			{Name: "myapp_darwin_arm64.tar.gz"},
		},
	}

	asset := findAsset(rel, "darwin", "amd64")
	require.NotNil(t, asset)
	require.Equal(t, "myapp_darwin_amd64.tar.gz", asset.Name)

	asset = findAsset(rel, "linux", "amd64")
	require.NotNil(t, asset)
	require.Equal(t, "myapp_linux_amd64.tar.gz", asset.Name)

	asset = findAsset(rel, "windows", "amd64")
	require.NotNil(t, asset)
	require.Equal(t, "myapp_windows_amd64.zip", asset.Name)

	asset = findAsset(rel, "darwin", "arm64")
	require.NotNil(t, asset)
	require.Equal(t, "myapp_darwin_arm64.tar.gz", asset.Name)

	asset = findAsset(rel, "freebsd", "amd64")
	require.Nil(t, asset)
}

func TestFindAsset_CaseInsensitive(t *testing.T) {
	t.Parallel()

	rel := &Release{
		Assets: []Asset{
			{Name: "myapp_Darwin_AMD64.tar.gz"},
		},
	}

	asset := findAsset(rel, "darwin", "amd64")
	require.NotNil(t, asset)
	require.Equal(t, "myapp_Darwin_AMD64.tar.gz", asset.Name)
}

func TestFindAsset_EmptyAssets(t *testing.T) {
	t.Parallel()

	rel := &Release{Assets: []Asset{}}
	asset := findAsset(rel, "darwin", "amd64")
	require.Nil(t, asset)
}

func TestOptions(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	WithOwner("test-owner").apply(cfg)
	WithRepo("test-repo").apply(cfg)
	WithGitHubToken("test-token").apply(cfg)

	require.Equal(t, "test-owner", cfg.Owner)
	require.Equal(t, "test-repo", cfg.Repo)
	require.Equal(t, "test-token", cfg.Token)
}

func TestCheckForUpdate_NewerVersion(t *testing.T) {
	t.Parallel()
	mu.Lock()
	defer mu.Unlock()

	content := []byte("test binary content")
	d := digest.FromBytes(content)

	var (
		server    *httptest.Server
		serverURL string
	)
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases/latest") {
			releaseJSON := fmt.Sprintf(`{
				"tag_name": "v2.0.0",
				"name": "v2.0.0",
				"draft": false,
				"prerelease": false,
				"assets": [{
					"name": "myapp_%s_%s.tar.gz",
					"browser_download_url": "%s/download",
					"digest": "%s",
					"size": %d
				}]
			}`, runtime.GOOS, runtime.GOARCH, serverURL, d.String(), len(content))
			_, _ = fmt.Fprint(w, releaseJSON)
			return
		}
		_, _ = w.Write(content)
	}))
	defer server.Close()

	serverURL = server.URL

	originalBase := githubAPIBase
	githubAPIBase = serverURL
	defer func() { githubAPIBase = originalBase }()

	rel, newer, err := CheckForUpdate(context.Background(), "v1.0.0",
		WithOwner("test"), WithRepo("test"))
	require.NoError(t, err)
	require.NotNil(t, rel)
	require.True(t, newer)
	require.Equal(t, "v2.0.0", rel.TagName)
}

func TestCheckForUpdate_SameVersion(t *testing.T) {
	t.Parallel()
	mu.Lock()
	defer mu.Unlock()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		releaseJSON := `{
			"tag_name": "v1.0.0",
			"name": "v1.0.0",
			"draft": false,
			"prerelease": false,
			"assets": []
		}`
		_, _ = fmt.Fprint(w, releaseJSON)
	}))
	defer server.Close()

	originalBase := githubAPIBase
	githubAPIBase = server.URL
	defer func() { githubAPIBase = originalBase }()

	rel, newer, err := CheckForUpdate(context.Background(), "v1.0.0",
		WithOwner("test"), WithRepo("test"))
	require.NoError(t, err)
	require.NotNil(t, rel)
	require.False(t, newer)
}

func TestCheckForUpdate_OlderVersion(t *testing.T) {
	t.Parallel()
	mu.Lock()
	defer mu.Unlock()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		releaseJSON := `{
			"tag_name": "v0.9.0",
			"name": "v0.9.0",
			"draft": false,
			"prerelease": false,
			"assets": []
		}`
		_, _ = fmt.Fprint(w, releaseJSON)
	}))
	defer server.Close()

	originalBase := githubAPIBase
	githubAPIBase = server.URL
	defer func() { githubAPIBase = originalBase }()

	rel, newer, err := CheckForUpdate(context.Background(), "v1.0.0",
		WithOwner("test"), WithRepo("test"))
	require.NoError(t, err)
	require.NotNil(t, rel)
	require.False(t, newer)
}

func TestCheckForUpdate_WithToken(t *testing.T) {
	t.Parallel()
	mu.Lock()
	defer mu.Unlock()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		require.Equal(t, "Bearer ghp_test123", auth)

		releaseJSON := `{
			"tag_name": "v2.0.0",
			"name": "v2.0.0",
			"draft": false,
			"prerelease": false,
			"assets": []
		}`
		_, _ = fmt.Fprint(w, releaseJSON)
	}))
	defer server.Close()

	originalBase := githubAPIBase
	githubAPIBase = server.URL
	defer func() { githubAPIBase = originalBase }()

	rel, newer, err := CheckForUpdate(context.Background(), "v1.0.0",
		WithOwner("test"), WithRepo("test"), WithGitHubToken("ghp_test123"))
	require.NoError(t, err)
	require.NotNil(t, rel)
	require.True(t, newer)
}

func TestCheckForUpdate_InvalidVersion(t *testing.T) {
	t.Parallel()

	_, _, err := CheckForUpdate(context.Background(), "not-a-version",
		WithOwner("test"), WithRepo("test"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "parsing current version")
}

func TestCheckForUpdate_APIError(t *testing.T) {
	t.Parallel()
	mu.Lock()
	defer mu.Unlock()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, `{"message": "Forbidden"}`)
	}))
	defer server.Close()

	originalBase := githubAPIBase
	githubAPIBase = server.URL
	defer func() { githubAPIBase = originalBase }()

	_, _, err := CheckForUpdate(context.Background(), "v1.0.0",
		WithOwner("test"), WithRepo("test"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected status code: 403")
}

func TestDownloadUpdate(t *testing.T) {
	t.Parallel()
	mu.Lock()
	defer mu.Unlock()
	content := []byte("test binary content for download")
	d := digest.FromBytes(content)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	rel := &Release{
		TagName: "v2.0.0",
		Assets: []Asset{
			{
				Name:               fmt.Sprintf("myapp_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH),
				BrowserDownloadURL: server.URL + "/download",
				Digest:             d.String(),
				Size:               len(content),
			},
		},
	}

	dir := t.TempDir()

	filePath, err := DownloadUpdate(context.Background(), rel, dir)
	require.NoError(t, err)
	require.FileExists(t, filePath)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, content, data)
}

func TestDownloadUpdate_NoAsset(t *testing.T) {
	t.Parallel()

	rel := &Release{
		TagName: "v2.0.0",
		Assets:  []Asset{},
	}

	_, err := DownloadUpdate(context.Background(), rel, os.TempDir())
	require.Error(t, err)
	require.Contains(t, err.Error(), "no asset found")
}
