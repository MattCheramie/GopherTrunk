package composer

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
)

// newTestComposer builds a minimal composer for the LMS-wiring seam tests,
// carrying only the TETRALMSEqualizer opt-in through New's validation.
func newTestComposer(t *testing.T, lms bool) *Composer {
	t.Helper()
	c, err := New(Options{
		Bus:               events.NewBus(1),
		Devices:           &fakeDevices{},
		IQSampleRate:      2_400_000,
		TETRALMSEqualizer: lms,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// TestTETRALMSDefaultsOff pins the default: recordings.tetra_lms_equalizer
// unset ⇒ the TETRA voice receiver is byte-identical to before — no SymbolSink
// is wired, so the extractor never receives raw symbols and the training-
// sequence equalizer stays inert (the shipped lever remains the blind CMA in
// the receiver). This is the #764/#771 discipline: the unvalidated LMS lever is
// NOT on by default.
func TestTETRALMSDefaultsOff(t *testing.T) {
	c := newTestComposer(t, false)
	if c.tetraLMS {
		t.Fatal("tetraLMS should default to false")
	}
	ext := tetra.NewTrafficExtractor(0, func([]byte, []float32, uint8, uint8) {})
	opts := tetrarx.Options{}
	enableTETRALMS(c, ext, &opts)
	if opts.SymbolSink != nil {
		t.Fatal("with LMS off, no SymbolSink must be wired (must stay byte-identical)")
	}
}

// TestTETRALMSWiredWhenEnabled is the failing-first regression for the fix:
// recordings.tetra_lms_equalizer=true ⇒ the TETRA voice receiver stashes raw
// symbols into the extractor (SymbolSink → StashSymbols) so the per-burst
// training-sequence LMS equalizer has its input. Against the pre-fix code
// (no enableTETRALMS call / no plumbing) SymbolSink stays nil and this fails.
func TestTETRALMSWiredWhenEnabled(t *testing.T) {
	c := newTestComposer(t, true)
	if !c.tetraLMS {
		t.Fatal("tetraLMS should be true when TETRALMSEqualizer is set")
	}
	ext := tetra.NewTrafficExtractor(0, func([]byte, []float32, uint8, uint8) {})
	opts := tetrarx.Options{}
	enableTETRALMS(c, ext, &opts)
	if opts.SymbolSink == nil {
		t.Fatal("with LMS on, the receiver must wire a SymbolSink to feed StashSymbols")
	}
	// The sink must be safe to call — it forwards the raw symbols to the
	// extractor exactly as the receiver does on each block.
	opts.SymbolSink([]complex64{0, 1i, -1, -1i}, 0)
}
