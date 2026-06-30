package airspy

import (
	"sync"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// TestProcessRawConcurrent reproduces the macOS crash from issue: the darwin USB
// transport runs one bulk-IN reaper goroutine per ring slot
// (usb.DefaultRingBuffers = 32), all of which call onPacket -> processRaw on the
// single shared iqConverter. Linux uses a single reaper so the bug never showed
// there, but on macOS the concurrent goroutines race on the converter's mutable
// filter state (qpos, phase, avg, iwin, qline), eventually reading qline[12]
// (out of a length-12 array) and panicking with
// "index out of range [12] with length 12".
//
// Without the converter mutex this fails: under `go test -race` the detector
// flags the concurrent qpos/avg access, and the recover() guard catches the
// index-out-of-range panic even without the race detector. With the mutex it
// passes.
func TestProcessRawConcurrent(t *testing.T) {
	c := newIQConverter()

	// A full-size bulk-IN payload, matching the real URB length the transport
	// hands to onPacket. Contents are arbitrary real samples.
	buf := make([]byte, usb.DefaultBufferLen)
	for i := range buf {
		buf[i] = byte(i)
	}

	const (
		goroutines = usb.DefaultRingBuffers // 32, as the macOS transport spawns
		iterations = 200
	)

	var (
		wg       sync.WaitGroup
		panicMu  sync.Mutex
		panicVal any
	)
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicMu.Lock()
					if panicVal == nil {
						panicVal = r
					}
					panicMu.Unlock()
				}
			}()
			for i := 0; i < iterations; i++ {
				c.processRaw(buf)
			}
		}()
	}
	wg.Wait()

	if panicVal != nil {
		t.Fatalf("processRaw panicked under concurrent delivery: %v", panicVal)
	}
}
