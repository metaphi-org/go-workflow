package limiter

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConcurrencyLimiter(t *testing.T) {
	maxConcurrency := 10
	cl := NewConcurrencyLimiter(maxConcurrency)

	lk := sync.Mutex{}
	currentCount := 0
	maxCount := 0

	wg := sync.WaitGroup{}
	var testFunc = func() {
		defer wg.Done()
		cl.Acquire()
		defer cl.Release()
		lk.Lock()
		currentCount++
		if currentCount > maxCount {
			maxCount = currentCount
		}
		lk.Unlock()
		time.Sleep(1 * time.Second)
		lk.Lock()
		currentCount--
		lk.Unlock()
	}
	startTime := time.Now()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go testFunc()
	}
	wg.Wait()
	timeElapsed := time.Since(startTime)
	assert.True(t, math.Abs(float64(timeElapsed.Milliseconds()-10*1000)) < 100, "should have takes equal to 10s")
	assert.Equal(t, maxConcurrency, maxCount)
}
