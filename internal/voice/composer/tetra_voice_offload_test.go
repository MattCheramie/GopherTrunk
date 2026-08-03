package composer

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

// TestTETRASlotOwnerEnqueueNeverBlocks pins the core property of running the
// TETRA per-call vocoder off the shared demux goroutine: enqueue must NEVER block
// the caller, even when the worker is not draining. The demux goroutine that
// drains the carrier's post-DDC IQ calls enqueue for every routed burst; if that
// blocked on a slow/stuck vocoder the goroutine would stop reading IQ, the
// upstream voice-tap buffer would overflow, and every concurrent slot would
// garble (the multislot garble this commit fixes). Fail-first: replace enqueue's
// non-blocking select with a plain `o.jobs <- job` and this test deadlocks/times
// out once the queue fills.
func TestTETRASlotOwnerEnqueueNeverBlocks(t *testing.T) {
	o := &tetraSlotOwner{serial: "s", jobs: make(chan tetraDecodeJob, 4)}

	const n = 20 // > queue depth, with no worker draining
	done := make(chan struct{})
	go func() {
		for i := 0; i < n; i++ {
			o.enqueue([]byte{byte(i)}, nil)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("enqueue blocked with a full queue — the demux goroutine would stall on a slow vocoder")
	}

	wantDrops := n - cap(o.jobs)
	if got := int(o.vocoderDrops.Load()); got != wantDrops {
		t.Fatalf("vocoderDrops = %d, want %d (enqueued %d into a %d-deep queue)", got, wantDrops, n, cap(o.jobs))
	}
	if len(o.jobs) != cap(o.jobs) {
		t.Fatalf("queued = %d, want the queue full at %d", len(o.jobs), cap(o.jobs))
	}
}

// TestTETRASlotOwnerEnqueueCopiesSlices guards the copy-on-enqueue contract: the
// TrafficExtractor reuses its frame/soft backing arrays for the next burst, so a
// job handed to the worker must not alias caller memory that is about to be
// overwritten. Fail-first: drop the append-copies in enqueue and the mutated
// bytes below leak into the queued job.
func TestTETRASlotOwnerEnqueueCopiesSlices(t *testing.T) {
	o := &tetraSlotOwner{serial: "s", jobs: make(chan tetraDecodeJob, 2)}
	frame := []byte{1, 2, 3}
	soft := []float32{1, 2, 3}
	o.enqueue(frame, soft)
	// Simulate the extractor overwriting its buffers for the next burst.
	frame[0], soft[0] = 9, 9
	job := <-o.jobs
	if job.frame[0] != 1 {
		t.Fatalf("queued frame aliases caller buffer: got %d, want 1", job.frame[0])
	}
	if job.softType5[0] != 1 {
		t.Fatalf("queued soft aliases caller buffer: got %v, want 1", job.softType5[0])
	}
}

// TestDrainTETRAIQProcessesBufferedTail pins the solo voice chain's teardown
// drain: when a call is cancelled (typically a control-channel D-RELEASE, which
// bypasses the voice hangtime), the IQ still buffered in the chain's channel must
// be demodulated before the chain exits, not dropped. Dropping it truncated the
// tail of the transmission — the "recordings end a beat early" report. Fail-first:
// before the fix runTETRAVoiceChain returned immediately on ctx.Done with no
// drain, so this buffered tail was lost.
func TestDrainTETRAIQProcessesBufferedTail(t *testing.T) {
	ch := make(chan []complex64, 8)
	const n = 5
	for i := 0; i < n; i++ {
		ch <- []complex64{complex(float32(i), 0)}
	}

	var got []float32
	done := make(chan struct{})
	go func() {
		drainTETRAIQ(ch, func(iq []complex64) { got = append(got, real(iq[0])) })
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("drainTETRAIQ blocked instead of returning once the buffer emptied")
	}

	if len(got) != n {
		t.Fatalf("drained %d buffered chunks, want %d (tail dropped)", len(got), n)
	}
	for i := 0; i < n; i++ {
		if got[i] != float32(i) {
			t.Fatalf("drain reordered chunks: got %v", got)
		}
	}

	// A closed channel makes drain return immediately (chain teardown when the
	// carrier IQ stream ends).
	closed := make(chan []complex64)
	close(closed)
	drainTETRAIQ(closed, func([]complex64) { t.Fatal("processed from a closed empty channel") })
}

// TestTETRAOwnerWorkerDrainsAndExits confirms the worker consumes its backlog and
// exits promptly on ctx cancel (call end). Garbage frames decode to no speech, so
// the worker loop is exercised without needing a real TCH/S bitstream or sink.
func TestTETRAOwnerWorkerDrainsAndExits(t *testing.T) {
	c := &Composer{log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	o := &tetraSlotOwner{
		serial: "s",
		jobs:   make(chan tetraDecodeJob, 8),
		bt:     &boundaryTracker{c: c, grantTG: 0},
	}
	for i := 0; i < 8; i++ {
		o.enqueue([]byte{byte(i)}, nil) // garbage → decodeTETRASpeech is a fast no-op
	}

	ctx, cancel := context.WithCancel(context.Background())
	workerDone := make(chan struct{})
	go func() { c.tetraOwnerWorker(ctx, o); close(workerDone) }()

	cancel()
	select {
	case <-workerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("vocoder worker did not exit after ctx cancel")
	}
	if n := len(o.jobs); n != 0 {
		t.Fatalf("worker exited without draining its backlog: %d jobs left", n)
	}
}
