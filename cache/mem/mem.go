package mem

import (
	"context"
	"sync"
	"time"

	"github.com/guionardo/go/cache"
)

type memoryCache[K comparable, V any] struct {
	mu         sync.RWMutex
	store      *memStore[K, V]
	defaultTTL time.Duration
	stop       chan struct{}
}

// New creates a new in-memory cache provider with optional functional options.
func New[K comparable, V any](ctx context.Context, opts ...ConfigFunc) cache.Cache[K, V] {
	config := Config{
		DefaultTTL:    5 * time.Minute, //nolint:mnd
		MaxEntries:    1000,            //nolint:mnd
		SweepInterval: 1 * time.Minute,
	}
	for _, opt := range opts {
		opt(&config)
	}

	cc := &memoryCache[K, V]{
		stop:       make(chan struct{}),
		store:      newMemStore[K, V](uint(config.MaxEntries)),
		defaultTTL: config.DefaultTTL,
	}

	if ctx != nil {
		go cc.sweepLoop(ctx, config.SweepInterval)
	}

	return cache.NewConcreteCache(cc)
}

func (c *memoryCache[K, V]) GetFunc(ctx context.Context, key K) (value V, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, ok := c.store.get(key)
	if !ok {
		err = cache.ErrMiss
	}

	return value, err
}

func (c *memoryCache[K, V]) SetFunc(ctx context.Context, key K, value V, ttl ...time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store.set(key, value, c.resolveTTL(ttl...))
	return nil
}

func (c *memoryCache[K, V]) DeleteFunc(ctx context.Context, key K) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store.delete(key)
	return nil
}

func (c *memoryCache[K, V]) CloseFunc() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = newMemStore[K, V](0)
	select {
	case <-c.stop:
		// already closed
	default:
		close(c.stop)
	}
	return nil
}

func (c *memoryCache[K, V]) resolveTTL(ttl ...time.Duration) *time.Time {
	if len(ttl) > 0 {
		if ttl[0] > 0 {
			t := time.Now().Add(ttl[0])
			return &t
		}

		return nil
	}
	if c.defaultTTL > 0 {
		t := time.Now().Add(c.defaultTTL)
		return &t
	}
	return nil
}
