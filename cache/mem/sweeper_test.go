package mem

import (
	"testing"
	"time"
)

func TestSweeper_sweep_removes_expired(t *testing.T) {
	t.Parallel()

	c := &Cache[string, string]{
		entries: make(map[string]*entry[string]),
	}
	now := time.Now()

	c.entries["expired"] = &entry[string]{value: "gone", expiresAt: ptr(now.Add(-time.Second))}
	c.entries["fresh"] = &entry[string]{value: "here", expiresAt: ptr(now.Add(time.Hour))}
	c.entries["no_ttl"] = &entry[string]{value: "forever", expiresAt: nil}

	c.sweep()

	_, ok := c.entries["expired"]
	if ok {
		t.Error("expected expired entry to be removed")
	}

	_, ok = c.entries["fresh"]
	if !ok {
		t.Error("expected fresh entry to remain")
	}

	_, ok = c.entries["no_ttl"]
	if !ok {
		t.Error("expected no-ttl entry to remain")
	}
}

func TestSweeper_sweep_all_fresh(t *testing.T) {
	t.Parallel()

	c := &Cache[string, string]{
		entries: make(map[string]*entry[string]),
	}
	now := time.Now()

	c.entries["a"] = &entry[string]{value: "1", expiresAt: ptr(now.Add(time.Hour))}
	c.entries["b"] = &entry[string]{value: "2", expiresAt: ptr(now.Add(2 * time.Hour))}

	c.sweep()

	if len(c.entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(c.entries))
	}
}

func ptr(t time.Time) *time.Time {
	return &t
}
