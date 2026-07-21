package release_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/guionardo/go/release"
	"github.com/stretchr/testify/require"
)

func TestGetLatestRelease(t *testing.T) {
	t.Parallel()

	rel, err := release.GetLatestRelease("iongion", "container-desktop")
	require.NoError(t, err)
	require.NotNil(t, rel)

	if len(rel.Assets) == 0 {
		t.Skip("no assets in latest release — nothing to download")
	}

	asset := rel.Assets[0]
	w := bytes.NewBuffer([]byte{})
	require.NoError(t, asset.Download(w))
	wb, _ := io.ReadAll(w)
	require.Len(t, wb, asset.Size)
}

func TestGetThisLatestRelease(t *testing.T) {
	t.Parallel()

	rel, err := release.GetThisLatestRelease()
	if err != nil {
		t.Skipf("skipping: GetThisLatestRelease failed (no build info or network): %v", err)
	}

	require.NotNil(t, rel)
}
