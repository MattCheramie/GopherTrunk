package soapyremote

import (
	"bytes"
	"log/slog"
	"math/rand"
	"strings"
	"testing"
)

// anchorFixture calibrates a combiner with branch 1 as the louder (and hence
// anchored) branch: ch1 carries the signal, ch0 a quieter rotated copy.
func anchorFixture(t *testing.T, m *mrcCombiner) (ch0, ch1 []complex64) {
	t.Helper()
	rng := rand.New(rand.NewSource(7))
	ch1 = mrcSignal(rng, mrcTestWindow, 0.4)
	ch0 = scaleC(ch1, complex(0, 0.2)) // −6 dB, +90°
	m.combine(twoBranchPayload(ch0, ch1), len(ch1))
	if h := m.health(); h.refIdx != 1 {
		t.Fatalf("fixture: reference = %d, want 1 (the louder branch)", h.refIdx)
	}
	if !m.cal.Calibrated() {
		t.Fatal("fixture: combiner did not calibrate")
	}
	return ch0, ch1
}

// TestMRCCombinerKeepsReferenceAcrossRearm is the failing-first regression for
// the 19 Aug 08:58 field event: a cc-hunt retune (requestRecalibrate) used to
// un-latch the anchor to -1, and the next datagram's bare argmax re-pick
// flipped the reference mid-stream — an unlogged ±180° phase + several dB
// amplitude discontinuity into the differential decoder. A retune invalidates
// the GAIN (new LO lock), never the choice of anchor: after a rearm the
// reference must stay where it was, even when the other branch is momentarily
// louder (the field condition after the operator's antenna swap).
func TestMRCCombinerKeepsReferenceAcrossRearm(t *testing.T) {
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	anchorFixture(t, m)

	m.requestRecalibrate()
	// Post-retune datagrams: branch 0 now ~6 dB louder than the incumbent
	// reference — far below the dead-incumbent margin, so the anchor must hold.
	rng := rand.New(rand.NewSource(8))
	for i := 0; i < 8; i++ {
		ch0 := mrcSignal(rng, 256, 0.4)
		ch1 := scaleC(ch0, complex(0.2, 0))
		m.combine(twoBranchPayload(ch0, ch1), 256)
		if h := m.health(); h.refIdx != 1 {
			t.Fatalf("reference flipped to %d after a rearm (datagram %d) — a retune must not move the phase anchor", h.refIdx, i)
		}
	}
}

// TestMRCCombinerRearmEscapesDeadReference pins the escape that keeps the kept
// anchor safe: when the incumbent reference is genuinely dead (>20 dB down)
// after a rearm, the uncalibrated dead-incumbent path still moves the anchor
// within mrcRefDeadDatagrams datagrams.
func TestMRCCombinerRearmEscapesDeadReference(t *testing.T) {
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	anchorFixture(t, m)

	m.requestRecalibrate()
	rng := rand.New(rand.NewSource(9))
	for i := 0; i < mrcRefDeadDatagrams+1; i++ {
		ch0 := mrcSignal(rng, 256, 0.4)
		dead := mrcNoise(rng, make([]complex64, 256), 1e-4) // ~ -80 dBFS: >20 dB down
		m.combine(twoBranchPayload(ch0, dead), 256)
	}
	if h := m.health(); h.refIdx != 0 {
		t.Fatalf("reference = %d, want 0: a genuinely dead incumbent must still be escaped after a rearm", h.refIdx)
	}
}

// TestMRCCombinerSetSampleRateKeepsReference: a stream-rate change re-sizes the
// estimation window and drops the gain estimate, but says nothing about which
// antenna is better — the anchor must survive it.
func TestMRCCombinerSetSampleRateKeepsReference(t *testing.T) {
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	anchorFixture(t, m)

	m.setSampleRate(2_400_000)
	if h := m.health(); h.refIdx != 1 {
		t.Fatalf("reference = %d after setSampleRate, want 1 (kept)", h.refIdx)
	}
}

// TestMRCCombinerDegeneratePayloadDoesNotMoveAnchor is failing-first for the
// short-payload stomp: a datagram that delivers only one channel used to
// unconditionally rewrite refIdx to 0, silently moving a held anchor. It must
// pass the payload through without touching the anchor.
func TestMRCCombinerDegeneratePayloadDoesNotMoveAnchor(t *testing.T) {
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	_, ch1 := anchorFixture(t, m)

	out := m.combine(encodeCS16(ch1[:64]), 64) // single-channel payload
	if !approxEqualC(out, ch1[:64], 1e-3) {
		t.Fatal("degenerate payload was not passed through verbatim")
	}
	if h := m.health(); h.refIdx != 1 {
		t.Fatalf("reference = %d after a degenerate payload, want 1 (a short datagram must not move the anchor)", h.refIdx)
	}
}

// TestMRCCombinerLogsReferenceChange: every anchor movement is a decode
// discontinuity, so both the initial latch and a dead-incumbent move must be
// visible in the log (the 19 Aug field flip was silent — only inferable from
// two 30 s health lines).
func TestMRCCombinerLogsReferenceChange(t *testing.T) {
	buf := &bytes.Buffer{}
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	m.log = slog.New(slog.NewTextHandler(buf, nil))

	rng := rand.New(rand.NewSource(10))
	live := mrcSignal(rng, 256, 0.4)
	dead := mrcNoise(rng, make([]complex64, 256), 1e-4)

	m.combine(twoBranchPayload(dead, live), 256)
	if !strings.Contains(buf.String(), "MRC phase anchor latched") {
		t.Fatalf("initial anchor latch was not logged\n%s", buf.String())
	}
	buf.Reset()

	for i := 0; i < mrcRefDeadDatagrams; i++ {
		m.combine(twoBranchPayload(live, dead), 256) // incumbent (1) now dead
	}
	if h := m.health(); h.refIdx != 0 {
		t.Fatalf("fixture: anchor did not move off the dead incumbent (ref=%d)", h.refIdx)
	}
	if !strings.Contains(buf.String(), "MRC phase anchor moved") {
		t.Fatalf("anchor move was not logged\n%s", buf.String())
	}
}
