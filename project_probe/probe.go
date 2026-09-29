package projectprobe

import (
	"fmt"
	"os"
	"path/filepath"
)

// Probe inspects folder and returns its project data.
//
// Probe never fails on content: an empty, unrecognized, or ignore-list-only
// folder returns ProjectData with LanguageUnknown and a nil error. Errors are
// reserved for hard folder-level I/O failures — ErrFolderNotFound,
// ErrNotDirectory, ErrPermissionDenied — always wrapped with the failing
// path (D-02). The returned Folder field is filepath.Clean(as-given);
// relative input stays relative (D-08). An empty folder argument returns
// ErrFolderNotFound and never probes the current directory (OQ-1).
func Probe(folder string) (ProjectData, error) {
	if folder == "" { // OQ-1 guard: never probe the cwd
		return ProjectData{}, fmt.Errorf("probe %s: %w", folder, ErrFolderNotFound)
	}

	clean := filepath.Clean(folder) // D-09: one canonical path

	info, err := os.Stat(clean)
	if err != nil {
		return ProjectData{}, mapFolderError(clean, err)
	}

	if !info.IsDir() {
		return ProjectData{}, fmt.Errorf("probe %s: %w", clean, ErrNotDirectory)
	}

	entries, err := os.ReadDir(clean)
	if err != nil {
		return ProjectData{}, mapFolderError(clean, err)
	}

	data := ProjectData{Folder: clean, Language: LanguageUnknown}
	if !hasContent(entries) { // empty or only ignored dirs (D-12, DETC-09)
		return data, nil
	}

	if pd, ok := runDetectors(clean); ok { // empty registry in Phase 10
		data.Language = pd.Language
		data.Name = pd.Name
		data.Version = pd.Version
		data.Description = pd.Description
	}

	return data, nil
}
