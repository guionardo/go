package projectprobe

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
)

// maxManifestSize caps manifest reads at 1 MB (ROBT-02). Pathological inputs
// (giant or binary files) yield a non-match, never a panic or OOM.
const maxManifestSize = 1 << 20 // 1 MB

var utf8BOM = []byte{0xEF, 0xBB, 0xBF} // UTF-8 encoded U+FEFF

// readManifest returns the BOM-stripped content of <folder>/<name> when it
// exists, is a regular file, and is at most maxManifestSize bytes. Any other
// outcome returns ok=false so callers degrade to Unknown (D-03).
func readManifest(folder, name string) (content []byte, ok bool) {
	// #nosec G304 -- name is always a compile-time constant at call sites
	// (phases 11-13: "go.mod", "package.json", ...), never user input;
	// root-scoped by construction (DETC-01).
	f, err := os.Open(filepath.Join(folder, name))
	if err != nil {
		return nil, false
	}
	defer f.Close() //nolint: errcheck

	content, err = io.ReadAll(io.LimitReader(f, maxManifestSize+1))
	if err != nil || len(content) > maxManifestSize {
		return nil, false
	}

	return bytes.TrimPrefix(content, utf8BOM), true
}
