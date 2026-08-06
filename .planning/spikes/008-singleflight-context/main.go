package main

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// Spike 008: is blocking singleflight.Do() acceptable in an HTTP request path?
// Under a slow setter, context-canceled callers using blocking Do() sit blocked
// until the leader finishes — accumulating goroutines. DoChan + select lets a
// canceled waiter abandon its own wait promptly.
//
// Compare two wrappers under a 300ms setter with waves of callers that all
// cancel their context at ~15ms:
//   DoBlk — blocking Do(); canceled waiters can't leave
//   DoCtx — DoChan + select; canceled waiters leave at ~15ms
//
// Metrics: peak live goroutines + average waiter wall-time.

const (
	slowMs = 300
	cancel = 15 * time.Millisecond
	batch  = 64
	waves  = 5
)

func main() {
	blkPeak, blkExit := measure("Do(blocking)   ", doBlocking)
	ctxPeak, ctxExit := measure("DoChan(ctx-aware)", doChanCtx)

	fmt.Println("\n=== SUMMARY ===")
	fmt.Printf("Do-blocking   : peak-goroutines=%d  avg-waiter=~%v  (callers stuck till setter ends)\n", blkPeak, blkExit)
	fmt.Printf("DoChan-ctx    : peak-goroutines=%d  avg-waiter=~%v  (callers leave at cancel)\n", ctxPeak, ctxExit)
}

func measure(name string, wrapper func(context.Context, *singleflight.Group)) (int64, time.Duration) {
	var gf singleflight.Group
	var peak atomic.Int64
	var exitSum atomic.Int64
	var exitCnt atomic.Int64
	stop := make(chan struct{})

	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			n := int64(runtime.NumGoroutine())
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(4 * time.Millisecond)
		}
	}()

	var wg sync.WaitGroup
	for wave := 0; wave < waves; wave++ {
		for i := 0; i < batch; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), cancel)
				defer cancel()
				st := time.Now()
				wrapper(ctx, &gf)
				exitSum.Add(int64(time.Since(st)))
				exitCnt.Add(1)
			}()
		}
		wg.Wait()
		time.Sleep(slowMs + 30*time.Millisecond) // let the group settle before the next wave
	}
	close(stop)

	avg := time.Duration(exitSum.Load() / max(1, exitCnt.Load()))
	fmt.Printf("  %s  peak-goroutines=%-6d avg-waiter=~%v\n", name, peak.Load(), avg)
	return peak.Load(), avg
}

// doBlocking: a canceled waiter CANNOT leave until the 300ms leader finishes.
func doBlocking(_ context.Context, g *singleflight.Group) {
	_, _, _ = g.Do("slow", func() (any, error) {
		time.Sleep(slowMs * time.Millisecond)
		return "v", nil
	})
}

// doChanCtx: a canceled waiter abandons its own wait via select on its ctx.
func doChanCtx(ctx context.Context, g *singleflight.Group) {
	ch := g.DoChan("slow", func() (any, error) {
		time.Sleep(slowMs * time.Millisecond)
		return "v", nil
	})
	select {
	case <-ch:
	case <-ctx.Done():
	}
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}