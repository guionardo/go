package mem

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSweeper_sweep_removes_expired(t *testing.T) {
	t.Parallel()

	c := New[string, string](
		t.Context(),
		WithDefaultTTL(1*time.Second),
		WithMaxEntries(10),
		WithSweepInterval(500*time.Millisecond),
	)

	_ = c.Set(t.Context(), "expired", "gone", time.Millisecond)
	_ = c.Set(t.Context(), "fresh", "here", time.Hour)
	_ = c.Set(t.Context(), "no_ttl", "forever", 0)
	_ = c.Set(t.Context(), "expired2", "gone", time.Millisecond)

	time.Sleep(time.Second)

	v, err := c.Get(t.Context(), "expired")
	require.Error(t, err)
	assert.Empty(t, v)

	v, err = c.Get(t.Context(), "fresh")
	require.NoError(t, err)
	assert.Equal(t, "here", v)

	v, err = c.Get(t.Context(), "no_ttl")
	require.NoError(t, err)
	assert.Equal(t, "forever", v)
}
