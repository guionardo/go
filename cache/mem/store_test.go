package mem

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_MemStore(t *testing.T) {
	t.Parallel()

	t.Run("get_entry", func(t *testing.T) {
		t.Parallel()

		s := newMemStore[string, string](10)
		assert.NotNil(t, s)

		now := time.Now()
		s.set("valid", "ok", new(now.AddDate(0, 0, 1)))
		value, ok := s.get("valid")
		assert.True(t, ok)
		assert.NotEmpty(t, value)

		s.set("expired", "oh-no", new(now.AddDate(0, 0, -1)))
		value, ok = s.get("expired")
		assert.False(t, ok)
		assert.Empty(t, value)

		s.set("expired", "oh-no-2", new(now.AddDate(0, 0, -1)))
		removedCount := s.removeExpired()
		assert.Equal(t, 1, removedCount)
	})
}

func Test_MaxEntries(t *testing.T) {
	t.Parallel()

	const maxStore = 10

	s := newMemStore[string, string](10)

	for n := range maxStore + 4 {
		s.set(fmt.Sprintf("key_%d", n), "value", new(time.Now().AddDate(0, 0, 1)))
		assert.LessOrEqual(t, s.entries.Len(), 10)
	}
}
