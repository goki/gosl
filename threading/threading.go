package threading

import (
	"math"
	"sync"

	"github.com/goki/ki/ints"
)

// Maps the given function across the [0, total) range of items, using
// nThreads goroutines.
func ParallelRun(fun func(st, ed int), total int, nThreads int) {
	itemsPerThr := int(math.Ceil(float64(total) / float64(nThreads)))
	waitGroup := sync.WaitGroup{}
	for start := 0; start < total; start += itemsPerThr {
		start := start // be extra sure with closure
		end := ints.MinInt(start+itemsPerThr, total)
		waitGroup.Add(1) // todo: move out of loop
		go func() {
			fun(start, end)
			waitGroup.Done()
		}()
	}
	waitGroup.Wait()
}
