package receiver

import (
	"io"
	"log/slog"
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
)

// buildSoftSBStream assembles a repeated TETRA synchronisation burst
// (BSCH SYNC + STS + broadcast + BNCH SYSINFO) as a dibit stream, with a
// warmup so the receiver loops converge.
func buildSoftSBStream(repeats int, cc6 uint8, mcc, mnc, la uint16) []uint8 {
	syncInfo := make([]byte, 60)
	put := func(off, n int, v uint32) {
		for i := 0; i < n; i++ {
			syncInfo[off+i] = byte((v >> uint(n-1-i)) & 1)
		}
	}
	put(4, 6, uint32(cc6))
	put(31, 10, uint32(mcc))
	put(41, 14, uint32(mnc))
	bsch := tetra.TetraBitsToDibits(tetra.EncodeBSCH(syncInfo))

	ext := tetra.ExtendedColourCode(mcc, mnc, cc6)
	payload := make([]byte, 11)
	payload[3] = byte((la >> 6) & 0xFF)
	payload[4] = byte((la & 0x3F) << 2)
	pdu := tetra.PDU{Disc: tetra.DiscMLE, Type: uint8(tetra.MLESystemInfo), Payload: payload}
	bnch := tetra.TetraBitsToDibits(tetra.EncodeSCHHD(pduToType1BitsTETRA(pdu, 124), ext))

	var out []uint8
	for i := 0; i < 600; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, bsch...)
		out = append(out, tetra.SyncTrainingDibits()...)
		out = append(out, make([]uint8, 15)...)
		out = append(out, bnch...)
		out = append(out, make([]uint8, 40)...) // inter-burst gap
	}
	return out
}

// runSoftSB modulates the SB-burst dibit stream, adds complex Gaussian
// noise, runs it through the receiver (soft path on iff soft==true) into
// a TETRA ControlChannel, and reports whether it locked.
func runSoftSB(t *testing.T, dibits []uint8, noise float64, seed int64, soft bool) bool {
	t.Helper()
	const sps, span, alpha, rate = 8, 8, 0.35, 144_000.0
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/4)
	rng := rand.New(rand.NewSource(seed))
	for i := range iq {
		iq[i] += complex(float32(rng.NormFloat64()*noise), float32(rng.NormFloat64()*noise))
	}

	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := tetra.New(tetra.Options{
		Bus: bus, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName: "soft", FrequencyHz: 412_000_000,
	})
	cc.SetChannelCoding(tetra.ChannelCodingOn)

	opts := Options{
		SampleRateHz: rate, ClockMode: ClockGardner, GardnerGain: 0.005, EnableAFC: true,
		DibitSink: func(d []uint8, base int) { cc.Process(d, base) },
	}
	if soft {
		opts.SoftSink = func(diffs []complex64, base int) { cc.StashSoft(diffs, base) }
	}
	rx := New(opts)
	for i := 0; i < len(iq); i += 4096 {
		e := i + 4096
		if e > len(iq) {
			e = len(iq)
		}
		rx.Process(iq[i:e])
	}

	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				return true
			}
			continue
		default:
		}
		return false
	}
}

// TestSoftDecisionLocksWhereHardFails drives noisy SB bursts through the
// full production receiver (channel decode via the SB-burst path) at a
// noise level where the soft-decision path closes the BSCH/BNCH FEC far
// more often than the hard path — the end-to-end demonstration of the
// soft-decision coding gain (issue #553 follow-up). It is statistical
// (lock rate across fixed seeds) because a single noisy run is binary
// and seed-sensitive near the FEC threshold.
func TestSoftDecisionLocksWhereHardFails(t *testing.T) {
	dibits := buildSoftSBStream(30, 0x2D, 0x0CE, 0x1234, 0x2B7)
	const (
		noise = 0.42
		N     = 16
	)
	hardLocks, softLocks := 0, 0
	for seed := int64(0); seed < N; seed++ {
		if runSoftSB(t, dibits, noise, seed, false) {
			hardLocks++
		}
		if runSoftSB(t, dibits, noise, seed, true) {
			softLocks++
		}
	}
	t.Logf("noise=%.2f over %d seeds: hard locked %d, soft locked %d", noise, N, hardLocks, softLocks)
	if softLocks <= hardLocks {
		t.Errorf("soft (%d/%d) did not beat hard (%d/%d)", softLocks, N, hardLocks, N)
	}
	if softLocks < hardLocks+3 || softLocks < 11 {
		t.Errorf("soft-decision gain too small: hard %d, soft %d of %d", hardLocks, softLocks, N)
	}
}

// TestSoftDecisionCleanStillLocks confirms the soft path is correct on a
// clean signal too (no regression vs hard).
func TestSoftDecisionCleanStillLocks(t *testing.T) {
	dibits := buildSoftSBStream(8, 0x11, 0x123, 0x2345, 0x0AB)
	if !runSoftSB(t, dibits, 0, 1, true) {
		t.Error("soft path failed to lock on a clean SB burst")
	}
}
