package mem

import (
	"slices"
	"time"

	"github.com/guionardo/go/flow"
)

type (
	entry[V any] struct {
		value     V
		expiresAt *time.Time // nil means no expiry
	}
	memStore[K comparable, V any] struct {
		entries    *flow.OrderedMap[K, *entry[V]]
		maxEntries uint
	}
)

func (e entry[V]) isExpired() bool {
	return e.expiresAt != nil && time.Now().After(*e.expiresAt)
}

func newMemStore[K comparable, V any](maxEntries uint) *memStore[K, V] {
	return &memStore[K, V]{
		entries:    flow.NewOrderedMap[K, *entry[V]](),
		maxEntries: maxEntries,
	}
}

func (ms *memStore[K, V]) get(key K) (value V, found bool) {
	v, ok := ms.entries.Get(key)
	if ok {
		if v.isExpired() {
			ms.entries.Delete(key)
			return value, false
		}

		return v.value, true
	}

	return value, false
}

func (ms *memStore[K, V]) set(key K, value V, expiresAt *time.Time) {
	ms.entries.Set(key, &entry[V]{value: value, expiresAt: expiresAt})

	ms.validateMaxEntries()
}

func (ms *memStore[K, V]) validateMaxEntries() {
	if ms.maxEntries == 0 {
		return
	}

	for uint(ms.entries.Len()) > ms.maxEntries {
		toRemoveKeys := slices.Collect(ms.entries.Keys(0, -int(ms.maxEntries)+1))

		for _, key := range toRemoveKeys {
			ms.entries.Delete(key)
		}
	}
}

func (ms *memStore[K, V]) delete(key K) {
	ms.entries.Delete(key)
}

func (ms *memStore[K, V]) removeExpired() int {
	count := 0

	for key, entry := range ms.entries.Range(0, -1) {
		if entry.isExpired() {
			ms.entries.Delete(key)

			count += 1
		}
	}

	return count
}
