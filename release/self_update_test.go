package release

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testFilePerms   = 0o600
	testVerCurrent  = "v1.0.0"
	testVerNew      = "v2.0.0"
	swapperExecMode = 0o755
)

func TestComputeFileSHA256(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.bin")

	err := os.WriteFile(filePath, []byte("hello"), testFilePerms)
	require.NoError(t, err)

	sum, err := computeFileSHA256(filePath)
	require.NoError(t, err)
	require.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", sum)

	_, err = computeFileSHA256(filepath.Join(dir, "nonexistent"))
	require.Error(t, err)
}

func TestUpdateLockPath(t *testing.T) {
	t.Parallel()

	path, err := updateLockPath()
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(path, ".update.lock"))
	require.True(t, filepath.IsAbs(path))
}

func TestExtractSwapper(t *testing.T) {
	t.Parallel()

	swapperName := "swapper_" + runtime.GOOS + "_" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		swapperName += ".exe"
	}

	_, err := swapperBinary.ReadFile("swapper/" + swapperName)
	if err != nil {
		t.Skip("swapper binary not found - run 'make swapper' first")
	}

	dir := t.TempDir()

	path, err := ExtractSwapper(dir)
	require.NoError(t, err)
	require.FileExists(t, path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(swapperExecMode), info.Mode().Perm())
}

func TestUpdateResultString(t *testing.T) {
	t.Parallel()

	result := &UpdateResult{Err: errors.New("test error")}
	require.Contains(t, result.String(), "Err: test error")

	result = &UpdateResult{Updated: true, Release: &Release{TagName: testVerNew}}
	require.Contains(t, result.String(), "Updated: true")
	require.Contains(t, result.String(), testVerNew)

	result = &UpdateResult{Current: true}
	require.Contains(t, result.String(), "Current: true")

	result = &UpdateResult{State: UpdateStateDownloaded}
	require.Contains(t, result.String(), "State: 2")
}

func TestPerformSelfUpdate_NoVersion(t *testing.T) {
	t.Parallel()

	result := PerformSelfUpdate(context.Background(), WithOwner("test"), WithRepo("test"))
	require.Error(t, result.Err)
	require.Contains(t, result.Err.Error(), "get current version")
	require.False(t, result.Updated)
}

//nolint:paralleltest // modifies global state (githubAPIBase, testCurrentVersion)
func TestPerformSelfUpdate_Current(t *testing.T) {
	testCurrentVersion = testVerCurrent
	defer func() { testCurrentVersion = "" }()

	mu.Lock()
	defer mu.Unlock()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"tag_name": "` + testVerCurrent + `",
			"name": "` + testVerCurrent + `",
			"draft": false,
			"prerelease": false,
			"assets": []
		}`))
	}))
	defer server.Close()

	originalBase := githubAPIBase
	githubAPIBase = server.URL

	defer func() { githubAPIBase = originalBase }()

	result := PerformSelfUpdate(context.Background(), WithOwner("test"), WithRepo("test"))
	require.NoError(t, result.Err)
	require.True(t, result.Current)
	require.False(t, result.Updated)
	require.Equal(t, UpdateStateChecked, result.State)
}

//nolint:paralleltest // modifies global state (githubAPIBase, testCurrentVersion)
func TestPerformSelfUpdate_LockExists(t *testing.T) {
	testCurrentVersion = testVerCurrent
	defer func() { testCurrentVersion = "" }()

	mu.Lock()
	defer mu.Unlock()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"tag_name": "` + testVerNew + `",
			"name": "` + testVerNew + `",
			"draft": false,
			"prerelease": false,
			"assets": []
		}`))
	}))
	defer server.Close()

	originalBase := githubAPIBase
	githubAPIBase = server.URL

	defer func() { githubAPIBase = originalBase }()

	lockPath, err := updateLockPath()
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Dir(lockPath), 0o750)
	require.NoError(t, err)

	//nolint:gosec // lockPath comes from internal updateLockPath, not user input
	lockFile, err := os.Create(lockPath)
	require.NoError(t, err)
	err = lockFile.Close()
	require.NoError(t, err)

	defer func() { _ = os.Remove(lockPath) }()

	result := PerformSelfUpdate(context.Background(), WithOwner("test"), WithRepo("test"))
	require.Error(t, result.Err)
	require.ErrorIs(t, result.Err, ErrUpdateInProgress)
	require.False(t, result.Updated)
}

func TestDownloadAndSwap_NoMatchingAsset(t *testing.T) {
	t.Parallel()

	rel := &Release{TagName: "v2.0.0", Assets: []Asset{}}

	_, err := downloadAndSwap(context.Background(), rel)
	require.Error(t, err)
	require.Contains(t, err.Error(), "download update")
}

//nolint:paralleltest // modifies global state (githubAPIBase, testCurrentVersion)
func TestPerformSelfUpdate_DownloadFails(t *testing.T) {
	testCurrentVersion = testVerCurrent
	defer func() { testCurrentVersion = "" }()

	mu.Lock()
	defer mu.Unlock()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"tag_name": "` + testVerNew + `",
			"name": "` + testVerNew + `",
			"draft": false,
			"prerelease": false,
			"assets": []
		}`))
	}))
	defer server.Close()

	originalBase := githubAPIBase
	githubAPIBase = server.URL
	defer func() { githubAPIBase = originalBase }()

	result := PerformSelfUpdate(context.Background(), WithOwner("test"), WithRepo("test"))
	require.Error(t, result.Err)
	require.Contains(t, result.Err.Error(), "download update")
	require.Equal(t, UpdateStateDownloaded, result.State)
}

//nolint:paralleltest // modifies global state (githubAPIBase, testCurrentVersion)
func TestPerformSelfUpdate_APIError(t *testing.T) {
	testCurrentVersion = testVerCurrent
	defer func() { testCurrentVersion = "" }()

	mu.Lock()
	defer mu.Unlock()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalBase := githubAPIBase
	githubAPIBase = server.URL

	defer func() { githubAPIBase = originalBase }()

	result := PerformSelfUpdate(context.Background(), WithOwner("test"), WithRepo("test"))
	require.Error(t, result.Err)
	require.Contains(t, result.Err.Error(), "check for update")
}
