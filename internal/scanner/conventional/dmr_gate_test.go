package conventional

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// Fixtures for the DMR call gate (issue #1261): a base-station repeater
// carrier, two timeslots interleaved (132-dibit burst + 12-dibit CACH per
// 30 ms slot), modulated as DMR 4-FSK at 48 kHz / 10 sps.

const dmrGateTestRate = 48_000

func dmrRandDibits(rng *rand.Rand, n int) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8(rng.Intn(4))
	}
	return out
}

// dmrDataBurst is a BS data burst with slot type dt; the payload is random
// (the gate reads only the sync and the slot type).
func dmrDataBurst(rng *rand.Rand, dt dmr.DataType) []uint8 {
	slot := framing.BitsToDibits(dmr.AssembleSlotType(dmr.SlotType{ColorCode: 1, DataType: dt}))
	b := make([]uint8, 0, dmr.BurstDibits)
	b = append(b, dmrRandDibits(rng, dmr.HalfPayloadDibits)...)
	b = append(b, slot[:dmr.SlotTypeDibits]...)
	b = append(b, dmr.BSData.Dibits[:]...)
	b = append(b, slot[dmr.SlotTypeDibits:]...)
	b = append(b, dmrRandDibits(rng, dmr.HalfPayloadDibits)...)
	return b
}

// dmrVoiceBurst is a voice burst: burst A of a superframe carries the BS voice
// sync, bursts B-F carry embedded signalling (random here) in its place.
func dmrVoiceBurst(rng *rand.Rand, withSync bool) []uint8 {
	b := make([]uint8, 0, dmr.BurstDibits)
	b = append(b, dmrRandDibits(rng, 54)...)
	if withSync {
		b = append(b, dmr.BSVoice.Dibits[:]...)
	} else {
		b = append(b, dmrRandDibits(rng, dmr.SyncDibits)...)
	}
	b = append(b, dmrRandDibits(rng, 54)...)
	return b
}

// dmrRepeaterDibits lays out slots on the repeater carrier: ts1[i] and ts2[i]
// are bursts of consecutive frames, alternating TS1/TS2, each preceded by a
// 12-dibit CACH.
func dmrRepeaterDibits(rng *rand.Rand, ts1, ts2 [][]uint8) []uint8 {
	var out []uint8
	for i := range ts1 {
		out = append(out, dmrRandDibits(rng, 12)...)
		out = append(out, ts1[i]...)
		out = append(out, dmrRandDibits(rng, 12)...)
		out = append(out, ts2[i]...)
	}
	return out
}

// dmrBeaconBursts is n frames of an idle repeater: Idle data bursts in both
// slots.
func dmrBeaconBursts(rng *rand.Rand, n int) (ts1, ts2 [][]uint8) {
	for i := 0; i < n; i++ {
		ts1 = append(ts1, dmrDataBurst(rng, dmr.DTIdle))
		ts2 = append(ts2, dmrDataBurst(rng, dmr.DTIdle))
	}
	return ts1, ts2
}

// dmrCallBursts is a call on TS1 (3 Voice LC Headers, then `superframes`
// voice superframes A-F) with TS2 idle.
func dmrCallBursts(rng *rand.Rand, superframes int) (ts1, ts2 [][]uint8) {
	for i := 0; i < 3; i++ {
		ts1 = append(ts1, dmrDataBurst(rng, dmr.DTVoiceLCHeader))
	}
	for s := 0; s < superframes; s++ {
		for k := 0; k < 6; k++ {
			ts1 = append(ts1, dmrVoiceBurst(rng, k == 0))
		}
	}
	for range ts1 {
		ts2 = append(ts2, dmrDataBurst(rng, dmr.DTIdle))
	}
	return ts1, ts2
}

func invertDibits(d []uint8) []uint8 {
	out := make([]uint8, len(d))
	for i, v := range d {
		out[i] = (v + dmr.PolarityFlip) & 3
	}
	return out
}

func modulateDMR(rng *rand.Rand, dibits []uint8) []complex64 {
	iq := demod.ModulateC4FM(dibits, 10, 8, 0.20, dmrGateTestRate, 1944.0)
	for i := range iq {
		iq[i] = iq[i]*0.5 + complex(float32(rng.NormFloat64()*0.01), float32(rng.NormFloat64()*0.01))
	}
	return iq
}

// gateTrace feeds iq in 1024-sample chunks and returns, per chunk, whether
// the gate was open.
func gateTrace(d *DMRCallDetector, iq []complex64) []bool {
	var out []bool
	for i := 0; i < len(iq); i += 1024 {
		out = append(out, d.Process(iq[i:min(i+1024, len(iq))]))
	}
	return out
}

func anyOpen(tr []bool) bool {
	for _, v := range tr {
		if v {
			return true
		}
	}
	return false
}

// A repeater's idle beacon is a carrier with no call on it: the gate must
// never open, whatever the discriminator polarity (an inverted front end
// turns every Idle data sync into a voice sync).
func TestDMRCallDetectorIgnoresIdleBeacon(t *testing.T) {
	for _, inverted := range []bool{false, true} {
		rng := rand.New(rand.NewSource(1))
		ts1, ts2 := dmrBeaconBursts(rng, 80) // 2.4 s
		dib := dmrRepeaterDibits(rng, ts1, ts2)
		if inverted {
			dib = invertDibits(dib)
		}
		d := NewDMRCallDetector(dmrGateTestRate)
		if anyOpen(gateTrace(d, modulateDMR(rng, dib))) {
			t.Errorf("inverted=%v: DMR gate opened on an idle beacon", inverted)
		}
		if d.polarity < 0 {
			t.Errorf("inverted=%v: the beacon's clean slot types should have fixed the polarity", inverted)
		}
	}
}

// A call (Voice LC Headers then voice superframes) opens the gate, at either
// polarity, and the gate closes again once the call is over and only the
// beacon remains.
func TestDMRCallDetectorOpensOnCallAndClosesAfter(t *testing.T) {
	for _, inverted := range []bool{false, true} {
		rng := rand.New(rand.NewSource(2))
		b1, b2 := dmrBeaconBursts(rng, 20)
		c1, c2 := dmrCallBursts(rng, 8) // ~3 s of voice
		a1, a2 := dmrBeaconBursts(rng, 80)
		lead := dmrRepeaterDibits(rng, b1, b2)
		call := dmrRepeaterDibits(rng, c1, c2)
		tail := dmrRepeaterDibits(rng, a1, a2)
		dib := append(append(append([]uint8{}, lead...), call...), tail...)
		if inverted {
			dib = invertDibits(dib)
		}
		iq := modulateDMR(rng, dib)
		tr := gateTrace(NewDMRCallDetector(dmrGateTestRate), iq)

		chunkOf := func(dibitIdx int) int { return dibitIdx * 10 / 1024 }
		leadEnd, callEnd := chunkOf(len(lead)), chunkOf(len(lead)+len(call))
		if anyOpen(tr[:leadEnd]) {
			t.Errorf("inverted=%v: gate open during the beacon before the call", inverted)
		}
		open := 0
		for _, v := range tr[leadEnd:callEnd] {
			if v {
				open++
			}
		}
		if frac := float64(open) / float64(callEnd-leadEnd); frac < 0.8 {
			t.Errorf("inverted=%v: gate open for %.0f%% of the call, want >= 80%%", inverted, 100*frac)
		}
		// dmrCallHold (1 s) after the call ends, the beacon alone must not hold it.
		closeBy := callEnd + 57 // 1.2 s of 1024-sample chunks at 48 kHz
		if anyOpen(tr[closeBy:]) {
			t.Errorf("inverted=%v: gate still open on the beacon after the call ended", inverted)
		}
	}
}

// dmrAt2400k interpolates a 48 kHz fixture to the scanner's SDR rate.
func dmrAt2400k(iq []complex64) []complex64 {
	return dsp.NewResampler(50, 1, 16, 8.6).Process(nil, iq)
}

// runDMRChannel scans one DMR channel carrying iq at the scanner's SDR rate
// and returns the engine fake after the stream (plus silence) has played.
func runDMRChannel(t *testing.T, iq []complex64) *fakeEngine {
	t.Helper()
	const freq = 442_387_500
	tuner := &fakeTuner{}
	eng := &fakeEngine{}
	src := &fakeIQ{chunks: map[uint32][][]complex64{freq: chunked(iq, 16384)}, tuner: tuner}
	s, err := New(Options{
		Tuner: tuner, IQ: src, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-DMR-GATE",
		Channels: []Channel{{
			Label: "Rptr", FrequencyHz: freq, Mode: ModeDMR,
			SquelchDbFS: -50, Hangtime: 300 * time.Millisecond,
		}},
		SampleRateHz: amTestRate,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Generous caps (the loop exits once the source drains): the call gate
	// runs a whole DMR receiver, far slower than real time under -race.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	go func() { _ = s.Run(ctx) }()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		src.mu.Lock()
		left := len(src.chunks[freq])
		src.mu.Unlock()
		if left == 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	cancel()
	return eng
}

// TestScannerDMRIdleBeaconOpensNoCall is the issue #1261 field report: "the
// scanner stays on the channel while the beacon is up". A repeater keyed with
// its idle beacon is a strong carrier, and on power alone the DMR channel
// opened a call that recorded nothing. With the call gate it must not.
func TestScannerDMRIdleBeaconOpensNoCall(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	b1, b2 := dmrBeaconBursts(rng, 100) // 3 s of beacon
	iq := dmrAt2400k(modulateDMR(rng, dmrRepeaterDibits(rng, b1, b2)))
	for i := range iq {
		iq[i] *= 0.2
	}
	if eng := runDMRChannel(t, iq); eng.startCount() != 0 {
		t.Fatalf("an idle DMR beacon opened %d call(s); the scanner must move on", eng.startCount())
	}
}

// TestScannerDMRCallStillOpens: the gate must not cost a real call.
func TestScannerDMRCallStillOpens(t *testing.T) {
	rng := rand.New(rand.NewSource(6))
	b1, b2 := dmrBeaconBursts(rng, 10)
	c1, c2 := dmrCallBursts(rng, 6)
	dib := append(dmrRepeaterDibits(rng, b1, b2), dmrRepeaterDibits(rng, c1, c2)...)
	iq := dmrAt2400k(modulateDMR(rng, dib))
	for i := range iq {
		iq[i] *= 0.2
	}
	if eng := runDMRChannel(t, iq); eng.startCount() != 1 {
		t.Fatalf("a DMR call opened %d call(s), want 1", eng.startCount())
	}
}

// TestDMRCallDetectorAcquiresAtAnySymbolPhase: the scanner tunes onto a
// repeater carrier that is already up, at an arbitrary sub-symbol phase. The
// receiver's timing loop pulls a bad phase in only slowly (measured: a
// 2-sample offset decoded no sync at all for a whole second), so the gate
// arms feed-forward timing acquisition on every start. A call must open the
// gate whatever the phase.
func TestDMRCallDetectorAcquiresAtAnySymbolPhase(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	c1, c2 := dmrCallBursts(rng, 4) // 1.6 s, no beacon lead-in
	base := modulateDMR(rng, dmrRepeaterDibits(rng, c1, c2))
	for phase := 0; phase < 10; phase++ {
		iq := append(make([]complex64, phase), base...)
		if !anyOpen(gateTrace(NewDMRCallDetector(dmrGateTestRate), iq)) {
			t.Errorf("symbol phase %d/10: a DMR call never opened the gate", phase)
		}
	}
}

// dmDirectModeIQ is a simplex (direct-mode) DMR transmission as a handheld
// sends it: one 132-dibit burst per 60 ms frame (288 dibits), the carrier OFF
// for the rest, framed by the DM syncs. 3 Voice LC Headers, then `superframes`
// voice superframes; `leadFrames` frames of silence (receiver noise) first.
func dmDirectModeIQ(rng *rand.Rand, leadFrames, superframes int) []complex64 {
	const frame, sps = 288, 10
	var bursts [][]uint8
	dmHeader := func() []uint8 {
		b := dmrDataBurst(rng, dmr.DTVoiceLCHeader)
		copy(b[dmr.HalfPayloadDibits+dmr.SlotTypeDibits:], dmr.DMData1.Dibits[:])
		return b
	}
	for i := 0; i < 3; i++ {
		bursts = append(bursts, dmHeader())
	}
	for s := 0; s < superframes; s++ {
		for k := 0; k < 6; k++ {
			b := dmrVoiceBurst(rng, false)
			if k == 0 {
				copy(b[54:], dmr.DMVoice1.Dibits[:])
			}
			bursts = append(bursts, b)
		}
	}
	total := leadFrames + len(bursts)
	dib := make([]uint8, 0, total*frame)
	for f := 0; f < total; f++ {
		if f >= leadFrames {
			dib = append(dib, bursts[f-leadFrames]...)
		} else {
			dib = append(dib, dmrRandDibits(rng, dmr.BurstDibits)...)
		}
		dib = append(dib, dmrRandDibits(rng, frame-dmr.BurstDibits)...)
	}
	const span = 8
	iq := demod.ModulateC4FM(dib, sps, span, 0.20, dmrGateTestRate, 1944.0)
	delay := span * sps
	for f := 0; f < total; f++ {
		on := f >= leadFrames
		base := f*frame*sps + delay
		for i := -delay; i < frame*sps-delay; i++ {
			j := base + i
			if j < 0 || j >= len(iq) {
				continue
			}
			if !(on && i >= -sps && i < (dmr.BurstDibits+1)*sps) {
				iq[j] = 0
			}
		}
	}
	for i := range iq {
		iq[i] = iq[i]*0.5 + complex(float32(rng.NormFloat64()*0.01), float32(rng.NormFloat64()*0.01))
	}
	return iq
}

// The #1261 reporter's channel is a simplex handheld: carrier off in every
// TDMA gap, DM syncs. The gate must open on its call (and stay shut on the
// silence before it).
func TestDMRCallDetectorOpensOnDirectModeCall(t *testing.T) {
	for _, seed := range []int64{8, 9, 10} {
		rng := rand.New(rand.NewSource(seed))
		lead := 10
		iq := dmDirectModeIQ(rng, lead, 4)
		tr := gateTrace(NewDMRCallDetector(dmrGateTestRate), iq)
		leadEnd := lead * 288 * 10 / 1024
		if anyOpen(tr[:leadEnd]) {
			t.Errorf("seed %d: gate open on the silence before the call", seed)
		}
		if !anyOpen(tr[leadEnd:]) {
			t.Errorf("seed %d: a direct-mode DMR call never opened the gate", seed)
		}
	}
}
