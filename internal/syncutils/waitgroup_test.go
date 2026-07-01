package syncutils

import (
	"sync/atomic"
	"testing"
)

func TestMyWaitGroup_Go(t *testing.T) {
	var wg MyWaitGroup
	var counter int32

	// Launch 100 goroutines
	numGoroutines := 100
	for i := 0; i < numGoroutines; i++ {
		wg.Go(func() {
			atomic.AddInt32(&counter, 1)
		})
	}

	// Wait for all to finish
	wg.Wait()

	if int(counter) != numGoroutines {
		t.Errorf("expected counter to be %d, got %d", numGoroutines, counter)
	}
}
