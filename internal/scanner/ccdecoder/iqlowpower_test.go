package ccdecoder

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// emitLowPowerWindow drives observeIQPower through one full window of
// low-absolute-level IQ (dbfs well below iqLowPowerThresholdDbFS) and returns
// nothing — the caller inspects the log buffer. pwLowLogAt is zeroed first so the
// 30 s throttle never suppresses the line; only the lock gate can.
func emitLowPowerWindow(d *Decoder) {
	chunk := make([]complex64, 128)
	for i := range chunk {
		chunk[i] = complex(0.001, 0.001) // mean power 2e-6 ⇒ ~-57 dBFS
	}
	d.pwLowLogAt = time.Time{}
	d.observeIQPower(chunk) // primes pwWindowAt, no record yet
	d.pwWindowAt = d.pwWindowAt.Add(-2 * iqPowerWindow)
	d.observeIQPower(chunk) // window elapses → evaluate
}

// TestIQLowPowerSuppressedWhileLocked is the regression for the cosmetic
// "iq power very low" DEBUG spam: on a USRP/SDR at conservative gain the channel
// legitimately sits at ~-60 dBFS yet decodes TETRA cleanly, so the line fired
// every few seconds while locked. It must only fire when the decoder is NOT
// locked (a real antenna/gain/USB fault), never while locked and decoding.
func TestIQLowPowerSuppressedWhileLocked(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus := events.NewBus(8)
	defer bus.Close()
	d, err := New(Options{Bus: bus, IQ: &fakeIQSource{}, SampleRateHz: 48000, Log: log})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	setActiveSystem(d, "TestSys")

	// Not locked: the low-power hint is meaningful and must be emitted.
	d.locked.Store(false)
	emitLowPowerWindow(d)
	if !strings.Contains(buf.String(), "iq power very low") {
		t.Fatalf("unlocked low-power window did not log the hint; got:\n%s", buf.String())
	}

	// Locked and decoding: a low absolute level is fine and must NOT be logged.
	buf.Reset()
	d.locked.Store(true)
	emitLowPowerWindow(d)
	if strings.Contains(buf.String(), "iq power very low") {
		t.Errorf("low-power hint logged while locked (spam); got:\n%s", buf.String())
	}
}
