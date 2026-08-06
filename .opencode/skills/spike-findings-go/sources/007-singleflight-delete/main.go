package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/guionardo/go/cache"
	"golang.org/x/sync/singleflight"
)

// Spike 007: does singleflight resurrect a key that was Deleted while its
// setter was still in flight?
//
// Classic pattern:  miss -> sf.Do(key, compute + Set).
// If another goroutine calls Delete(key) mid-flight, the in-flight leader still
// finishes its Set, re-populating a key the caller explicitly deleted.
//
// Prevention candidates:
//   A. bare Delete
//   B. Delete + group.Forget(key)
//   C. Delete + deletion-generation tombstone re-checked inside the fn before Set

func main() {
	run("A-bare-Delete", false, false)
	run("B-Delete+Forget", true, false)
	run("C-Delete+gen-tombstone", false, true)
}

type store struct {
	mu   sync.Mutex
	data map[string]string
}

func newStore() *store { return &store{data: map[string]string{}} }

func (s *store) get(k string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[k]
	if !ok {
		return "", cache.ErrMiss
	}
	return v, nil
}
func (s *store) set(k, v string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = v
	return nil
}
func (s *store) del(k string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, k)
	return nil
}

// getOrSet mirrors the helper pattern; isDeletedGate() is re-checked right
// before Set when the tombstone mode is active.
func getOrSet(
	g *singleflight.Group, st *store, key string,
	isDeleted func() bool, setter func() (string, error),
) (string, error) {
	if v, err := st.get(key); err == nil {
		return v, nil
	}
	v, err, _ := g.Do(key, func() (any, error) {
		if v, err := st.get(key); err == nil {
			return v, nil
		}
		c, err := setter()
		if err != nil {
			return "", err
		}
		if isDeleted != nil && isDeleted() {
			return "", cache.ErrMiss // do NOT Set: key was deleted while computing
		}
		if err := st.set(key, c); err != nil {
			return "", err
		}
		return c, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func run(label string, useForget bool, useGen bool) {
	var gf singleflight.Group
	st := newStore()
	var setterRuns atomic.Int64
	var leaderDone = make(chan struct{})
	deletedWhileFly := atomic.Bool{}

	// leader goroutine performs GetOrSet; the gate closure asks "deleted yet?"
	go func() {
		defer close(leaderDone)
		var gate func() bool
		if useGen {
			gate = func() bool { return deletedWhileFly.Load() }
		}
		_, _ = getOrSet(&gf, st, "k", gate, func() (string, error) {
			setterRuns.Add(1)
			time.Sleep(40 * time.Millisecond)
			return "computed", nil
		})
	}()

	time.Sleep(8 * time.Millisecond) // leader in-flight
	st.del("k")
	deletedWhileFly.Store(true)
	if useForget {
		gf.Forget("k")
	}
	<-leaderDone

	stored, _ := st.get("k")
	v2, _ := getOrSet(&gf, st, "k", nil, func() (string, error) {
		setterRuns.Add(1)
		return "second", nil
	})
	_ = v2

	resurrected := stored == "computed"
	fmt.Printf("[%s] stored_after=%q resurrected=%v setterRuns=%d\n",
		label, stored, resurrected, setterRuns.Load())

	switch label {
	case "A-bare-Delete", "B-Delete+Forget":
		if resurrected {
			fmt.Println("  => RESURRECTED: leader's late Set() overwrote the Delete. (B: Forget did NOT help)")
		} else {
			fmt.Println("  => not resurrected (timing luck; not guaranteed)")
		}
	case "C-Delete+gen-tombstone":
		if !resurrected {
			fmt.Println("  => PREVENTED reliably: tombstone checked inside Do fn suppresses the late Set")
		}
	}
}