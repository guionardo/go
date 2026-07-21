package release

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
)

type (
	UpdateState int

	UpdateResult struct {
		Release *Release
		Updated bool
		Current bool
		State   UpdateState
		Err     error
	}

	lockRef struct {
		path string
		file *os.File
	}
)

const (
	UpdateStateUnknown UpdateState = iota
	UpdateStateChecked
	UpdateStateDownloaded
	UpdateStateSwapperSpawned

	swapperArgCount = 7
	filePerms       = 0o644
)

var (
	ErrUpdateInProgress = errors.New("update already in progress")
	testCurrentVersion  string
)

func (r *UpdateResult) String() string {
	if r.Err != nil {
		return fmt.Sprintf("UpdateResult{Err: %v}", r.Err)
	}

	if r.Updated {
		return fmt.Sprintf("UpdateResult{Updated: true, Version: %s}", r.Release.TagName)
	}

	if r.Current {
		return "UpdateResult{Current: true}"
	}

	return fmt.Sprintf("UpdateResult{State: %d}", r.State)
}

func updateLockPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Dir(exe), ".update.lock"), nil
}

func computeFileSHA256(path string) (string, error) {
	//nolint:gosec // path comes from DownloadUpdate output, not user input
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:]), nil
}

func acquireUpdateLock() (*lockRef, error) {
	lockPath, err := updateLockPath()
	if err != nil {
		return nil, fmt.Errorf("update lock path: %w", err)
	}

	//nolint:gosec // lockPath comes from updateLockPath() which uses os.Executable()
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, filePerms)
	if err != nil {
		if os.IsExist(err) {
			return nil, ErrUpdateInProgress
		}

		return nil, fmt.Errorf("acquire lock: %w", err)
	}

	return &lockRef{path: lockPath, file: lockFile}, nil
}

func downloadAndSwap(ctx context.Context, rel *Release) (bool, error) {
	exePath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("get executable: %w", err)
	}

	exeDir := filepath.Dir(exePath)

	downloadedPath, err := DownloadUpdate(ctx, rel, exeDir)
	if err != nil {
		return false, fmt.Errorf("download update: %w", err)
	}

	sha256Hex, err := computeFileSHA256(downloadedPath)
	if err != nil {
		return false, fmt.Errorf("compute sha256: %w", err)
	}

	swapperPath, err := ExtractSwapper(exeDir)
	if err != nil {
		return false, fmt.Errorf("extract swapper: %w", err)
	}

	argv := make([]string, 0, swapperArgCount+len(os.Args[1:]))
	argv = append(argv, swapperPath, "--new-binary", downloadedPath, "--checksum", sha256Hex, "--target", exePath)
	argv = append(argv, os.Args[1:]...)

	//nolint:gosec // swapper path comes from ExtractSwapper which writes to a known dir
	proc, err := os.StartProcess(swapperPath, argv, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		return false, fmt.Errorf("start swapper: %w", err)
	}

	if err := proc.Release(); err != nil {
		return false, fmt.Errorf("release swapper process: %w", err)
	}

	return true, nil
}

func resolveCurrentVersion() (*version.Version, error) {
	if testCurrentVersion != "" {
		return ParseVersion(testCurrentVersion)
	}

	return GetCurrentVersion()
}

//nolint:cyclop
func PerformSelfUpdate(ctx context.Context, opts ...Option) *UpdateResult {
	result := &UpdateResult{}

	currentVersion, err := resolveCurrentVersion()
	if err != nil {
		result.Err = fmt.Errorf("get current version: %w", err)

		return result
	}

	rel, hasUpdate, err := CheckForUpdate(ctx, currentVersion.String(), opts...)
	if err != nil {
		result.Err = fmt.Errorf("check for update: %w", err)

		return result
	}

	result.Release = rel
	result.State = UpdateStateChecked

	if !hasUpdate {
		result.Current = true

		return result
	}

	lock, err := acquireUpdateLock()
	if err != nil {
		result.Err = err

		return result
	}

	defer func() {
		_ = lock.file.Close()
		_ = os.Remove(lock.path)
	}()

	result.State = UpdateStateDownloaded

	updated, err := downloadAndSwap(ctx, rel)
	if err != nil {
		result.Err = err

		return result
	}

	result.Updated = updated
	result.State = UpdateStateSwapperSpawned

	return result
}
