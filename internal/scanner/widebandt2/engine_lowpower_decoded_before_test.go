package widebandt2

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

// TestLowPowerHintAfterEarlierDecodeIsNotMistuned is the #1321 regression for
// the low-power WARN's diagnosis on a weak cell. The reporter's weakest TETRA
// cell (about -55 dBFS on a small indoor antenna, wideband input -40 dBFS)
// locked, fell silent, and locked again every minute or two, and drew 123
// "channel iq power very low — this carrier is likely outside the captured
// passband or tuned to the wrong frequency" WARNs in 11 minutes. A channel
// that has decoded on this tap is on frequency and inside the passband by
// construction: once its decodes stop for longer than the grace, the WARN must
// say the carrier is too weak to hold, not that it is mistuned.
func TestLowPowerHintAfterEarlierDecodeIsNotMistuned(t *testing.T) {
	const n = 4096
	ec := &engineChannel{
		freqHz:   390_637_500,
		sysName:  "Cell4",
		protoTag: "tetra",
		// Three sync bursts decode before the first window closes, then the
		// cell fades: the lifetime count never moves again.
		decoded: func() uint64 { return 3 },
	}

	// Windows are 6 s apart: the decode lands in window 1, the 15 s grace
	// covers windows 2 and 3, and window 4 (18 s after the decode) warns.
	recs := runDiagWindows(t, ec, 4, quietIQ(n), loudIQ(n))
	var warns []capturedRecord
	for _, r := range recs {
		if r.level == slog.LevelWarn && strings.Contains(r.msg, "iq power very low") {
			warns = append(warns, r)
		}
	}
	if len(warns) != 1 {
		t.Fatalf("low-power WARNs = %d, want 1 (only once the decode grace has run out)", len(warns))
	}
	w := warns[0]
	if strings.Contains(w.msg, "outside the captured passband") || strings.Contains(w.msg, "wrong frequency") {
		t.Errorf("WARN blames tuning on a channel that has decoded on this tap: %q", w.msg)
	}
	if !strings.Contains(w.msg, "too weak") {
		t.Errorf("WARN msg = %q, want the weak-signal hint", w.msg)
	}
	ago, ok := w.attrs["last_decode_ago"].(time.Duration)
	if !ok || ago != 18*time.Second {
		t.Errorf("last_decode_ago = %v, want 18s", w.attrs["last_decode_ago"])
	}
}
