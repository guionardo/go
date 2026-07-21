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

	release, err := release.GetLatestRelease("iongion", "container-desktop")
	require.NoError(t, err)
	require.NotNil(t, release)

	w := bytes.NewBuffer([]byte{})
	require.NoError(t, release.Assets[0].Download(w))
	wb, _ := io.ReadAll(w)
	require.Len(t, wb, release.Assets[0].Size)
}

func TestGetThisLatestRelease(t *testing.T) {
	t.Parallel()

	release, err := release.GetThisLatestRelease()
	require.NoError(t, err)
	require.NotNil(t, release)
}
