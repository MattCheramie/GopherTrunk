package tuner

import (
	"runtime"
	"sync"
)

// parallelTapThreshold is the smallest tap count worth parallelizing. Below it
// the barrier + scheduling overhead outweighs the per-tap fine-tune work and the
// serial loop is faster, so small plans stay serial and byte-identical to the
// pre-pool path. Dense wideband plans (the 70-DMR case) sit well above it.
const parallelTapThreshold = 16

// DefaultTapWorkers sizes the per-tap worker pool from the host CPU count,
// leaving headroom for the SDR read loop, the wideband pump goroutine, and the
// downstream decoders. It is capped because the shared channelizer FFT runs
// serially before the parallel region, so past a point more workers only add
// barrier contention without shortening the critical path.
func DefaultTapWorkers() int {
	switch n := runtime.NumCPU() - 2; {
	case n < 1:
		return 1
	case n > 8:
		return 8
	default:
		return n
	}
}

// tapPool runs a per-tap function over a fixed set of taps across persistent
// worker goroutines, with a reusable barrier so there is no per-chunk goroutine
// spawn. The taps are sharded into contiguous static ranges once at
// construction (the tap set is fixed after AddTap — dynamic add/remove is out of
// scope per the Bank contract); each round publishes the work function and
// releases every worker, then waits for all shards before returning, so the
// caller's post-Process reads see a fully-settled result and the next round's
// function write can't race a still-running worker.
type tapPool struct {
	fn       func(i int)
	starts   []chan struct{}
	bounds   [][2]int
	wg       sync.WaitGroup
	quit     chan struct{}
	quitOnce sync.Once
}

// newTapPool spawns workers goroutines sharding [0,total). workers is clamped to
// total so no shard is empty; callers gate creation on total >= a threshold.
func newTapPool(total, workers int) *tapPool {
	if workers > total {
		workers = total
	}
	if workers < 1 {
		workers = 1
	}
	p := &tapPool{
		starts: make([]chan struct{}, workers),
		bounds: make([][2]int, workers),
		quit:   make(chan struct{}),
	}
	// Even contiguous split: the first `rem` shards take one extra tap.
	base, rem := total/workers, total%workers
	lo := 0
	for w := 0; w < workers; w++ {
		sz := base
		if w < rem {
			sz++
		}
		p.bounds[w] = [2]int{lo, lo + sz}
		lo += sz
		p.starts[w] = make(chan struct{}, 1)
		go p.worker(w)
	}
	return p
}

func (p *tapPool) worker(w int) {
	lo, hi := p.bounds[w][0], p.bounds[w][1]
	for {
		select {
		case <-p.quit:
			return
		case <-p.starts[w]:
			for i := lo; i < hi; i++ {
				p.fn(i)
			}
			p.wg.Done()
		}
	}
}

// run executes fn(i) for every tap index across the workers and blocks until all
// have finished. The channel send/receive on each worker's start channel
// publishes the fn write (happens-before), and wg.Wait pairs with every Done.
func (p *tapPool) run(fn func(i int)) {
	p.fn = fn
	p.wg.Add(len(p.starts))
	for _, s := range p.starts {
		s <- struct{}{}
	}
	p.wg.Wait()
}

// close stops the workers. Idempotent.
func (p *tapPool) close() { p.quitOnce.Do(func() { close(p.quit) }) }
