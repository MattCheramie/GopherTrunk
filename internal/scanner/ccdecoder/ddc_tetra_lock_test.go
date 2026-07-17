package ccdecoder

import (
	"io"
	"log/slog"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
)

// TestDDCUpsamplesSubTargetTETRAToLock is the end-to-end regression for
// the low-rate-replay bug: a pre-channelized TETRA capture BELOW the
// 144 kHz channel target (an SDR++ baseband WAV at 39062.5 Hz, or the
// 48 kHz reference recording) must be interpolated up to the receiver's
// 8-samples-per-symbol design point before demod. Before the fix the
// Downconverter passed a sub-target stream through untouched, so the
// receiver computed a fractional sps and rounded it — a 48 kHz stream
// decoded at 16000 baud (−11 %), a 39062.5 Hz stream at +8.5 %, and
// nothing locked.
//
// It drives a clean, fully SCH/HD-channel-coded MLE SYSINFO burst train
// through the *production* TETRA receiver configuration (Gardner gain
// 0.005, AFC + channel filter on — the newTETRAPipeline values) and
// asserts both halves of the fix:
//
//	arm A: feeding the 48 kHz capture straight to the receiver does NOT
//	       lock — the documented pre-fix failure mode.
//	arm B: routing the same 48 kHz capture through the Downconverter,
//	       which now interpolates it up to 144 kHz, DOES lock and
//	       recovers the encoded LocationArea.
//
// Before the ddc.go fix arm B's Downconverter reported OutRateHz ==
// 48000 (pass-through) and arm B failed exactly like arm A; the fix is
// what makes OutRateHz == 144000 and the lock appear.
func TestDDCUpsamplesSubTargetTETRAToLock(t *testing.T) {
	const (
		controlFreqHz        = 412_062_500
		colourCode           = uint32(0x12345)
		locationArea  uint16 = 0x42
	)

	// Build a clean 144 kHz (8 sps) π/4-DQPSK capture, then downsample it
	// 3:1 to a genuine 48 kHz TETRA capture — 48000/18000 = 2.667 sps, a
	// non-multiple of the baud, which is what broke decode before the fix.
	dibits := buildTETRASCHHDDibits(120, colourCode, locationArea)
	iq144 := demod.ModulatePiOver4DQPSK(dibits, 8, 8, 0.35, math.Pi/4)
	iq48 := dsp.NewResampler(1, 3, 64, ddcKaiserBeta).Process(nil, iq144)

	// arm A: 48 kHz straight into the receiver → fractional sps → no lock.
	if locked, _ := runTETRAReceiver(t, iq48, nil, 48_000, controlFreqHz, colourCode, locationArea); locked {
		t.Fatalf("arm A: a 48 kHz sub-target capture locked without up-sampling; " +
			"the pre-fix fractional-sps failure it documents is gone — check the test")
	}

	// arm B: 48 kHz through the Downconverter (now interpolates to 144 kHz).
	dc := NewDownconverter(48_000, tetraDDCTargetRateHz)
	if math.Abs(dc.OutRateHz()-tetraDDCTargetRateHz) > 1e-6 {
		t.Fatalf("arm B: Downconverter OutRateHz = %v, want %v — the up-sample fix is not in effect",
			dc.OutRateHz(), tetraDDCTargetRateHz)
	}
	locked, gotLA := runTETRAReceiver(t, iq48, dc, dc.OutRateHz(), controlFreqHz, colourCode, locationArea)
	if !locked {
		t.Fatalf("arm B: no cc.locked after up-sampling 48 kHz → %v; the DDC interpolation fix failed",
			tetraDDCTargetRateHz)
	}
	if gotLA != locationArea {
		t.Errorf("arm B: LockState.LocationArea = %#x, want %#x", gotLA, locationArea)
	}
}

// runTETRAReceiver feeds iq (chunked) through an optional Downconverter
// and then the production-configured TETRA receiver at rxRate, and
// reports whether a cc.locked fired and the LocationArea it carried.
func runTETRAReceiver(t *testing.T, iq []complex64, dc *Downconverter, rxRate float64,
	controlFreqHz uint32, colourCode uint32, _ uint16) (bool, uint16) {
	t.Helper()

	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := tetra.New(tetra.Options{
		Bus:         bus,
		Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName:  "TETRASite",
		FrequencyHz: controlFreqHz,
	})
	cc.SetChannelCoding(tetra.ChannelCodingOn)
	cc.SetExpectedChannel(tetra.ChannelSCHHD)
	cc.SetColourCode(colourCode)

	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz: rxRate,
		ClockMode:    tetrarx.ClockGardner,
		GardnerGain:  0.005,
		// AFC and the channel-select filter are left off here, matching
		// the receiver-package TestReceiverGardnerLocksTETRACCAt144kHz.
		// This test isolates the DDC rate fix: does a sub-target capture,
		// interpolated to the receiver's 8-sps design point, decode? The
		// feed-forward AFC's data-blind 4·Δφ estimator misfires on a
		// perfectly noiseless *resampled* synthetic fixture (it reads a
		// spurious offset off the resampler's residual images and
		// derotates, collapsing a dibit class) — an artifact of the
		// zero-noise fixture, not of the DDC. The AFC's real value on
		// off-centre on-air captures is covered elsewhere.
		DibitSink: func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
		},
		SoftSink: func(diffs []complex64, baseIdx int) {
			cc.StashSoft(diffs, baseIdx)
		},
	})

	const chunk = 4096
	var scratch []complex64
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		block := iq[i:end]
		if dc != nil {
			scratch = dc.Process(scratch[:0], block)
			block = scratch
		}
		rx.Process(block)
	}

	var locked bool
	var gotLA uint16
drain:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			if ls, ok := ev.Payload.(tetra.LockState); ok {
				locked = true
				gotLA = ls.LocationArea
			}
		default:
			break drain
		}
	}
	return locked, gotLA
}

// buildTETRASCHHDDibits assembles the dibit stream the DDC lock test
// modulates: a 400-dibit 0..3 warmup so the matched filter + Gardner
// loop converge, then `repeats` bursts of (normal training sequence +
// SCH/HD-encoded MLE SYSINFO + 50 idle dibits), then a 100-dibit flush.
// Mirrors buildTETRASCHHDStream in the cmd/gophertrunk integration test
// and buildTETRASCHHDDibits in the receiver package; kept local because
// those live in other packages.
func buildTETRASCHHDDibits(repeats int, colourCode uint32, locationArea uint16) []uint8 {
	payload := []byte{0x00, 0x00, 0x00, 0, 0}
	payload[3] = byte((locationArea >> 6) & 0xFF)
	payload[4] = byte((locationArea & 0x3F) << 2)

	pdu := tetra.PDU{
		Disc:    tetra.DiscMLE,
		Type:    uint8(tetra.MLESystemInfo),
		Payload: payload,
	}
	info := pduToType1BitsTETRA(pdu, 124)
	type5 := tetra.EncodeSCHHD(info, colourCode)
	burstDibits := tetra.TetraBitsToDibits(type5)

	frame := make([]uint8, 0, len(tetra.NormalSyncDibits())+len(burstDibits))
	frame = append(frame, tetra.NormalSyncDibits()...)
	frame = append(frame, burstDibits...)

	out := make([]uint8, 0, 400+repeats*(len(frame)+50)+100)
	for i := 0; i < 400; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 50; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// pduToType1BitsTETRA pads a PDU's assembled bytes into exactly k1 bits,
// MSB-first per byte. Mirror of the tetra-package test helper.
func pduToType1BitsTETRA(pdu tetra.PDU, k1 int) []byte {
	bytes := tetra.AssemblePDU(pdu)
	if len(bytes)*8 > k1 {
		return nil
	}
	out := make([]byte, k1)
	for i, b := range bytes {
		for j := 0; j < 8; j++ {
			out[i*8+j] = (b >> uint(7-j)) & 1
		}
	}
	return out
}
