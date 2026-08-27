package geo

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNilRedisDispatchLockIsExclusive(t *testing.T) {
	g := NewGeoService(nil)
	ctx := context.Background()

	ok, err := g.AcquireDispatchLock(ctx, "drv-1", "trip-a", time.Second)
	if err != nil || !ok {
		t.Fatalf("first lock failed ok=%v err=%v", ok, err)
	}
	ok, err = g.AcquireDispatchLock(ctx, "drv-1", "trip-b", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("second lock on same driver must fail")
	}
	if err := g.ReleaseDispatchLock(ctx, "drv-1"); err != nil {
		t.Fatal(err)
	}
	ok, err = g.AcquireDispatchLock(ctx, "drv-1", "trip-c", time.Second)
	if err != nil || !ok {
		t.Fatal("lock should be reusable after release")
	}
}

func TestConcurrentLockAcquisition(t *testing.T) {
	g := NewGeoService(nil)
	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	var won int
	var mu sync.Mutex
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			ok, err := g.AcquireDispatchLock(context.Background(), "drv-hot", "trip", time.Second)
			if err != nil {
				t.Errorf("lock err: %v", err)
				return
			}
			if ok {
				mu.Lock()
				won++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if won != 1 {
		t.Fatalf("expected 1 winner, got %d", won)
	}
}
