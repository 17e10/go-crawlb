package mutex

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMutex(t *testing.T) {
	var wg sync.WaitGroup
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	defer t.Log("main done")

	wg.Add(1)
	go func() {
		mu := New(time.Second)
		for i := 0; i < 5; i++ {
			if err := mu.Lock(ctx); err != nil {
				break
			}
			t.Logf("#%d", i)
			mu.Unlock()
		}
		t.Log("go routine done")
		wg.Done()
	}()

	time.Sleep(3 * time.Second)
	cancel()

	wg.Wait()
}
