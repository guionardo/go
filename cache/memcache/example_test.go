package memcache_test

import (
	"net"
	"testing"
	"time"

	"github.com/guionardo/go/cache/memcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoExampleMemcache(t *testing.T) {
	t.Helper()

	conn, err := net.DialTimeout("tcp", "localhost:11211", 100*time.Millisecond)
	if err != nil {
		t.Skip("memcache not available")
	}
	conn.Close()
}

func TestMemcacheExample_SetGet(t *testing.T) {
	skipIfNoExampleMemcache(t)

	c := memcache.New[string, string]()

	err := c.Set(t.Context(), "example", "memcache-value")
	require.NoError(t, err)

	value, err := c.Get(t.Context(), "example")
	require.NoError(t, err)
	assert.Equal(t, "memcache-value", value)

	err = c.Close()
	require.NoError(t, err)
}
