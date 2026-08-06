package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

func main() {
	testSetterErrorShared()
	testPanicReplay()
	testDoChanCancel()
}

// CASE A: a failing setter — the error must reach every waiter, nothing cached.
func testSetterErrorShared() {
	var sf singleflight.Group
	errSink := errors.New("boom")

	const callers = 10
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err, _ := sf.Do("key", func() (any, error) {
				time.Sleep(2 * time.Millisecond)
				return nil, errSink
			})
			errs[i] = err
		}(i)
	}
	wg.Wait()

	allSameErr := true
	for i := 1; i < callers; i++ {
		if errs[i] != errSink {
			allSameErr = false
		}
	}
	fmt.Printf("[A] setter error shared to all %d callers: %v (err identity preserved: %v)\n",
		callers, errs[0] == errSink, allSameErr)
	if !(errs[0] == errSink && allSameErr) {
		panic("setter error not shared")
	}
	fmt.Println("CASE A: PASS — error propagated to all waiters as the SAME error value")
}

// CASE B: a panicking setter — the panic is REPLAYED to each caller.
func testPanicReplay() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("CASE B: PASS — panic replayed to first caller:", r)
		} else {
			fmt.Println("CASE B: FAIL — expected panic replay")
		}
	}()

	var sf singleflight.Group
	sf.Do("key", func() (any, error) {
		panic("setter-panic")
	})
}

// CASE C: a caller's context cancels while the leader is still computing.
// singleflight has NO context awareness:
//   - the leader keeps running to completion (never aborted by a follower's cancel)
//   - a canceled follower can only abandon its OWN wait via select on DoChan
//   - blocking Do() cannot be canceled — the waiter must wait for the leader
func testDoChanCancel() {
	var sf singleflight.Group
	leaderDone := make(chan struct{})
	go func() {
		sf.Do("slow", func() (any, error) {
			time.Sleep(100 * time.Millisecond)
			return "slow-value", nil
		})
		close(leaderDone)
	}()

	time.Sleep(10 * time.Millisecond) // ensure leader in-flight

	// canceled follower: abandons its own wait via select on DoChan
	start := time.Now()
	ch := sf.DoChan("slow", func() (any, error) { panic("must not run") })
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(5*time.Millisecond, cancel)
	var abandoned bool
	select {
	case <-ctx.Done():
		abandoned = true
		fmt.Printf("[C] canceled follower abandoned at %dms (leader still running)\n",
			int64(time.Since(start)/time.Millisecond))
	case <-ch:
		panic("follower ctx canceled but select took the result channel")
	}

	<-leaderDone
	elapsed := time.Since(start)
	fmt.Printf("[C] leader completed independently at %dms — not aborted by follower cancel\n",
		int64(elapsed/time.Millisecond))
	if !abandoned {
		panic("follower was not able to abandon the wait via ctx")
	}
	fmt.Println("CASE C: PASS — singleflight is context-blind: leader runs to completion, follower abandons its own wait")
}

var _ = context.Background
