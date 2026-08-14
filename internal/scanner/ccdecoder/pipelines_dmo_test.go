package ccdecoder

import (
	"io"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// tchSpeechFrameBits / tchType3Bits mirror the unexported tetra constants: a TCH/S
// speech frame is 137 type-1 bits, and the full slot is 432 type-4 (post-FEC, pre-
// scramble) bits.
const (
	dmoTestSpeechBits = 137
	dmoTestType4Bits  = 432
)

// Shared DMO test channel parameters: the production DDC target for
// ProtocolTETRADMO (ddcTargetForProtocol) and the resulting samples per symbol.
const (
	dmoTestSampleRate = 144_000.0
	dmoTestSPS        = 8 // 144000 / 18000
	dmoTestFreqHz     = 438_900_000
)

func dmoSeqBits(seed, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte((seed + i*5) % 2)
	}
	return out
}

func dmoFiller(seed, n int) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8((seed + i*3) & 3)
	}
	return out
}

// DMO slot geometry in dibits, from EN 300 396-2 Tables 15/16 (the same table the
// dmDSB*/dmDNB* offsets in tetra/dmo.go are derived from), expressed here as
// positions WITHIN a 255-dibit timeslot rather than relative to the training lead:
//
//	DSB: freq-corr 7..47, SCH/S 47..107, sync train 107..126 (lead 107), BKN2 126..234
//	DNB: BKN1 7..115, normal train 115..126 (lead 115), BKN2 126..234
//
// Both bursts occupy dibits 7..234 of their slot, leaving the real guard either side.
// A synthetic stream MUST be laid out this way: a real transmitter emits one burst per
// timeslot from one clock, so all its DNB leads share one residue mod 255 — the
// property tetra.DMSlotGrid uses to tell traffic from correlator noise.
const (
	dmoTestSlotDibits = 255
	dmoTestDSBLead    = 107
	dmoTestDNBLead    = 115
	dmoTestBurstStart = 7
)

// dmoSlot places a burst's fields into one 255-dibit timeslot, padding either side
// with filler so the emitted slot is exactly dmoTestSlotDibits long.
func dmoSlot(seed int, fields ...[]uint8) []uint8 {
	slot := dmoFiller(seed, dmoTestSlotDibits)
	at := dmoTestBurstStart
	for _, f := range fields {
		copy(slot[at:], f)
		at += len(f)
	}
	if at != 234 {
		panic("dmo test slot: burst ends at dibit " + string(rune('0'+at%10)) + ", want 234")
	}
	return slot
}

// buildDMODibitStream synthesizes a DMO transmission's demodulated dibit stream: a
// long receiver lead-in, one DSB (SCH/S + SCH/H), then nDNB TCH/S DNBs scrambled with
// colour — every burst on its own 255-dibit timeslot, as a real radio transmits them.
// It mirrors the buildDSB/buildDNB layout the tetra package tests use (EN 300 396-2
// Tables 15/16), built from the exported encoders so it can live in the ccdecoder
// package.
func buildDMODibitStream(colour uint32, nDNB int) []uint8 {
	b2d := tetra.TetraBitsToDibits
	schs := b2d(tetra.EncodeBSCH(dmoSeqBits(1, 60)))
	schh := b2d(tetra.EncodeSCHHD(dmoSeqBits(2, 124), 0))

	var out []uint8
	// Lead-in (a whole number of slots): let Gardner/AFC/equalizer converge.
	out = append(out, dmoFiller(0, 4*dmoTestSlotDibits)...)

	out = append(out, dmoSlot(1,
		dmoFiller(3, dmoTestDSBLead-dmoTestBurstStart-60), // 40-dibit freq-corr field
		schs,
		tetra.SyncTrainingDibits(),
		schh,
	)...)

	for i := 0; i < nDNB; i++ {
		frameA := dmoSeqBits(7+i, dmoTestSpeechBits)
		frameB := dmoSeqBits(9+i, dmoTestSpeechBits)
		type4 := framing.UnpackBitsMSB(tetra.EncodeTCHS(frameA, frameB), dmoTestType4Bits)
		onair := framing.ScrambleTetra(type4, colour)
		blk1 := b2d(onair[:dmoTestType4Bits/2])
		blk2 := b2d(onair[dmoTestType4Bits/2:])
		out = append(out, dmoSlot(11+i, blk1, tetra.NormalSyncDibits(), blk2)...)
	}
	// Trailing filler so the last burst is inside the extractor's window.
	out = append(out, dmoFiller(99, 3*dmoTestSlotDibits)...)
	return out
}

// TestTETRADMOPipelineLocksAndGrants drives newTETRADMOPipeline with a synthetic DMO
// transmission modulated to the 144 kHz channel rate (the production DDC target) and
// asserts the production control path: it locks on the DSB SCH/S (publishing
// cc.locked), brute-force-recovers the DM traffic colour code, and publishes a
// tetra-dmo grant so the composer starts a voice chain. This is the production-path
// analogue of the offline TestTETRADMOReplay — a green synthetic decode is a
// necessary (not sufficient) gate; on-air A/B on the operator's capture is still
// required to close #1003 (#764/#771).
func TestTETRADMOPipelineLocksAndGrants(t *testing.T) {
	const (
		sampleRate = 144_000.0 // ddcTargetForProtocol(ProtocolTETRADMO)
		sps        = 8         // 144000 / 18000
		span       = 8
		alpha      = 0.35
		freqHz     = 438_900_000
		testColour = uint32(3) // the DM traffic colour to recover (0 = config default)
		nDNB       = 40
	)

	dibits := buildDMODibitStream(testColour, nDNB)
	iq := demod.ModulatePiOver4DQPSK(dibits, sps, span, alpha, math.Pi/4)
	// Light noise floor: the blind CMA equalizer's constant-modulus adaptation is
	// well-defined only against a signal with a noise floor (mirrors the TMO
	// integration test + receiver clean-channel tests).
	iq = demod.ApplyImpairments(iq, sampleRate, demod.Impairments{SNRdB: 40, Seed: 7})

	bus := events.NewBus(256)
	defer bus.Close()
	sub := bus.Subscribe()
	var (
		mu     sync.Mutex
		locked bool
		grants []trunking.Grant
	)
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for ev := range sub.C {
			mu.Lock()
			switch ev.Kind {
			case events.KindCCLocked:
				locked = true
			case events.KindGrant:
				if g, ok := ev.Payload.(trunking.Grant); ok {
					grants = append(grants, g)
				}
			}
			mu.Unlock()
		}
	}()

	pl, err := newTETRADMOPipeline(PipelineOptions{
		Bus: bus,
		// Debug logger: the tch_crc telemetry decode this test asserts on is
		// debug-gated (it only feeds the debug status line in production).
		Log:          slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})),
		SystemName:   "DMO",
		FrequencyHz:  freqHz,
		SampleRateHz: sampleRate,
		System:       trunking.System{Protocol: trunking.ProtocolTETRADMO},
	})
	if err != nil {
		t.Fatalf("newTETRADMOPipeline: %v", err)
	}
	dmo := pl.(*tetraDMOPipeline)

	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		pl.Process(iq[i:end])
	}

	// Inspect the pipeline's own decode state (same package → unexported access).
	if !dmo.locked {
		t.Errorf("pipeline did not lock (dsb_total=%d dsb_crc=%d)", dmo.dsbTotal, dmo.dsbCRC)
	}
	if dmo.dsbCRC == 0 {
		t.Errorf("no CRC-valid DSB SCH/S decoded")
	}
	if !dmo.colourKnown {
		t.Errorf("DM colour code not recovered (dnb_total=%d tch_crc=%d)", dmo.dnbTotal, dmo.tchCRC)
	} else if dmo.colour != testColour {
		t.Errorf("recovered colour=%d, want %d", dmo.colour, testColour)
	}
	if dmo.tchCRC == 0 {
		t.Errorf("no CRC-valid TCH/S decoded at the recovered colour")
	}

	bus.Close()
	<-drainDone
	mu.Lock()
	defer mu.Unlock()
	if !locked {
		t.Errorf("no cc.locked event published")
	}
	if len(grants) == 0 {
		t.Fatalf("no tetra-dmo grant published")
	}
	g := grants[0]
	if g.Protocol != "tetra-dmo" {
		t.Errorf("grant protocol=%q, want tetra-dmo", g.Protocol)
	}
	if g.FrequencyHz != freqHz {
		t.Errorf("grant freq=%d, want %d", g.FrequencyHz, freqHz)
	}
}

// dmoTestEvents subscribes to bus and records the ordered lock/grant events the DMO
// control path publishes, so a test can assert BOTH how many grants fired and that
// none preceded the lock.
type dmoTestEvents struct {
	mu       sync.Mutex
	locked   bool
	grants   []trunking.Grant
	preLock  int // grants published while the pipeline had not yet locked
	drainEnd chan struct{}
}

func dmoWatch(bus *events.Bus) *dmoTestEvents {
	w := &dmoTestEvents{drainEnd: make(chan struct{})}
	sub := bus.Subscribe()
	go func() {
		defer close(w.drainEnd)
		for ev := range sub.C {
			w.mu.Lock()
			switch ev.Kind {
			case events.KindCCLocked:
				w.locked = true
			case events.KindGrant:
				if g, ok := ev.Payload.(trunking.Grant); ok {
					if !w.locked {
						w.preLock++
					}
					w.grants = append(w.grants, g)
				}
			}
			w.mu.Unlock()
		}
	}()
	return w
}

// dmoNoiseIQ returns seconds' worth of complex Gaussian noise at the DMO channel
// rate — an idle DMO frequency, which is what the channel looks like almost all the
// time (EN 300 396-2: silent between transmissions).
func dmoNoiseIQ(seconds float64, seed int64) []complex64 {
	rng := rand.New(rand.NewSource(seed))
	out := make([]complex64, int(seconds*dmoTestSampleRate))
	for i := range out {
		out[i] = complex(float32(rng.NormFloat64()*0.1), float32(rng.NormFloat64()*0.1))
	}
	return out
}

func dmoFeed(pl ProtocolPipeline, iq []complex64) {
	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		pl.Process(iq[i:end])
	}
}

// dmoTestPipeline builds a DMO pipeline whose clock the test drives, so a
// multi-second scenario runs in milliseconds and the 3 s grant re-arm is exercised
// deterministically rather than by sleeping.
func dmoTestPipeline(t *testing.T, bus *events.Bus, clock *time.Time) *tetraDMOPipeline {
	t.Helper()
	pl, err := newTETRADMOPipeline(PipelineOptions{
		Bus:          bus,
		SystemName:   "DMO",
		FrequencyHz:  dmoTestFreqHz,
		SampleRateHz: dmoTestSampleRate,
		System:       trunking.System{Protocol: trunking.ProtocolTETRADMO},
	})
	if err != nil {
		t.Fatalf("newTETRADMOPipeline: %v", err)
	}
	p := pl.(*tetraDMOPipeline)
	p.now = func() time.Time { return *clock }
	return p
}

// dmoModulate renders a dibit stream to IQ at the DMO channel rate with a light noise
// floor (the blind CMA equalizer needs one to be well defined).
func dmoModulate(dibits []uint8, seed int64) []complex64 {
	iq := demod.ModulatePiOver4DQPSK(dibits, dmoTestSPS, 8, 0.35, math.Pi/4)
	return demod.ApplyImpairments(iq, dmoTestSampleRate, demod.Impairments{SNRdB: 40, Seed: seed})
}

// TestTETRADMOPipelineIgnoresIdleChannel is the core regression for the reported
// "as soon as I start GopherTrunk it starts call recording and grants" symptom. The
// DNB correlator fires ~18 times a second on noise (see tetra/dmo_grid.go), and the
// grant path used to take that at face value: four detections — about 230 ms — were
// enough to publish a grant, open a voice chain and a recording session, on an idle
// channel with no lock. Ten simulated seconds of an idle DMO frequency must produce
// no lock, no grant, and no qualified DNB.
func TestTETRADMOPipelineIgnoresIdleChannel(t *testing.T) {
	bus := events.NewBus(256)
	defer bus.Close()
	w := dmoWatch(bus)
	clock := time.Unix(1_760_000_000, 0)
	p := dmoTestPipeline(t, bus, &clock)

	dmoFeed(p, dmoNoiseIQ(10, 424242))

	if p.dnbTotal == 0 {
		t.Fatalf("no raw DNB detections on noise — the test is not exercising the gate")
	}
	if p.dnbQualified != 0 {
		t.Errorf("qualified %d of %d noise DNB detections as traffic, want 0",
			p.dnbQualified, p.dnbTotal)
	}
	if p.grantActive {
		t.Errorf("grant fired on an idle channel (dnb_total=%d)", p.dnbTotal)
	}
	if p.locked {
		t.Errorf("locked on an idle channel (dsb_total=%d dsb_crc=%d)", p.dsbTotal, p.dsbCRC)
	}

	bus.Close()
	<-w.drainEnd
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.grants) != 0 {
		t.Errorf("published %d grants on an idle channel, want 0", len(w.grants))
	}
}

// TestTETRADMOPipelineGrantsOnlyAfterLock pins the ordering the grant guard now
// enforces: dnbSinceLock was named for a lock the guard never consulted, so a grant
// could precede cc.locked entirely.
//
// The idle-channel lead-in is what makes this bite, and it is what the operator
// actually had: the daemon starts on a silent DMO frequency, and four correlator
// false alarms — about 230 ms — were enough to grant long before any DSB appeared.
func TestTETRADMOPipelineGrantsOnlyAfterLock(t *testing.T) {
	bus := events.NewBus(256)
	defer bus.Close()
	w := dmoWatch(bus)
	clock := time.Unix(1_760_000_000, 0)
	p := dmoTestPipeline(t, bus, &clock)

	dmoFeed(p, dmoNoiseIQ(3, 5150)) // silent channel before anyone keys up
	if p.locked {
		t.Fatalf("locked on the idle lead-in")
	}
	dmoFeed(p, dmoModulate(buildDMODibitStream(3, 40), 7))

	bus.Close()
	<-w.drainEnd
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.locked {
		t.Fatalf("pipeline did not lock")
	}
	if len(w.grants) == 0 {
		t.Fatalf("no grant published for a real transmission")
	}
	if w.preLock != 0 {
		t.Errorf("%d grant(s) published before cc.locked", w.preLock)
	}
}

// buildDMOUndecodableDibitStream mirrors buildDMODibitStream — a lock-worthy DSB
// followed by slot-aligned, correctly-trained DNBs — but the DNB payload blocks are
// filler, not TCH/S codewords at any colour. The shape of an encrypted or
// unreceivably-weak transmission: the grid qualifies every burst, but
// RecoverDMColourCode's confidence gate can never clear.
// dmoRandDibits returns n pseudo-random dibits (LCG-mixed). The periodic
// dmoFiller pattern is fine as inter-burst guard, but as a 108-dibit payload it
// modulates to a strong tone that upsets timing recovery / the CMA equalizer
// and most bursts go undetected — random payload keeps the receiver honest.
func dmoRandDibits(seed, n int) []uint8 {
	s := uint32(seed)*2654435761 + 12345
	out := make([]uint8, n)
	for i := range out {
		s = s*1664525 + 1013904223
		out[i] = uint8(s >> 30)
	}
	return out
}

func buildDMOUndecodableDibitStream(nDNB int) []uint8 {
	b2d := tetra.TetraBitsToDibits
	schs := b2d(tetra.EncodeBSCH(dmoSeqBits(1, 60)))
	schh := b2d(tetra.EncodeSCHHD(dmoSeqBits(2, 124), 0))

	var out []uint8
	out = append(out, dmoFiller(0, 4*dmoTestSlotDibits)...)
	out = append(out, dmoSlot(1,
		dmoFiller(3, dmoTestDSBLead-dmoTestBurstStart-60),
		schs,
		tetra.SyncTrainingDibits(),
		schh,
	)...)
	for i := 0; i < nDNB; i++ {
		out = append(out, dmoSlot(11+i,
			dmoRandDibits(211+i*7, 108),
			tetra.NormalSyncDibits(),
			dmoRandDibits(509+i*13, 108),
		)...)
	}
	out = append(out, dmoFiller(99, 3*dmoTestSlotDibits)...)
	return out
}

// TestTETRADMOPipelineRecoversColourAfterRearm is the regression for the operator's
// run where colour_known stayed false for the daemon's whole life: colourTries was
// never reset, so six failed attempts on an early undecodable transmission disabled
// colour recovery permanently — hundreds of later, perfectly decodable qualified DNBs
// arrived with recovery already switched off. The attempt budget is per-transmission
// now: a re-arm (traffic drought) clears colourTries and the stale candidate buffer,
// so the next PTT recovers. Fails against the old code (colour never recovered on the
// second transmission).
func TestTETRADMOPipelineRecoversColourAfterRearm(t *testing.T) {
	bus := events.NewBus(256)
	defer bus.Close()
	w := dmoWatch(bus)
	clock := time.Unix(1_760_000_000, 0)
	p := dmoTestPipeline(t, bus, &clock)

	// First transmission: undecodable payload, long enough (~90 qualified DNBs) to
	// burn all dmoColourMaxAttempts recovery attempts.
	dmoFeed(p, dmoModulate(buildDMOUndecodableDibitStream(90), 7))
	if p.colourKnown {
		t.Fatalf("recovered a colour from undecodable payload (colour=%d)", p.colour)
	}
	if p.colourTries < dmoColourMaxAttempts {
		t.Fatalf("burned %d attempts, want %d — the scenario is not exercising the cap",
			p.colourTries, dmoColourMaxAttempts)
	}

	// Silence past the re-arm drought: the attempt budget must reset with the grid.
	gap := dmoNoiseIQ(1, 99)
	for i := 0; i < 5; i++ {
		clock = clock.Add(time.Second)
		dmoFeed(p, gap)
	}
	if p.colourTries != 0 {
		t.Errorf("colourTries = %d after the re-arm drought, want 0", p.colourTries)
	}
	if len(p.colourCand) != 0 {
		t.Errorf("%d stale colour candidates survived the re-arm, want 0", len(p.colourCand))
	}

	// Second transmission: clean colour-3 traffic must now recover.
	clock = clock.Add(time.Second)
	dmoFeed(p, dmoModulate(buildDMODibitStream(3, 40), 11))
	if !p.colourKnown {
		t.Fatalf("colour not recovered on a clean transmission after re-arm (tries=%d, qualified=%d)",
			p.colourTries, p.dnbQualified)
	}
	if p.colour != 3 {
		t.Errorf("recovered colour=%d, want 3", p.colour)
	}

	bus.Close()
	<-w.drainEnd
}

// TestTETRADMOPipelinePublishesColourToSink pins the colour hand-off to the decoder:
// the pipeline must call TETRADMOColourSink when recovery lands (and immediately for
// a tetra_colour_code override), because the same-carrier voice chain's grant
// structurally fires before recovery completes and polls the decoder for the answer.
func TestTETRADMOPipelinePublishesColourToSink(t *testing.T) {
	bus := events.NewBus(256)
	defer bus.Close()
	w := dmoWatch(bus)

	var got []uint32
	pl, err := newTETRADMOPipeline(PipelineOptions{
		Bus:                bus,
		SystemName:         "DMO",
		FrequencyHz:        dmoTestFreqHz,
		SampleRateHz:       dmoTestSampleRate,
		System:             trunking.System{Protocol: trunking.ProtocolTETRADMO},
		TETRADMOColourSink: func(c uint32) { got = append(got, c) },
	})
	if err != nil {
		t.Fatalf("newTETRADMOPipeline: %v", err)
	}
	dmoFeed(pl, dmoModulate(buildDMODibitStream(3, 40), 7))
	if len(got) != 1 || got[0] != 3 {
		t.Errorf("colour sink calls = %v, want exactly [3]", got)
	}

	// A configured colour publishes at construction, before any IQ.
	got = nil
	_, err = newTETRADMOPipeline(PipelineOptions{
		Bus:                bus,
		SystemName:         "DMO",
		FrequencyHz:        dmoTestFreqHz,
		SampleRateHz:       dmoTestSampleRate,
		System:             trunking.System{Protocol: trunking.ProtocolTETRADMO, TETRAColourCode: 44},
		TETRADMOColourSink: func(c uint32) { got = append(got, c) },
	})
	if err != nil {
		t.Fatalf("newTETRADMOPipeline (override): %v", err)
	}
	if len(got) != 1 || got[0] != 44 {
		t.Errorf("override colour sink calls = %v, want exactly [44]", got)
	}

	bus.Close()
	<-w.drainEnd
}

// TestDecoderTETRADMOColourRoundTrip pins the decoder-side store the sink writes
// into: unknown until stored (distinguishing "colour 0" from "nothing known"), and
// cleared when the active pipeline is torn down so one system's colour cannot leak
// into the next acquisition.
func TestDecoderTETRADMOColourRoundTrip(t *testing.T) {
	d := &Decoder{}
	if c, known := d.TETRADMOColour(); known || c != 0 {
		t.Fatalf("fresh decoder colour = (%d, %v), want (0, false)", c, known)
	}
	d.dmoColour.Store(0<<1 | 1) // a recovered colour of literally 0 is still known
	if c, known := d.TETRADMOColour(); !known || c != 0 {
		t.Fatalf("stored colour 0 reads (%d, %v), want (0, true)", c, known)
	}
	d.dmoColour.Store(44<<1 | 1)
	if c, known := d.TETRADMOColour(); !known || c != 44 {
		t.Fatalf("stored colour 44 reads (%d, %v), want (44, true)", c, known)
	}
	d.clearActiveLocked()
	if c, known := d.TETRADMOColour(); known || c != 0 {
		t.Fatalf("colour after clearActiveLocked = (%d, %v), want (0, false)", c, known)
	}
}

// TestTETRADMOPipelineRearmsBetweenTransmissions is the regression for the second
// half of the report: after the bogus startup grant latched, the operator's real
// 10 s PTT never granted and never recorded. The re-arm needed a DNB drought that the
// ~18/s noise detections made impossible, and it was only ever evaluated when a burst
// arrived — so on a genuinely silent channel it could not fire at all. A second
// transmission after a silent gap must grant again.
func TestTETRADMOPipelineRearmsBetweenTransmissions(t *testing.T) {
	bus := events.NewBus(256)
	defer bus.Close()
	w := dmoWatch(bus)
	clock := time.Unix(1_760_000_000, 0)
	p := dmoTestPipeline(t, bus, &clock)

	tx := dmoModulate(buildDMODibitStream(3, 40), 7)
	gap := dmoNoiseIQ(1, 99) // idle channel, still raining false DNB detections

	dmoFeed(p, tx)
	afterFirst := p.dnbQualified
	w.mu.Lock()
	first := len(w.grants)
	w.mu.Unlock()
	if first != 1 {
		t.Fatalf("first transmission published %d grants, want exactly 1", first)
	}

	// Silence long enough to clear the grant edge (the clock is ours to advance).
	for i := 0; i < 5; i++ {
		clock = clock.Add(time.Second)
		dmoFeed(p, gap)
	}
	if p.grantActive {
		t.Errorf("grant still armed after %v of silence", 5*time.Second)
	}

	clock = clock.Add(time.Second)
	dmoFeed(p, dmoModulate(buildDMODibitStream(3, 40), 11))

	if p.dnbQualified <= afterFirst {
		t.Errorf("second transmission qualified no DNBs (%d → %d)", afterFirst, p.dnbQualified)
	}
	bus.Close()
	<-w.drainEnd
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.grants) != 2 {
		t.Errorf("published %d grants across two transmissions, want 2", len(w.grants))
	}
}
