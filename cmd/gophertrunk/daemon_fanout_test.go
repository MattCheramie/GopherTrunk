package main

import (
	"sync"
	"testing"
)

// rawFrameRecorder is a test stub that implements both the
// composer.PCMSink shape (WritePCM) and the rawFrameSink shape
// (WriteRawFrame). Plays the role of the real *voice.Recorder in
// production fanouts.
type rawFrameRecorder struct {
	mu     sync.Mutex
	pcm    map[string][][]int16
	frames map[string][][]byte
}

func newRawFrameRecorder() *rawFrameRecorder {
	return &rawFrameRecorder{
		pcm:    make(map[string][][]int16),
		frames: make(map[string][][]byte),
	}
}

func (r *rawFrameRecorder) WritePCM(serial string, samples []int16) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := append([]int16(nil), samples...)
	r.pcm[serial] = append(r.pcm[serial], cp)
	return nil
}

func (r *rawFrameRecorder) WriteRawFrame(serial string, frame []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := append([]byte(nil), frame...)
	r.frames[serial] = append(r.frames[serial], cp)
	return nil
}

func (r *rawFrameRecorder) framesFor(serial string) [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([][]byte, len(r.frames[serial]))
	copy(out, r.frames[serial])
	return out
}

// pcmOnlySink implements only WritePCM. Stands in for sinks like the
// tone-out detector, the live player, and the audio publisher — none
// of which consume raw IMBE / AMBE frames.
type pcmOnlySink struct {
	mu      sync.Mutex
	written int
}

func (s *pcmOnlySink) WritePCM(_ string, _ []int16) error {
	s.mu.Lock()
	s.written++
	s.mu.Unlock()
	return nil
}

// TestFanoutSinkWriteRawFrameReachesRawFrameSinks is the regression
// guard for issue #356. Before the fix, fanoutSink implemented only
// WritePCM, so the voice composer chains' rs, _ := c.sink.(rawFrameSink)
// type assertion failed against any multi-sink production wiring (e.g.
// recorder + audio publisher). rs stayed nil, every IMBE / AMBE frame
// was dropped at the `if rs == nil { return }` short-circuit, and
// .raw files came out 0 bytes / .wav files came out 44 bytes (header
// only). This test asserts that fanoutSink.WriteRawFrame fans the
// frame out to every contained sink that implements the rawFrameSink
// shape and silently skips the ones that don't.
func TestFanoutSinkWriteRawFrameReachesRawFrameSinks(t *testing.T) {
	rec := newRawFrameRecorder()
	pcm := &pcmOnlySink{}
	fanout := fanoutSink{rec, pcm}

	frame := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b}
	if err := fanout.WriteRawFrame("VOICE-1", frame); err != nil {
		t.Fatalf("WriteRawFrame: %v", err)
	}

	got := rec.framesFor("VOICE-1")
	if len(got) != 1 {
		t.Fatalf("recorder frames = %d, want 1", len(got))
	}
	if string(got[0]) != string(frame) {
		t.Errorf("recorder frame = %x, want %x", got[0], frame)
	}
	if pcm.written != 0 {
		t.Errorf("pcm-only sink got %d WritePCM calls; WriteRawFrame must not call WritePCM on it", pcm.written)
	}
}

// TestFanoutSinkWriteRawFrameNoRawSinks confirms that a fanout with
// no rawFrameSink-implementing members returns nil without error.
// Mirrors the "every downstream is PCM-only" edge of the dispatcher.
func TestFanoutSinkWriteRawFrameNoRawSinks(t *testing.T) {
	a, b := &pcmOnlySink{}, &pcmOnlySink{}
	fanout := fanoutSink{a, b}

	if err := fanout.WriteRawFrame("VOICE-1", []byte{0xaa}); err != nil {
		t.Errorf("WriteRawFrame with no raw sinks should be nil; got %v", err)
	}
	if a.written != 0 || b.written != 0 {
		t.Errorf("WriteRawFrame must not call WritePCM on any sink; got a=%d b=%d", a.written, b.written)
	}
}

// TestFanoutSinkWritePCMUnchanged is the negative-space guard: the
// existing WritePCM fanout behaviour must remain unchanged by the
// WriteRawFrame addition.
func TestFanoutSinkWritePCMUnchanged(t *testing.T) {
	rec := newRawFrameRecorder()
	pcm := &pcmOnlySink{}
	fanout := fanoutSink{rec, pcm}

	samples := []int16{1, 2, 3, 4}
	if err := fanout.WritePCM("VOICE-1", samples); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	if pcm.written != 1 {
		t.Errorf("pcm-only sink WritePCM count = %d, want 1", pcm.written)
	}
	rec.mu.Lock()
	gotPCM := len(rec.pcm["VOICE-1"])
	rec.mu.Unlock()
	if gotPCM != 1 {
		t.Errorf("recorder WritePCM count = %d, want 1", gotPCM)
	}
}

// errAwareRecorder is a test stub that, like the real *voice.Recorder,
// implements the error-aware raw-frame shape (WriteRawFrameWithErrors) on
// top of WritePCM/WriteRawFrame. It records the corrected-bit count it was
// handed so tests can assert the count survives the fanout.
type errAwareRecorder struct {
	mu        sync.Mutex
	frames    [][]byte
	errCounts []int
	plainHits int // WriteRawFrame (no error count) calls
}

func (r *errAwareRecorder) WritePCM(string, []int16) error { return nil }

func (r *errAwareRecorder) WriteRawFrame(serial string, frame []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plainHits++
	r.frames = append(r.frames, append([]byte(nil), frame...))
	r.errCounts = append(r.errCounts, -1) // sentinel: arrived without a count
	return nil
}

func (r *errAwareRecorder) WriteRawFrameWithErrors(serial string, frame []byte, correctedBits int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frames = append(r.frames, append([]byte(nil), frame...))
	r.errCounts = append(r.errCounts, correctedBits)
	return nil
}

// TestFanoutSinkSatisfiesErrAwareShape pins that fanoutSink implements the
// error-aware raw-frame interface the P25 Phase 1 / DMR voice chains probe
// for via c.sink.(errAwareRawSink). If this assertion fails the chains
// silently fall back to the plain WriteRawFrame and the IMBE decoder's
// adaptive smoothing never receives a non-zero channel error rate — i.e.
// smoothing is inert in the daemon even though it works in unit tests that
// drive the recorder directly. The interface is redeclared here (it is
// unexported in the composer package) to match that probe exactly.
func TestFanoutSinkSatisfiesErrAwareShape(t *testing.T) {
	var f any = fanoutSink{}
	if _, ok := f.(interface {
		WriteRawFrameWithErrors(string, []byte, int) error
	}); !ok {
		t.Fatal("fanoutSink does not implement WriteRawFrameWithErrors — " +
			"the voice chains' errAwareRawSink assertion will fail and adaptive smoothing will be inert in the daemon")
	}
}

// TestFanoutSinkForwardsErrorCounts asserts WriteRawFrameWithErrors hands
// the corrected-bit count to error-aware sinks while still delivering the
// frame (without a count) to plain raw-frame sinks.
func TestFanoutSinkForwardsErrorCounts(t *testing.T) {
	ea := &errAwareRecorder{}
	plain := newRawFrameRecorder()
	pcm := &pcmOnlySink{}
	fanout := fanoutSink{ea, plain, pcm}

	frame := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b}
	if err := fanout.WriteRawFrameWithErrors("VOICE-1", frame, 7); err != nil {
		t.Fatalf("WriteRawFrameWithErrors: %v", err)
	}

	// Error-aware sink got the count.
	ea.mu.Lock()
	gotCounts := append([]int(nil), ea.errCounts...)
	gotPlainHits := ea.plainHits
	ea.mu.Unlock()
	if len(gotCounts) != 1 || gotCounts[0] != 7 {
		t.Fatalf("error-aware sink counts = %v, want [7]", gotCounts)
	}
	if gotPlainHits != 0 {
		t.Errorf("error-aware sink took the plain path %d time(s); want the WithErrors path", gotPlainHits)
	}

	// Plain raw-frame sink still got the frame (count dropped, frame kept).
	if got := plain.framesFor("VOICE-1"); len(got) != 1 || string(got[0]) != string(frame) {
		t.Errorf("plain sink frames = %x, want one frame %x", got, frame)
	}
	// PCM-only sink is untouched by raw-frame writes.
	if pcm.written != 0 {
		t.Errorf("pcm-only sink got %d WritePCM calls; raw-frame write must not call WritePCM", pcm.written)
	}
}
