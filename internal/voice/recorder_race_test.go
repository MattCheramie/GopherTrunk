package voice

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// TestRecorderWriteFinalizeRaceNoDrainCoordination is the failing-first
// regression for the write-after-close hazard (finding C1): the composer voice
// chains call WriteRawFrame on their own goroutines while the recorder's Run
// goroutine finalizes and closes the session on KindCallEnd. r.mu only guards
// the sessions map, not the per-session WAV/raw writers, so before the
// per-session lock a frame in flight could write to a WAV the finalizer was
// closing — a data race the -race detector flags, and in the worst case a
// write to a closed file.
//
// This test drives that exact ordering WITHOUT EnableDrainCoordination (the
// opt-in mechanism that previously masked it), so it fails under `go test
// -race` against the pre-fix recorder and passes once WritePCM/writeRawFrame
// and finalizeLocked/close share the session lock. It must run cleanly many
// times, so it repeats the overlap.
func TestRecorderWriteFinalizeRaceNoDrainCoordination(t *testing.T) {
	const frameSize = 11 // loudVocoder.FrameSize()
	frame := make([]byte, frameSize)

	const writers = 6
	for iter := 0; iter < 40; iter++ {
		r, _ := newDrainRecorder(t) // deliberately NOT drain-coordinated
		cs := drainCallStart()
		r.handleStart(cs)

		// Prime a frame so a session with open files exists.
		_ = r.WriteRawFrame(cs.DeviceSerial, frame)

		var wg sync.WaitGroup
		var progress atomic.Int64 // total writes started across writers
		go2 := make(chan struct{})
		for w := 0; w < writers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-go2
				for i := 0; i < 2000; i++ {
					progress.Add(1)
					// Frames in flight during the finalizer's close must not
					// touch the WAV/raw writers unsynchronised; once the session
					// is closed these return nil.
					_ = r.WriteRawFrame(cs.DeviceSerial, frame)
				}
			}()
		}
		// Release the writers, then spin until they are demonstrably mid-loop
		// before finalizing — so writes are genuinely in flight while handleEnd
		// closes the session (the write-after-close window).
		close(go2)
		for progress.Load() < int64(writers) {
			runtime.Gosched()
		}
		r.handleEnd(drainCallEnd(cs))
		wg.Wait()

		if r.HasSession(cs.DeviceSerial) {
			t.Fatal("session not finalized after CallEnd")
		}
	}
}
