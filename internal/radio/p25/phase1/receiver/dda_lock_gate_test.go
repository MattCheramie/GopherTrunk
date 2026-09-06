package receiver

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// The decision-directed AFC (issue #402) once committed a self-consistent
// wrong offset with nothing downstream able to tell it, so a lock probe from
// the framer now gates the handoff and reverts it. These tests drive a
// synthesized C4FM stream with a carrier offset through the receiver one
// second at a time and script the probe.

const lockGateSeconds = 10

// runLockGate feeds lockGateSeconds of random dibits (with a 300 Hz carrier
// offset for the AFC to track) through a DDA-enabled receiver in one-second
// chunks, consulting probeAt(second) as the lock probe for that chunk.
func runLockGate(t *testing.T, probeAt func(second int) bool) *Receiver {
	t.Helper()
	dibits := randDibits(7, lockGateSeconds*4800)
	iq := demod.ModulateP25C4FM(dibits, loopSR, loopDev)
	iq = demod.ApplyImpairments(iq, loopSR, demod.Impairments{FreqOffsetHz: 300})
	second := 0
	r := New(Options{
		SampleRateHz:              loopSR,
		DeviationHz:               loopDev,
		DemodMode:                 DemodC4FM,
		EnableDecisionDirectedAFC: true,
		LockProbe:                 func() bool { return probeAt(second) },
		DibitSink:                 func([]uint8, int) {},
	})
	perSecond := int(loopSR)
	for i := 0; i < len(iq); i += perSecond {
		end := i + perSecond
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
		second++
	}
	return r
}

func TestDDAHandsOffWithoutProbe(t *testing.T) {
	// Sanity: the internal gates alone still commit on a clean stream, so
	// the probe tests below are exercising the probe and not a dead path.
	dibits := randDibits(7, lockGateSeconds*4800)
	iq := demod.ModulateP25C4FM(dibits, loopSR, loopDev)
	r := New(Options{
		SampleRateHz:              loopSR,
		DeviationHz:               loopDev,
		DemodMode:                 DemodC4FM,
		EnableDecisionDirectedAFC: true,
		DibitSink:                 func([]uint8, int) {},
	})
	r.Process(iq)
	if !r.DDAActive() {
		t.Fatal("DDA never handed off on a clean synthesized stream with no probe")
	}
}

func TestDDAHandoffWaitsForLock(t *testing.T) {
	r := runLockGate(t, func(int) bool { return false })
	if r.DDAActive() {
		t.Fatal("DDA handed off while the lock probe never reported lock")
	}
	if r.DDALockReverts() != 0 {
		t.Fatalf("DDALockReverts = %d with no handoff", r.DDALockReverts())
	}
}

func TestDDAHandsOffOnLock(t *testing.T) {
	r := runLockGate(t, func(int) bool { return true })
	if !r.DDAActive() {
		t.Fatal("DDA did not hand off while the lock probe reported lock throughout")
	}
	if r.DDALockReverts() != 0 {
		t.Fatalf("DDALockReverts = %d on a held lock", r.DDALockReverts())
	}
}

func TestDDARevertsWhenLockDrops(t *testing.T) {
	// Locked for the first four seconds (long enough to hand off), then the
	// framer stops reporting progress: ddaLockLossProbes false polls must
	// revert the receiver to CoarseAFC-alone.
	r := runLockGate(t, func(second int) bool { return second < 4 })
	if r.DDAActive() {
		t.Fatal("DDA still active after the lock probe went false for the rest of the stream")
	}
	if r.DDALockReverts() < 1 {
		t.Fatalf("DDALockReverts = %d, want >= 1", r.DDALockReverts())
	}
}
